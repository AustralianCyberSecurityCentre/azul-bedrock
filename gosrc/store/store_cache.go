package store

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"time"

	st "github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/settings"
	"github.com/allegro/bigcache/v3"
	"github.com/prometheus/client_golang/prometheus"
)

/* Adds memory cache on to another store type. */

/* !! Bigcache Set function limitation !!
The bigcache set function will create a buffer equal to the size of the file you attempt
to insert into the cache, even if that file is much larger than the shard size.
And this is done per shard.
This means that whatever size files you let into bigcache will increase RAM consumption by
max_file_size_allowed * number_of_shards.

so if you allow 50MB files in and 32 shards it means 1.6GiB of RAM.
*/
// Maximum size of a file that can be in a shard as a fraction. e.g 0.5 == 50% of the maximum shard size.
const max_fractional_usage_for_one_file_in_a_shard = 0.5

type StoreCacheMetricCollectors struct {
	CacheLookup                  prometheus.Counter
	CacheHits                    prometheus.Counter
	PromStreamsOperationDuration *prometheus.HistogramVec
}

type StoreCache struct {
	cache             *bigcache.BigCache
	store             FileStorage
	maxFileSetToShard int64
	promMetrics       StoreCacheMetricCollectors // For collection of metrics on storage options
}

func NewDataCache(maxsize, ttl, shards int, store FileStorage, metricCollector StoreCacheMetricCollectors) (FileStorage, error) {
	config := bigcache.DefaultConfig(time.Duration(ttl) * time.Second)
	config.HardMaxCacheSize = maxsize
	config.Verbose = false
	config.Shards = shards
	ctx := context.Background()
	cache, err := bigcache.New(ctx, config)
	if err != nil {
		return &StoreCache{}, err
	}
	// This is done due to the bigcache limitation (refer to top comment on bigcache)
	shardSizeBytes := float32(config.HardMaxCacheSize*1024*1024) / float32(config.Shards)
	maxFileSetToShard := int64(max_fractional_usage_for_one_file_in_a_shard * shardSizeBytes)
	return &StoreCache{cache, store, maxFileSetToShard, metricCollector}, nil
}

func (c *StoreCache) Put(source, label, id string, data io.ReadCloser, fileSize int64) error {
	err := c.store.Put(source, label, id, data, fileSize)
	if err == nil {
		// Only cache if file is small enough - This is done due to the bigcache limitation (refer to top comment on bigcache)
		if fileSize > c.maxFileSetToShard {
			return nil
		}
		// Don't cache the file because it's more time consuming than just waiting for a miss on the get request.
	}
	return err
}

func (c *StoreCache) Copy(sourceOld, labelOld, idOld, sourceNew, labelNew, idNew string) error {
	err := c.store.Copy(sourceOld, labelOld, idOld, sourceNew, labelNew, idNew)
	if err != nil {
		return fmt.Errorf("error while copying object %v", err)
	}
	return nil
}

func (c *StoreCache) Fetch(source, label, id string, opts ...FileStorageFetchOption) (DataSlice, error) {
	// should we always pull full object for caching or just skip if not full size requested?
	var err error
	startTimeA := time.Now().UnixNano()
	cacheDataSlice := NewDataSlice()
	ent, err := c.cache.Get(id)
	if c.promMetrics.CacheLookup != nil {
		c.promMetrics.CacheLookup.Inc()
	}

	if err == nil && len(ent) > 0 {
		defer func() {
			reportStreamsOpMetric(c.promMetrics.PromStreamsOperationDuration, startTimeA, "cache-fetch", err)
		}()
		if c.promMetrics.CacheHits != nil {
			c.promMetrics.CacheHits.Inc()
		}
	} else {
		defer func() {
			reportStreamsOpMetric(c.promMetrics.PromStreamsOperationDuration, startTimeA, "cache-fetch-miss", err)
		}()
		ds, err := c.store.Fetch(source, label, id, WithOffsetAndSize(0, -1))
		if err != nil {
			return ds, err
		}
		defer ds.DataReader.Close()
		// If the file is too large for the cache return it immediately. - This is done due to the bigcache limitation (refer to top comment on bigcache)
		if int64(ds.Size) > c.maxFileSetToShard {
			return c.store.Fetch(source, label, id, opts...)
		}
		// Cache the file
		buf := make([]byte, ds.Size)
		// This will empty the reader but we set it again before we return it.
		_, err = ds.DataReader.Read(buf)

		ent = buf
		if err != nil && err != io.EOF {
			return ds, err
		}
		id := fmt.Sprintf("%x", sha256.Sum256(ent))
		err = c.cache.Set(id, ent)
		// failing to cache is not terminal, just log
		if err != nil {
			log.Printf("Ignoring data cache fail for %s size: %d err: %s\n", id, len(ent), err.Error())
		}
	}
	fetchOpt := NewFileStorageFetchOptions(opts...)
	// -ve is read all
	if fetchOpt.Size <= 0 {
		fetchOpt.Size = int64(len(ent))
	}
	// treat -ve as relative to end
	if fetchOpt.Offset < 0 {
		fetchOpt.Offset = int64(len(ent)) + fetchOpt.Offset
	}
	// still -ve (-ve offset was bigger than file)
	if fetchOpt.Offset < 0 {
		fetchOpt.Offset = 0
	}
	// offset after end of file
	if fetchOpt.Offset > 0 && fetchOpt.Offset >= int64(len(ent)) {
		return NewDataSlice(), fmt.Errorf("%w", &OffsetAfterEnd{msg: fmt.Sprintf("offset after EOF: %d", len(ent))})
	}

	// requested more than available.. be lenient and adjust
	if fetchOpt.Offset+fetchOpt.Size > int64(len(ent)) {
		fetchOpt.Size = int64(len(ent)) - fetchOpt.Offset
	}
	cacheDataSlice.Avail = int64(len(ent))
	cacheDataSlice.Start = fetchOpt.Offset
	cacheDataSlice.Size = fetchOpt.Size
	cacheDataSlice.DataReader = io.NopCloser(bytes.NewReader(ent[fetchOpt.Offset : fetchOpt.Offset+fetchOpt.Size]))
	return cacheDataSlice, nil
}

func (c *StoreCache) Exists(source, label, id string) (bool, error) {
	if c.promMetrics.CacheLookup != nil {
		c.promMetrics.CacheLookup.Inc()
	}
	if _, err := c.cache.Get(id); err == nil {
		if c.promMetrics.CacheHits != nil {
			c.promMetrics.CacheHits.Inc()
		}
		return true, nil
	}
	return c.store.Exists(source, label, id)
}

func (c *StoreCache) Delete(source, label, id string, opts ...FileStorageDeleteOption) (bool, error) {
	if _, err := c.cache.Get(id); err == nil {
		err = c.cache.Delete(id)
		if err != nil {
			st.Logger.Warn().Err(err).Msgf("Failed to delete id %s from cache and it should have been there", id)
		}

	}

	return c.store.Delete(source, label, id, opts...)
}

func (c *StoreCache) List(ctx context.Context, prefix string, startAfter string) <-chan FileStorageObjectListInfo {
	return c.store.List(ctx, prefix, startAfter)
}

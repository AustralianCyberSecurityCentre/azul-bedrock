package store

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/models"
	st "github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/settings"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/prometheus/client_golang/prometheus"
)

/* Store files via s3 provider. */
type StoreS3 struct {
	client                       *minio.Client
	bucket                       string
	promStreamsOperationDuration *prometheus.HistogramVec // For collection of metrics on storage options
}

type AutomaticAgeOffSettings struct {
	// Create ageoff rules in S3 that removes data older than the age-off of a source (based on last modified dates).
	EnableAutomaticAgeOff bool
	// Remove any rules that were created by the automatic ageoff policies.
	EnableCleanupAutoAgeOff bool
	// Configuration of Azul sources used to determine what ageoff should be set to.
	SourceConf *models.SourcesConf
}

func getS3DefaultTransport() *http.Transport {
	// default transport with response header timeout set to a minute (which doesn't appear to be default?)
	// from https://github.com/minio/minio-go/blob/master/transport.go
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          256,
		MaxIdleConnsPerHost:   16,
		ResponseHeaderTimeout: time.Minute,
		IdleConnTimeout:       time.Minute,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 10 * time.Second,
		// Set this value so that the underlying transport round-tripper
		// doesn't try to auto decode the body of objects with
		// content-encoding set to `gzip`.
		//
		// Refer:
		//    https://golang.org/src/net/http/transport.go?h=roundTrip#L1843
		DisableCompression: true,
	}
}

func handleAutoAgeoff(client *minio.Client, bucket string, autoAgeoffSettings AutomaticAgeOffSettings) error {
	var err error
	err = nil
	if autoAgeoffSettings.SourceConf == nil {
		return err
	}

	if autoAgeoffSettings.EnableAutomaticAgeOff {
		err = setLifecycleForBucket(client, bucket, autoAgeoffSettings.SourceConf)
	} else if autoAgeoffSettings.EnableCleanupAutoAgeOff {
		err = removeLifeCycleForBucket(client, bucket, autoAgeoffSettings.SourceConf)
		if err != nil {
			st.Logger.Warn().Err(err).Msg("Unable to remove old lifecycle policy if there was any.")
		}
	}

	return err
}

/** Creates a new S3 store with static credentials. */
func NewS3Store(endpoint string, accessKey string, secretKey string, secure bool, bucket string, region string, promStreamsOperationDuration *prometheus.HistogramVec, autoAgeoffSettings AutomaticAgeOffSettings) (FileStorage, error) {
	var client *minio.Client
	var err error
	opts := minio.Options{
		Secure:    secure,
		Region:    region,
		Creds:     credentials.NewStaticV4(accessKey, secretKey, ""),
		Transport: getS3DefaultTransport(),
	}
	// accessKey, secretKey
	client, err = minio.New(endpoint, &opts)
	if err != nil {
		return nil, err
	}
	b, err := client.BucketExists(context.Background(), bucket)
	if err != nil {
		return nil, err
	}
	if !b {
		err = client.MakeBucket(context.Background(), bucket, minio.MakeBucketOptions{})
	}
	if err != nil {
		return nil, err
	}

	err = handleAutoAgeoff(client, bucket, autoAgeoffSettings)
	if err != nil {
		return nil, err
	}

	return &StoreS3{
		client,
		bucket,
		promStreamsOperationDuration,
	}, err
}

/** Creates a new S3 store using IAM credentials. */
func NewS3StoreIAM(endpoint string, secure bool, bucket string, region string, promStreamsOperationDuration *prometheus.HistogramVec, autoAgeoffSettings AutomaticAgeOffSettings) (FileStorage, error) {
	var client *minio.Client
	var err error
	opts := minio.Options{
		Secure:    secure,
		Region:    region,
		Creds:     credentials.NewIAM(""),
		Transport: getS3DefaultTransport(),
	}

	client, err = minio.New(endpoint, &opts)
	if err != nil {
		return nil, err
	}
	b, err := client.BucketExists(context.Background(), bucket)
	if err != nil {
		return nil, err
	}
	if !b {
		err = client.MakeBucket(context.Background(), bucket, minio.MakeBucketOptions{})
	}
	if err != nil {
		return nil, err
	}

	err = handleAutoAgeoff(client, bucket, autoAgeoffSettings)
	if err != nil {
		return nil, err
	}

	return &StoreS3{
		client,
		bucket,
		promStreamsOperationDuration,
	}, err
}

func (s *StoreS3) Put(source, label, id string, data io.ReadCloser, fileSize int64) error {
	var err error
	startTime := time.Now().UnixNano()
	defer func() {
		reportStreamsOpMetric(s.promStreamsOperationDuration, startTime, "put", err)
	}()
	key := createIdPath(source, label, id)

	var bufRead io.Reader
	// If the file size is not known read the whole file into memory to get the file size, otherwise this S3 library leaks memory.
	// The leak is it failing to release the memory it uses when uploading a file to minio.
	if fileSize == -1 {
		rawData, err := io.ReadAll(data)
		if err != nil {
			return err
		}
		fileSize = int64(len(rawData))
		bufRead = bytes.NewReader(rawData)
	} else {
		bufRead = bufio.NewReaderSize(data, MAX_BUFFERED_READER_BYTES)
	}

	options := minio.PutObjectOptions{ContentType: "binary/octet-stream"}
	// If file is large enough use concurrency.
	if fileSize > MAX_FILE_BYTES_BEFORE_CONCURRENT_UPLOAD {
		options.NumThreads = uint(NUM_CONCURRENT_UPLOAD_THREADS)
		options.ConcurrentStreamParts = true
		options.PartSize = uint64(CONCURRENT_BUFFER_SIZE_BYTES)
	}
	_, err = s.client.PutObject(
		context.Background(),
		s.bucket,
		key,
		bufRead,
		fileSize, // set to -1 unless size is known
		options,
	)
	if err != nil {
		return err
	}
	return nil
}

func (s *StoreS3) Fetch(source, label, id string, opts ...FileStorageFetchOption) (DataSlice, error) {
	var err error
	startTime := time.Now().UnixNano()
	defer func() {
		reportStreamsOpMetric(s.promStreamsOperationDuration, startTime, "fetch", err)
	}()
	key := createIdPath(source, label, id)
	empty := NewDataSlice()
	reader, err := s.client.GetObject(context.Background(), s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		resp := minio.ToErrorResponse(err)
		code := resp.Code
		if code == "NoSuchKey" || code == "NoSuchBucket" {
			return empty, fmt.Errorf("%w", &NotFoundError{})
		}
		return empty, fmt.Errorf("%w", &AccessError{msg: fmt.Sprintf("%v", code)})
	}

	// Custom logic to ensure all implementations of the storage interface handle offset and size
	// in the same way.
	stat, err := reader.Stat()
	if err != nil {
		resp := minio.ToErrorResponse(err)
		code := resp.Code
		if code == "NoSuchKey" || code == "NoSuchBucket" {
			reader.Close()
			return empty, fmt.Errorf("%w", &NotFoundError{})
		}
		reader.Close()
		return empty, fmt.Errorf("%w", &AccessError{msg: fmt.Sprintf("%v", code)})
	}

	fetchOpt := NewFileStorageFetchOptions(opts...)

	// -ve or zero is read all
	if fetchOpt.Size <= 0 {
		fetchOpt.Size = stat.Size
	}
	// treat -ve as relative to end
	if fetchOpt.Offset < 0 {
		fetchOpt.Offset = stat.Size + fetchOpt.Offset
	}
	// still -ve (-ve offset was bigger than file)
	if fetchOpt.Offset < 0 {
		fetchOpt.Offset = 0
	}
	// offset after end of file
	if fetchOpt.Offset > 0 && fetchOpt.Offset >= stat.Size {
		reader.Close()
		return empty, &OffsetAfterEnd{msg: fmt.Sprintf("offset after EOF: %d", stat.Size)}
	}
	// requested more than available
	// should we error or be lenient?
	if fetchOpt.Offset+fetchOpt.Size > stat.Size {
		fetchOpt.Size = stat.Size - fetchOpt.Offset
	}

	// don't bother going back to remote for a 0 byte object, just return the empty dataslice with details
	if stat.Size == 0 {
		reader.Close()
		return empty, nil
	}

	_, err = reader.Seek(fetchOpt.Offset, 0)
	if err != nil {
		reader.Close()
		return empty, fmt.Errorf("%w", &ReadError{msg: fmt.Sprintf("%v", err)})
	}
	// Limit the reader so gin doesn't read beyond the selected file size.
	wrappedLimitedReader := NewCloseWrapper(io.LimitReader(reader, fetchOpt.Size), reader)
	return DataSlice{wrappedLimitedReader, fetchOpt.Offset, fetchOpt.Size, stat.Size}, nil
}

func (s *StoreS3) Exists(source, label, id string) (bool, error) {
	var err error
	startTime := time.Now().UnixNano()
	defer func() {
		reportStreamsOpMetric(s.promStreamsOperationDuration, startTime, "exists", err)
	}()
	key := createIdPath(source, label, id)
	_, err = s.client.StatObject(context.Background(), s.bucket, key, minio.StatObjectOptions{})
	if err == nil {
		return true, nil
	}
	resp := minio.ToErrorResponse(err)
	if resp.Code == "NoSuchKey" || resp.Code == "NoSuchBucket" {
		return false, nil
	}
	return false, fmt.Errorf("%w", &AccessError{msg: fmt.Sprintf("%v", resp.Code)})
}

func (s *StoreS3) Copy(sourceOld, labelOld, idOld, sourceNew, labelNew, idNew string) error {
	var err error
	startTime := time.Now().UnixNano()
	defer func() {
		reportStreamsOpMetric(s.promStreamsOperationDuration, startTime, "copy", err)
	}()
	// default srcObj to just the object name/hash
	srcObj := idOld
	existsAtRoot := false
	existsUnderSource, err := s.Exists(sourceOld, labelOld, idOld)
	if err != nil {
		return fmt.Errorf("error locating source object %s", srcObj)
	}
	if !existsUnderSource { // check under root if not found under source/label
		existsAtRoot, err = s.Exists("", "", idOld)
		if err != nil {
			return fmt.Errorf("error locating source object %s", srcObj)
		}
	}

	if existsUnderSource {
		srcObj = createIdPath(sourceOld, labelOld, idOld)
	} else if existsAtRoot {
		// the object being copied does not exist at source/label/ as expected, copy from root
		srcObj = createIdPath("", "", idOld)
	} else {
		// silently fail copy as we could not find the source file under root or source/label
		st.Logger.Debug().Msgf("Object %s not found for copy operation", idOld)
		return nil
	}

	idNew = createIdPath(sourceNew, labelNew, idNew)
	src := minio.CopySrcOptions{
		Bucket: s.bucket,
		Object: srcObj,
	}
	dst := minio.CopyDestOptions{
		Bucket: s.bucket,
		Object: idNew,
	}
	ui, err := s.client.CopyObject(context.Background(), dst, src)
	if err != nil {
		resp := minio.ToErrorResponse(err)
		return fmt.Errorf("s3 copy operation retured error %s", resp.Code)
	}
	fmt.Printf("Copied %s, successfully to %s - UploadInfo %v\n", src.Object, dst.Object, ui)

	return nil
}

func (s *StoreS3) Delete(source, label, id string, opts ...FileStorageDeleteOption) (bool, error) {
	var err error
	startTime := time.Now().UnixNano()
	defer func() {
		reportStreamsOpMetric(s.promStreamsOperationDuration, startTime, "delete", err)
	}()
	// a timing issue exists here as the read and delete are not atomic operations
	// we could enable versioning to fix this, but probably not worth it
	key := createIdPath(source, label, id)

	obj, err := s.client.StatObject(context.Background(), s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		resp := minio.ToErrorResponse(err)
		code := resp.Code
		if code == "NoSuchKey" || code == "NoSuchBucket" {
			return false, fmt.Errorf("%w", &NotFoundError{})
		}
		return false, fmt.Errorf("%w", &AccessError{msg: fmt.Sprintf("%v", code)})
	}

	deleteOpt := NewFileStorageDeleteOptions(opts...)

	if deleteOpt.IfOlderThan > 0 && obj.LastModified.Unix() >= deleteOpt.IfOlderThan {
		// don't delete if object is newer than the required timestamp
		return false, nil
	}
	err = s.client.RemoveObject(context.Background(), s.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		resp := minio.ToErrorResponse(err)
		code := resp.Code
		if code == "NoSuchKey" || code == "NoSuchBucket" {
			return false, fmt.Errorf("%w", &NotFoundError{})
		}
		return false, fmt.Errorf("%w", &AccessError{msg: fmt.Sprintf("%v", code)})
	}
	return true, nil

}

func (s *StoreS3) List(ctx context.Context, prefix string, startAfter string) <-chan FileStorageObjectListInfo {
	minioOptions := minio.ListObjectsOptions{Prefix: prefix, Recursive: true, WithMetadata: false, WithVersions: false}
	if startAfter != "" {
		minioOptions.StartAfter = startAfter
	}
	storageObjects := make(chan FileStorageObjectListInfo)
	go func() {
		sourceChan := s.client.ListObjects(ctx, s.bucket, minioOptions)
		defer func() { close(storageObjects) }() // Close the channel when all files are processed

		for {
			dataFromMinio := minio.ObjectInfo{}
			var ok bool
			select {
			case <-ctx.Done():
				return
			case dataFromMinio, ok = <-sourceChan:
				// The source channel is closed so exit.
				if !ok {
					return
				}
			}
			// Forward data from the minio channel to the next channel.
			// Split the key into source label and id.
			source, label, id := splitIdPath(dataFromMinio.Key)
			select {
			case <-ctx.Done():
				return
			case storageObjects <- FileStorageObjectListInfo{
				Key:    dataFromMinio.Key,
				Source: source,
				Label:  label,
				Id:     id,
			}:
				continue
			}
		}
	}()

	return storageObjects
}

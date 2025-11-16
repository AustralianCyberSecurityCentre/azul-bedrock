package store

import (
	"bytes"
	"io"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func getDataSliceBytes(t *testing.T, ds DataSlice) []byte {
	buf := make([]byte, ds.Size)
	readBytes, err := ds.DataReader.Read(buf)
	defer ds.DataReader.Close()

	if err != nil && err != io.EOF {
		t.Logf("input was %v", ds)
		t.Errorf("Failed to read the data from dataslice in tests! with error %s", err.Error())
	}
	return buf[:readBytes]
}

func TestCacheExtended(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "test-bedrock-store")
	require.NoError(t, err, "Error creating tmp")
	defer os.RemoveAll(dir)
	store, err := NewEmptyLocalStore(dir)
	require.NoError(t, err, "Error creating LocalStore")
	// Ensure max file size stored is 2kb.
	store, err = NewDataCache(1, 300, 256, store, StoreCacheMetricCollectors{})
	require.NoError(t, err, "Error creating LocalStore Cache")

	StoreImplementationBaseTests(t, store)
	StoreImplementationListBaseTests(t, store)

}

func TestCache(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "test-bedrock-store")
	defer os.RemoveAll(dir)
	if err != nil {
		t.Error("Error creating temp dir", err)
	}
	tables := []struct {
		input     []byte
		sha256    string
		sha1      string
		md5       string
		mimeType  string
		mimeMagic string
		fileType  string
	}{
		{[]byte(""), "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", "da39a3ee5e6b4b0d3255bfef95601890afd80709", "d41d8cd98f00b204e9800998ecf8427e", "inode/x-empty", "empty", "Data"},
		{[]byte("This is a really boring sentence."), "14aac9db48ce84652d68216fa32b4e495c04bd6ae538bd98e1faedae52f016b0", "3530adce5393b3cb165ee0cd7bac035cdc9a418c", "0f4b02a8173099970f3dcd1bcaf6bb0b", "text/plain", "ASCII text, with no line terminators", "Text"},
		// test sending/fetching duplicate content
		{[]byte("This is a really boring sentence."), "14aac9db48ce84652d68216fa32b4e495c04bd6ae538bd98e1faedae52f016b0", "3530adce5393b3cb165ee0cd7bac035cdc9a418c", "0f4b02a8173099970f3dcd1bcaf6bb0b", "text/plain", "ASCII text, with no line terminators", "Text"},
	}
	store, err := NewEmptyLocalStore(dir)
	if err != nil {
		t.Error("Error creating LocalStore", err)
	}
	store, err = NewDataCache(10, 300, 256, store, StoreCacheMetricCollectors{})
	if err != nil {
		t.Error("Error creating LocalStore Cache", err)
	}
	for _, table := range tables {
		size := uint64(len(table.input))

		// Convert raw bytes to reader
		reader := bytes.NewReader(table.input)
		readCloser := io.NopCloser(reader)

		// Seek back to 0 to allow usage of the file
		err = store.Put("source", "label", table.sha256, readCloser, int64(len(table.input)))
		if err != nil {
			t.Error("Failed to put data", err)
		}

		// now fetch the object back
		ds, err := store.Fetch("source", "label", table.sha256, WithOffsetAndSize(0, -1))
		if err != nil {
			t.Error("Failed to fetch object", err)
		}
		fetchedBytes := getDataSliceBytes(t, ds)
		if !bytes.Equal(fetchedBytes, table.input) {
			t.Errorf("Fetched data returned different content to input \nActual: %s\nExpect: %s", string(fetchedBytes), string(table.input))
		}
		// bail if was an empty object
		if len(table.input) <= 1 {
			continue
		}
		ds, err = store.Fetch("source", "label", table.sha256, WithOffsetAndSize(1, -1))
		if err != nil {
			t.Error("Failed to fetch slice of data for non-zero offset", err)
		}
		if !bytes.Equal(getDataSliceBytes(t, ds), table.input[1:]) {
			t.Error("Fetched data from offset returned different content to input")
		}
		ds, err = store.Fetch("source", "label", table.sha256, WithOffsetAndSize(0, 1))
		if err != nil {
			t.Error("Failed to fetch slice of data for non-zero size", err)
		}
		if !bytes.Equal(getDataSliceBytes(t, ds), table.input[0:1]) {
			t.Error("Fetched data with non-zero size returned different content to input")
		}
		ds, err = store.Fetch("source", "label", table.sha256, WithOffsetAndSize(1, 1))
		if err != nil {
			t.Error("Failed to fetch slice of data for non-zero size and non-zero offset", err)
		}
		if !bytes.Equal(getDataSliceBytes(t, ds), table.input[1:2]) {
			t.Error("Fetched data from non-zero offset and size returned different content to input")
		}
		ds, err = store.Fetch("source", "label", table.sha256, WithOffsetAndSize(int64(size)+1, 1))
		if err == nil {
			t.Error("Fetched data with offset outside of bounds was not handled")
		}
		if !reflect.DeepEqual(ds, NewDataSlice()) {
			t.Error("Fetched data from outside bounds returned result")
		}
		ds, err = store.Fetch("source", "label", table.sha256, WithOffsetAndSize(-int64(size*2), -1))
		if err != nil {
			t.Error("Failed to clamp offset to 0 for negative offset before file start", err)
		}
		if uint64(len(getDataSliceBytes(t, ds))) != size {
			t.Error("Did not return expected buffer size")
		}
	}
}

func BenchmarkCacheReadStore(b *testing.B) {
	dir, err := os.MkdirTemp("/tmp", "test-bedrock-store")
	defer os.RemoveAll(dir)
	require.NoError(b, err, "Error creating temp dir", err)

	store, err := NewEmptyLocalStore(dir)
	require.NoError(b, err, "Error creating local store", err)

	store, err = NewDataCache(10, 300, 256, store, StoreCacheMetricCollectors{})
	require.NoError(b, err, "Error creating cache store", err)

	BaseBenchmarkReadStore(b, store)
}

func BenchmarkCacheWriteStore(b *testing.B) {
	dir, err := os.MkdirTemp("/tmp", "test-bedrock-store")
	defer os.RemoveAll(dir)
	require.NoError(b, err, "Error creating temp dir", err)

	store, err := NewEmptyLocalStore(dir)
	require.NoError(b, err, "Error creating local store", err)

	store, err = NewDataCache(10, 300, 256, store, StoreCacheMetricCollectors{})
	require.NoError(b, err, "Error creating cache store", err)

	BaseBenchmarkWriteStore(b, store)
}

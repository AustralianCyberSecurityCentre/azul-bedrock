package store

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Must be 24 characters long
const aesDummyKey = "dummyaeskeyexactly24char"

func TestAESCtrStore(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "test-bedrock-store")
	defer os.RemoveAll(dir)
	require.NoError(t, err, "Error creating temp dir", err)

	store, err := NewEmptyLocalStore(dir)
	require.NoError(t, err, "Error creating local store", err)

	aesCtrStore := NewAESCtrStore(store, aesDummyKey, true)

	StoreImplementationBaseTests(t, aesCtrStore)
	StoreImplementationListBaseTests(t, aesCtrStore)
}

func TestPlainAesCtrStore(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "test-bedrock-store")
	defer os.RemoveAll(dir)
	require.NoError(t, err, "Error creating temp dir", err)

	store, err := NewEmptyLocalStore(dir)
	require.NoError(t, err, "Error creating local store", err)

	aesCtrStore := NewAESCtrStore(store, aesDummyKey, true)

	StoreImplementationBaseTests(t, aesCtrStore)
	StoreImplementationListBaseTests(t, aesCtrStore)
}

func TestAesCtrAtRest(t *testing.T) {
	/* Assert that data is actually encoded when stored */
	assert := assert.New(t)

	dir, err := os.MkdirTemp("/tmp", "test-bedrock-store")
	defer os.RemoveAll(dir)
	require.NoError(t, err, "Error creating temp dir", err)

	store, err := NewEmptyLocalStore(dir)
	require.NoError(t, err, "Error creating local store", err)

	aesCtrStore := NewAESCtrStore(store, aesDummyKey, true)

	var probMalware = []byte("Hello, this is malware!")
	// Convert raw bytes to reader
	reader := bytes.NewReader(probMalware)
	readCloser := io.NopCloser(reader)

	err = aesCtrStore.Put("testsource", "testlabel", "testid", readCloser, int64(len(probMalware)))
	require.NoError(t, err, "Error writing to AES_CTR store", err)

	// The AES_CTR store should return the original text
	testData, err := aesCtrStore.Fetch("testsource", "testlabel", "testid", WithOffsetAndSize(0, -1))
	require.NoError(t, err, "Error reading from AES_CTR store", err)

	readBuffer := getDataSliceBytesInterfaceTest(t, testData)
	assert.Equal(probMalware, readBuffer)

	// The filesystem store should not
	testData, err = store.Fetch("testsource", "testlabel", "testid"+AES_CTR_FILE_EXT, WithOffsetAndSize(0, -1))
	require.NoError(t, err, "Error reading from local store", err)

	readBuffer = getDataSliceBytesInterfaceTest(t, testData)
	assert.NotEqual(probMalware, readBuffer)
}

func TestAesExtensionsStripped(t *testing.T) {
	/* Verify that the AES extension is stripped off of various extensions. */
	dir, err := os.MkdirTemp("/tmp", "test-bedrock-store")
	defer os.RemoveAll(dir)
	require.NoError(t, err, "Error creating temp dir", err)

	store, err := NewEmptyLocalStore(dir)
	require.NoError(t, err, "Error creating local store", err)

	aesCtrStore := NewAESCtrStore(store, aesDummyKey, true)

	var probMalware = []byte("Hello, this is malware!")
	// Convert raw bytes to reader
	reader := bytes.NewReader(probMalware)
	readCloser := io.NopCloser(reader)

	err = aesCtrStore.Put("testsource", "testlabel", "testid", readCloser, int64(len(probMalware)))
	require.NoError(t, err, "Error writing to AES_CTR store", err)

	data, err := aesCtrStore.Fetch("testsource", "testlabel", "testid")
	require.Nil(t, err)
	resultData, err := io.ReadAll(data.DataReader)
	require.Nil(t, err)
	require.Equal(t, resultData, probMalware)

	// Add extension to request and it still works
	data, err = aesCtrStore.Fetch("testsource", "testlabel", "testid"+AES_CTR_FILE_EXT)
	require.Nil(t, err)
	resultData, err = io.ReadAll(data.DataReader)
	require.Nil(t, err)
	require.Equal(t, resultData, probMalware)

	// Even if you have accidental double extension.
	data, err = aesCtrStore.Fetch("testsource", "testlabel", "testid"+AES_CTR_FILE_EXT+AES_CTR_FILE_EXT)
	resultData, err = io.ReadAll(data.DataReader)
	require.Nil(t, err)
	require.Equal(t, resultData, probMalware)

	// Even if you have accidental double extension on exists.
	isExisting, err := aesCtrStore.Exists("testsource", "testlabel", "testid"+AES_CTR_FILE_EXT+AES_CTR_FILE_EXT)
	require.Nil(t, err)
	require.True(t, isExisting)

	// Copy
	copyErr := aesCtrStore.Copy("testsource", "testlabel", "testid"+AES_CTR_FILE_EXT+AES_CTR_FILE_EXT, "newSource", "NewLabel", "testid")
	require.Nil(t, copyErr)
	isExisting, err = aesCtrStore.Exists("newSource", "NewLabel", "testid"+AES_CTR_FILE_EXT)

	require.Nil(t, err)
	require.True(t, isExisting)

	// Delete should work as well.
	isDeleted, err := aesCtrStore.Delete("testsource", "testlabel", "testid"+AES_CTR_FILE_EXT+AES_CTR_FILE_EXT)
	require.Nil(t, err)
	require.True(t, isDeleted)
}

func TestPlainAfterAESCtr(t *testing.T) {
	/* Asserts that a disabled AES_CTR wrapper correctly finds AES_CTR'd files & that files afterwards
	   are stored without a AES_CTR */
	assert := assert.New(t)

	dir, err := os.MkdirTemp("/tmp", "test-bedrock-store")
	defer os.RemoveAll(dir)
	require.NoError(t, err, "Error creating temp dir", err)

	store, err := NewEmptyLocalStore(dir)
	require.NoError(t, err, "Error creating local store", err)

	aesCtrStore := NewAESCtrStore(store, aesDummyKey, true)

	var probMalware = []byte("Hello, this is malware!")
	// Convert raw bytes to reader
	reader := bytes.NewReader(probMalware)
	readCloser := io.NopCloser(reader)

	err = aesCtrStore.Put("testsource", "testlabel", "aesctredfile", readCloser, int64(len(probMalware)))
	require.NoError(t, err, "Error writing to AES_CTR store", err)

	// The filesystem (non-encrypted) store should not return the original string and instead return the encrypted version
	testData, err := store.Fetch("testsource", "testlabel", "aesctredfile"+AES_CTR_FILE_EXT, WithOffsetAndSize(0, -1))
	require.NoError(t, err, "Error reading from local store", err)

	readBuffer := getDataSliceBytesInterfaceTest(t, testData)
	assert.NotEqual(probMalware, readBuffer)

	// Disabling AES_CTR should still return valid contents for a file stored with AES_CTR on when
	// fetched via the AES_CTR store
	aesCtrStore = NewAESCtrStore(store, aesDummyKey, false)

	testData, err = aesCtrStore.Fetch("testsource", "testlabel", "aesctredfile", WithOffsetAndSize(0, -1))
	require.NoError(t, err, "Error reading from AES_CTR store", err)

	readBuffer = getDataSliceBytesInterfaceTest(t, testData)
	assert.Equal(probMalware, readBuffer)

	reader = bytes.NewReader(probMalware)
	readCloser = io.NopCloser(reader)
	// Storing a 'new' file should result in it not being AES_CTR'd
	err = aesCtrStore.Put("testsource", "testlabel", "notaesctredfile", readCloser, int64(len(probMalware)))
	require.NoError(t, err, "Error writing to AES_CTR store", err)

	// The filesystem provider should return the correct content
	testData, err = store.Fetch("testsource", "testlabel", "notaesctredfile", WithOffsetAndSize(0, -1))
	require.NoError(t, err, "Error reading from local store", err)

	readBuffer = getDataSliceBytesInterfaceTest(t, testData)
	assert.Equal(probMalware, readBuffer)
}

type Reader interface {
	Read(p []byte) (n int, err error)
}

/*
	Reader that intentionally releases bytes in an odd order, to catch out potential issues with the buffering of AES encryption.

As if you get a choppy buffer and are over reading (the buffer is longer than the content) it can cause AES to move along it's cipher on garbage.
This results in the output being corrupted AES content.
*/
type CustomChoppyByteReader struct {
	innerReader    io.Reader
	lastReadLength int
}

func (c *CustomChoppyByteReader) Read(p []byte) (n int, err error) {
	if c.lastReadLength <= 0 || c.lastReadLength > 50 {
		c.lastReadLength = 1
	}

	resultCount, err := c.innerReader.Read(p[:c.lastReadLength])
	// grow the buffer from 0-10
	c.lastReadLength = c.lastReadLength + rand.Intn(10)
	return resultCount, err
}

func TestAesChoppyBuffer(t *testing.T) {
	/* Verifies that reading and writing from a buffer that gives content in a random order works.

	e.g the buffer may give the first 10 bytes then the next 12 etc.
	This can occur in production in buffering and streaming situations where not all bytes can be immediately provided.

	Regression test due to issue that occurred when reading from Piped Gzip content that was providing bytes in a choppy manner.
	This caused AES corruption.
	*/
	assert := assert.New(t)

	dir, err := os.MkdirTemp("/tmp", "test-bedrock-store")
	defer os.RemoveAll(dir)
	require.NoError(t, err, "Error creating temp dir", err)

	store, err := NewEmptyLocalStore(dir)
	require.NoError(t, err, "Error creating local store", err)

	aesCtrStore := NewAESCtrStore(store, aesDummyKey, true)

	var probMalware = []byte("Hello, this is malware! held in a choppy buffer that will come through at an odd pace.")
	// Convert raw bytes to reader
	reader := bytes.NewReader(probMalware)
	// Choppy reader that provides bytes to a choppy fashion into AES.
	choppyReader := CustomChoppyByteReader{
		innerReader:    reader,
		lastReadLength: 1,
	}
	readCloser := io.NopCloser(&choppyReader)

	err = aesCtrStore.Put("testsource", "testlabel", "aesctredfile", readCloser, -1)
	require.NoError(t, err, "Error writing to AES_CTR store", err)

	testData, err := aesCtrStore.Fetch("testsource", "testlabel", "aesctredfile", WithOffsetAndSize(0, -1))
	require.NoError(t, err, "Error reading from AES_CTR store", err)

	readBuffer := getDataSliceBytesInterfaceTest(t, testData)
	assert.Equal(probMalware, readBuffer, "Bad encoding has occurred and AES encryption is corrupting files.")

	// Second fetch to verify that choppy reads don't prevent the read by corrupting the file during read.
	testData, err = aesCtrStore.Fetch("testsource", "testlabel", "aesctredfile", WithOffsetAndSize(0, -1))
	require.NoError(t, err, "Error reading from AES_CTR store", err)

	byteBufferLargerThanContent := make([]byte, len(probMalware)*4)
	readBytes, err := testData.DataReader.Read(byteBufferLargerThanContent)
	require.Equal(t, err, nil)
	assert.Equal(probMalware, byteBufferLargerThanContent[:readBytes])

	// confirm EOF with 0 bytes read is returned on subsequent reads.
	readBytes, err = testData.DataReader.Read(byteBufferLargerThanContent)
	require.Equal(t, err, io.EOF)
	require.Equal(t, 0, readBytes)
}

// Verify that if the load order is Store -> AES -> cache, the cache stores the raw data in it.
// This is used in dispatcher caching during Azul's normal operation.
func TestCacheWithAES(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "test-bedrock-store")
	require.NoError(t, err, "Error creating tmp")
	defer os.RemoveAll(dir)
	store, err := NewEmptyLocalStore(dir)
	require.NoError(t, err, "Error creating LocalStore")

	// Add AES encryption
	aesStore := NewAESCtrStore(store, aesDummyKey, true)

	// Ensure max file size stored is 2kb.
	cacheStore, err := NewDataCache(4, 300, 256, aesStore, StoreCacheMetricCollectors{})
	require.NoError(t, err, "Error creating LocalStore Cache")

	content := []byte("This is a really boring sentence.")
	source := "testing"
	label := "content"
	sha256 := fmt.Sprintf("%x", sha256.Sum256(content))
	reader := bytes.NewReader(content)
	readCloser := io.NopCloser(reader)

	cacheStore.Put(source, label, sha256, readCloser, int64(len(content)))
	// Load the file into the cache.
	_, err = cacheStore.Fetch(source, label, sha256, WithOffsetAndSize(0, -1))
	require.Nil(t, err, "Failed to fetch file from cache after storing it %v.", err)

	// Delete from AES store directly so only cache can hold th file now.
	aesStore.Delete(source, label, sha256)
	_, err = aesStore.Fetch(source, label, sha256, WithOffsetAndSize(0, -1))
	require.NotNil(t, err)

	// Fetch directly from the cache bypassing AES and content should be in raw form.
	dataSlice, err := cacheStore.Fetch(source, label, sha256, WithOffsetAndSize(0, -1))
	var notFoundError *NotFoundError
	if errors.As(err, &notFoundError) {
		require.Nil(t, err, "Cache doesn't contain the raw file and should.")
	}
	require.Nil(t, err)

	readData, err := io.ReadAll(dataSlice.DataReader)
	require.Nil(t, err)

	require.Equal(t, readData, content)

}

func BenchmarkAESCtrReadStore(b *testing.B) {
	dir, err := os.MkdirTemp("/tmp", "test-bedrock-store")
	defer os.RemoveAll(dir)
	require.NoError(b, err, "Error creating temp dir", err)

	store, err := NewEmptyLocalStore(dir)
	require.NoError(b, err, "Error creating local store", err)

	aesCtrStore := NewAESCtrStore(store, aesDummyKey, true)

	BaseBenchmarkReadStore(b, aesCtrStore)
}

func BenchmarkAESCtrWriteStore(b *testing.B) {
	dir, err := os.MkdirTemp("/tmp", "test-bedrock-store")
	defer os.RemoveAll(dir)
	require.NoError(b, err, "Error creating temp dir", err)

	store, err := NewEmptyLocalStore(dir)
	require.NoError(b, err, "Error creating local store", err)

	aesCtrStore := NewAESCtrStore(store, aesDummyKey, true)

	BaseBenchmarkWriteStore(b, aesCtrStore)
}

package store

import (
	"bytes"
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

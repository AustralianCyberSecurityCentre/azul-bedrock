package store

import (
	"bytes"
	"io"
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

	// The filesystem store should not return the original string while AES_CTR was on
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

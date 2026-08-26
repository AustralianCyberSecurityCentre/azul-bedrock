package store

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDirectorySplitterStore(t *testing.T) {
	// dir, err := os.MkdirTemp("/tmp", "test-bedrock-store")
	// defer os.RemoveAll(dir)
	// require.NoError(t, err, "Error creating temp dir", err)

	// store, err := NewEmptyLocalStore(dir)
	// require.NoError(t, err, "Error creating local store", err)
	store := NewStoreMem()

	splitterStore := NewDirectorySplitterStore(store)

	StoreImplementationBaseTests(t, splitterStore)
	StoreImplementationListBaseTests(t, splitterStore)
}

func TestDirectoryStructure(t *testing.T) {
	/* Assert that data is actually encoded when stored */
	assert := assert.New(t)
	var err error
	dir, err := os.MkdirTemp("/tmp", "test-bedrock-store")
	defer os.RemoveAll(dir)
	require.NoError(t, err, "Error creating temp dir", err)

	store, err := NewEmptyLocalStore(dir)
	require.NoError(t, err, "Error creating local store", err)

	// store := NewStoreMem()

	splitterStore := NewDirectorySplitterStore(store)

	var probMalware = []byte("Hello, this is malware!")
	// Convert raw bytes to reader
	reader := bytes.NewReader(probMalware)
	readCloser := io.NopCloser(reader)

	err = splitterStore.Put("testsource", "testlabel", "testid", readCloser, int64(len(probMalware)))
	require.NoError(t, err, "Error writing to splitter store", err)

	// The Splitter store should return the original text
	testData, err := splitterStore.Fetch("testsource", "testlabel", "testid", WithOffsetAndSize(0, -1))
	require.NoError(t, err, "Error reading from splitter store", err)

	readBuffer := getDataSliceBytesInterfaceTest(t, testData)
	assert.Equal(probMalware, readBuffer)

	// The filesystem store should not find file
	testData, err = store.Fetch("testsource", "testlabel", "testid", WithOffsetAndSize(0, -1))
	require.NotNil(t, err)

	// Adding Id prefix manually should find the file
	testData, err = store.Fetch("testsource", "testlabel", "testid"[:NumberOfChars]+"/testid", WithOffsetAndSize(0, -1))
	require.Nil(t, err)

	readBuffer = getDataSliceBytesInterfaceTest(t, testData)
	assert.Equal(probMalware, readBuffer)
}

func TestMigrationFromLegacy(t *testing.T) {
	/* Assert that data is actually encoded when stored */
	assert := assert.New(t)
	var err error
	dir, err := os.MkdirTemp("/tmp", "test-bedrock-store")
	defer os.RemoveAll(dir)
	require.NoError(t, err, "Error creating temp dir", err)

	store, err := NewEmptyLocalStore(dir)
	require.NoError(t, err, "Error creating local store", err)

	var probMalware = []byte("Hello, this is malware!")
	// Convert raw bytes to reader
	reader := bytes.NewReader(probMalware)
	readCloser := io.NopCloser(reader)

	err = store.Put("testsource", "testlabel", "testid", readCloser, int64(len(probMalware)))
	require.NoError(t, err, "Error writing to base store", err)

	// Read file from legacy store
	splitterStore := NewDirectorySplitterStore(store)
	testData, err := splitterStore.Fetch("testsource", "testlabel", "testid")
	require.NoError(t, err, "Error reading from splitter store", err)

	readBuffer := getDataSliceBytesInterfaceTest(t, testData)
	assert.Equal(probMalware, readBuffer)

	// List
	listCount := 0
	var objectVal FileStorageObjectListInfo
	for item := range splitterStore.List(context.Background(), "", "") {
		listCount += 1
		objectVal = item
	}
	// Should be one object
	require.Equal(t, 1, listCount)
	// Should be inserted item
	require.Contains(t, objectVal.Key, "testsource/testlabel")
	require.Equal(t, objectVal.Source, "testsource")
	require.Equal(t, objectVal.Label, "testlabel")
	require.Equal(t, objectVal.Id, "testid")

	err = splitterStore.Copy("testsource", "testlabel", "testid", "testsource", "testlabel", "testid2")
	require.Nil(t, err)

	// Fetch copied object and verify it.
	testData, err = splitterStore.Fetch("testsource", "testlabel", "testid2")
	require.NoError(t, err, "Error reading copied file from splitter store: %v", err)
	readBuffer = getDataSliceBytesInterfaceTest(t, testData)
	assert.Equal(probMalware, readBuffer)

	// Verify delete works on both the old and new version
	isDeleted, err := splitterStore.Delete("testsource", "testlabel", "testid")
	require.True(t, isDeleted)
	require.Nil(t, err)
	isDeleted, err = splitterStore.Delete("testsource", "testlabel", "testid2")
	require.True(t, isDeleted)
	require.Nil(t, err)
}

func TestDirectoryInMemory(t *testing.T) {
	/* Assert that data is actually encoded when stored */
	assert := assert.New(t)
	var err error
	// Memory store doesn't have pathing like directory store so is different.
	store := NewStoreMem()
	splitterStore := NewDirectorySplitterStore(store)

	var probMalware = []byte("Hello, this is malware!")
	// Convert raw bytes to reader
	reader := bytes.NewReader(probMalware)
	readCloser := io.NopCloser(reader)

	err = splitterStore.Put("testsource", "testlabel", "testid", readCloser, int64(len(probMalware)))
	require.NoError(t, err, "Error writing to splitter store", err)

	// The Splitter store should return the original text
	testData, err := splitterStore.Fetch("testsource", "testlabel", "testid", WithOffsetAndSize(0, -1))
	require.NoError(t, err, "Error reading from splitter store", err)

	readBuffer := getDataSliceBytesInterfaceTest(t, testData)
	assert.Equal(probMalware, readBuffer)

	// The filesystem store should not find file
	testData, err = store.Fetch("testsource", "testlabel", "testid", WithOffsetAndSize(0, -1))
	require.NotNil(t, err)

	// Adding Id prefix manually should find the file
	testData, err = store.Fetch("testsource", "testlabel", "testid"[:NumberOfChars]+"/testid", WithOffsetAndSize(0, -1))
	require.Nil(t, err)

	readBuffer = getDataSliceBytesInterfaceTest(t, testData)
	assert.Equal(probMalware, readBuffer)
}

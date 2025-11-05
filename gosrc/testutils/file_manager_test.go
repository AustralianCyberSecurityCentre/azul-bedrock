package testutils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const KNOWN_VT_SHA256 = "011dfdc48be1f58b8dee36c1098700e0154dac96688c1e38b6c45fda6a032fb7"
const KNOWN_VT_SHA256_LENGTH = 992
const NON_EXISTENT_URL = "localhost:9180"

type testContext struct {
	fileManager *FileManager
}

func setupTest(t *testing.T) *testContext {
	manager, err := NewFileManager()
	if err != nil {
		t.Fatalf("Failed to setup File manager with error %v", err)
	}
	return &testContext{
		fileManager: manager,
	}
}

/*Distinct combinations for enabling/disabling cache methods that should work.*/
func commonChecks(t *testing.T, verifyFunction func(*testContext)) {
	currentTestContext := setupTest(t)

	// Ensure file can be cached from VT into azure blob storage.
	currentTestContext.fileManager.settings.VirustotalEnabled = true
	currentTestContext.fileManager.settings.AzureBlobCacheEnabled = true
	currentTestContext.fileManager.settings.FileCachingEnabled = false
	t.Log("Ensure file can be cached from VT into azure blob storage.")
	verifyFunction(currentTestContext)

	// Ensure file can be cached from blob storage into azure file storage.
	currentTestContext.fileManager.settings.VirustotalEnabled = false
	currentTestContext.fileManager.settings.AzureBlobCacheEnabled = true
	currentTestContext.fileManager.settings.FileCachingEnabled = true
	t.Log("Ensure file can be cached from blob storage into azure file storage.")
	verifyFunction(currentTestContext)

	// Verification
	// Verify file can be downloaded from virustotal.
	currentTestContext.fileManager.settings.VirustotalEnabled = true
	currentTestContext.fileManager.settings.AzureBlobCacheEnabled = false
	currentTestContext.fileManager.settings.FileCachingEnabled = false
	t.Log("Verify file can be downloaded from virustotal.")
	verifyFunction(currentTestContext)

	// Verify file can be collected from file cache.
	currentTestContext.fileManager.settings.VirustotalEnabled = false
	currentTestContext.fileManager.settings.AzureBlobCacheEnabled = false
	currentTestContext.fileManager.settings.FileCachingEnabled = true
	t.Log("Verify file can be collected from file cache.")
	verifyFunction(currentTestContext)

	// Verify file can be collected from azure blob storage.
	currentTestContext.fileManager.settings.VirustotalEnabled = false
	currentTestContext.fileManager.settings.AzureBlobCacheEnabled = true
	currentTestContext.fileManager.settings.FileCachingEnabled = false
	t.Log("Verify file can be collected from azure blob storage.")
	verifyFunction(currentTestContext)
}

func TestGetFileWorks(t *testing.T) {
	verifyFunc := func(currentTestContext *testContext) {
		fileBytes, err := currentTestContext.fileManager.DownloadFileBytes(KNOWN_VT_SHA256)
		require.Nil(t, err)
		require.Equal(t, KNOWN_VT_SHA256_LENGTH, len(fileBytes))
	}
	commonChecks(t, verifyFunc)
}

func TestGetFileWorksNumber2(t *testing.T) {
	verifyFunc := func(currentTestContext *testContext) {
		fileLength := 1716
		fileBytes, err := currentTestContext.fileManager.DownloadFileBytes("3777cd83e38bb39e7c16bc63ecdf3d1732eb7b2d9d7ce912f4fe55f9a859a020")
		require.Nil(t, err)
		require.Equal(t, fileLength, len(fileBytes))

	}
	commonChecks(t, verifyFunc)
}

func TestFileNotFound(t *testing.T) {
	verifyFunc := func(currentTestContext *testContext) {
		// Fake hash that should cause failures.
		sha256 := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		_, err := currentTestContext.fileManager.DownloadFileBytes(sha256)
		if err == nil {
			t.Logf("Found sha256 %s and shouldn't have!", sha256)
		}
		require.IsType(t, &NotFoundError{}, err, err)
	}
	commonChecks(t, verifyFunc)
}

// Error cases

func SetKeyWithReset(settingKey string, newValue string) func() {
	originalValue := os.Getenv(settingKey)
	os.Setenv(settingKey, newValue)
	resetEnv := func() {
		os.Setenv(settingKey, originalValue)
	}
	return resetEnv
}

func TestBadVTUrl(t *testing.T) {
	vm, err := NewFileManager()
	vm.settings.RequestTimeout = 2
	vm.settings.VirustotalEnabled = true
	vm.settings.FileCachingEnabled = false
	vm.settings.AzureBlobCacheEnabled = false
	vm.settings.VirustotalApiUrl = "https://localhost:9180"
	require.Nil(t, err)
	_, err = vm.DownloadFileBytes(KNOWN_VT_SHA256)
	require.NotNil(t, err)
}

func TestBadBlobStorageCredentials(t *testing.T) {
	SetKeyWithReset("FILE_MANAGER_AZURE_STORAGE_ACCESS_KEY", "invalidsecretkeyohdear")
	_, err := NewFileManager()
	require.NotNil(t, err)
}

func TestNoBlobStorageCredentials(t *testing.T) {
	SetKeyWithReset("FILE_MANAGER_AZURE_STORAGE_ACCESS_KEY", "")
	_, err := NewFileManager()
	require.NotNil(t, err)
}

func TestBlobStorageBadUrl(t *testing.T) {
	SetKeyWithReset("FILE_MANAGER_AZURE_STORAGE_ACCOUNT_ADDRESS", "localhost:9180")
	_, err := NewFileManager()
	require.NotNil(t, err)
}

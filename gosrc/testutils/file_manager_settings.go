package testutils

import (
	"strings"

	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/settings"
	"github.com/go-viper/mapstructure/v2"
)

const CART_FILE_SUFFIX = ".cart"

type FileManagerSettings struct {
	RequestTimeout int `koanf:"file_manager_request_timeout"`
	// The URL to Virustotal's V3 API (guarantee no trailing slash).
	VirustotalApiUrl string `koanf:"file_manager_virustotal_api_url"`
	// Virustotal API key used to download files from Virustotal
	VirustotalApiKey string `koanf:"file_manager_virustotal_api_key"`
	// whether to attempt to download files from virustotal or not.
	VirustotalEnabled bool `koanf:"file_manager_virustotal_enabled"`
	// Directory where files are cached on the local file system when downloaded. (stored as carts)
	FileCacheDir string `koanf:"file_manager_file_cache_dir"`
	// Flag used to enable/disable the caching of test files.
	FileCachingEnabled bool `koanf:"file_manager_file_caching_enabled"`
	// URL from Azure storage blob (storage account name address)
	AzureStorageAccountAddress string `koanf:"file_manager_azure_storage_account_address"`
	// Storage account Access key. (SAS key) used to access the azure storage.
	AzureStorageAccessKey string `koanf:"file_manager_azure_storage_access_key"`
	// Name of the storage container within the blob storage.
	AzureContainerName string `koanf:"file_manager_azure_container_name"`
	// Flag used to enable/disable bucket caching.
	AzureBlobCacheEnabled bool `koanf:"file_manager_azure_blob_cache_enabled"`
}

var defaultSettings = FileManagerSettings{
	RequestTimeout:             30,
	VirustotalApiUrl:           "https://www.virustotal.com/api/v3",
	VirustotalApiKey:           "",
	VirustotalEnabled:          true,
	FileCacheDir:               "/var/tmp/azul",
	FileCachingEnabled:         true,
	AzureStorageAccountAddress: "",
	AzureStorageAccessKey:      "",
	AzureContainerName:         "azul-test-file-cache",
	AzureBlobCacheEnabled:      true,
}

func ParseFileManagerSettings() *FileManagerSettings {
	parsedSettings := settings.ParseSettings(defaultSettings, "", []mapstructure.DecodeHookFunc{})
	// Remove any trailing slashes
	parsedSettings.VirustotalApiKey = strings.TrimRight(parsedSettings.VirustotalApiKey, "/")
	return parsedSettings
}

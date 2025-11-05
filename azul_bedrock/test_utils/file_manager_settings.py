"""Settings used by file manager."""

from typing import Annotated

from pydantic import AfterValidator, ConfigDict
from pydantic_settings import BaseSettings


class FileManagerSettings(BaseSettings):
    """Environment settings that dictate how and where to download download test files."""

    model_config = ConfigDict(env_prefix="file_manager_")

    request_timeout: int = 30

    # The URL to Virustotal's V3 API (guarantee no trailing slash).
    virustotal_api_url: Annotated[str, AfterValidator(lambda url: url.rstrip("/"))] = (
        "https://www.virustotal.com/api/v3"
    )
    # Virustotal API key used to download files from Virustotal
    virustotal_api_key: str = ""
    # whether to attempt to download files from virustotal or not.
    virustotal_enabled: bool = True

    # Directory where files are cached on the local file system when downloaded. (stored as carts)
    file_cache_dir: str = "/var/tmp/azul"  # nosec B108
    # Flag used to enable/disable the caching of test files.
    file_caching_enabled: bool = True

    # URL from Azure storage blob (storage account name address)
    azure_storage_account_address: str = ""
    # Storage account Access key. (SAS key) used to access the azure storage.
    azure_storage_access_key: str = ""
    # Name of the storage container within the blob storage.
    azure_container_name: str = "azul-test-file-cache"
    # Flag used to enable/disable bucket caching.
    azure_blob_cache_enabled: bool = True

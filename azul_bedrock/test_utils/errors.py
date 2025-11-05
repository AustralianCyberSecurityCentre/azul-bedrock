"""Custom errors raised by any test utitlity."""

from azul_bedrock.test_utils.file_manager_settings import FileManagerSettings


class AzulFileNotFoundError(Exception):
    """Exception raised when file not found."""

    def __init__(self, sha256: str, local_cache_path: str, settings: FileManagerSettings) -> None:
        message = f"The file with hash '{sha256}' could not be found."
        if settings.file_caching_enabled:
            message += f" The file cache path {local_cache_path} was checked."
        if settings.azure_blob_cache_enabled:
            message += f" The azure blob url {settings.azure_storage_account_address} was checked."
        if settings.virustotal_enabled:
            message += f" Virustotal was checked using the base API {settings.virustotal_api_url}"

        super().__init__(message)


class AzureAuthError(Exception):
    """Exception to raise when authentication has failed to azure blob storage."""

    pass


class AzureBadContainerUrlError(Exception):
    """Exception raised when the azure blob storage couldn't be found."""

    pass

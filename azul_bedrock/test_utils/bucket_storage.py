"""Base class and implementations of bucket storage."""

import abc
import contextlib
import typing
from abc import abstractmethod
from io import BytesIO
from typing import Callable, TypeVar

import cart
from azure.core.exceptions import (
    ClientAuthenticationError,
    ResourceExistsError,
    ServiceRequestError,
)
from azure.storage.blob import ContainerClient

from azul_bedrock.test_utils.errors import AzureAuthError, AzureBadContainerUrlError
from azul_bedrock.test_utils.file_manager_settings import FileManagerSettings

_T = TypeVar("_T")


class BaseStorage(metaclass=abc.ABCMeta):
    """Base class for storage to implement so other storage providers can be implemented."""

    @abstractmethod
    def __init__(self, settings: FileManagerSettings):
        """Provide the settings to the storage implementation."""
        ...

    @abstractmethod
    def init_client(self):
        """Create the underlying storage account and verify auth is working."""
        ...

    @abstractmethod
    def download_file(self, sha256: str, bytes_handler: Callable[[str, typing.Iterator[bytes]], _T]) -> _T | None:
        """Download file from storage and pass the bytes of the file into bytes_handler function."""
        ...

    @abstractmethod
    def save_file(self, sha256: str, uncarted_content: BytesIO):
        """Save a file to the storage container."""
        ...


class AzureBlobStorage(BaseStorage):
    """Azure blob storage provider."""

    def __init__(self, settings: FileManagerSettings):
        self._settings = settings
        self._sha256_suffix = ".cart"

    def init_client(self):
        """Create the azure container client and verify the container exists."""
        if not self._settings.azure_storage_access_key:
            raise AzureAuthError("No storage access key provided to access the azure blob storage.")
        try:
            self._client = ContainerClient(
                account_url=self._settings.azure_storage_account_address,
                container_name=self._settings.azure_container_name,
                credential=self._settings.azure_storage_access_key,
            )
            # Create the container if it doesn't exist
            if not self._client.exists():
                self._client.create_container()
        except ClientAuthenticationError as e:
            raise AzureAuthError(f"Azure blob storage failed to authenticate with error {str(e)}")
        except ServiceRequestError as e:
            raise AzureBadContainerUrlError(
                f"Azure blob storage could not be found (check the blob URL) error: {str(e)}"
            )
        except ValueError as e:
            raise AzureBadContainerUrlError(f"Azure blob storage URL was completely invalid with error {str(e)}")
        except Exception:
            raise

    def download_file(self, sha256: str, bytes_handler: Callable[[str, typing.Iterator[bytes]], _T]) -> _T | None:
        """Download file from azure blob storage and pass the bytes of the file into bytes_handler function."""
        try:
            blob_client = self._client.get_blob_client(f"{sha256}{self._sha256_suffix}")
            if not blob_client.exists():
                return None
            file = blob_client.download_blob()
            return bytes_handler(sha256, file.chunks())
        except Exception:
            raise

    def save_file(self, sha256: str, uncarted_content: BytesIO):
        """Save a file to an azure blob storage container."""
        if not self._settings.azure_blob_cache_enabled:
            return
        carted_file = BytesIO()
        # NOTE - this reads the whole file into memory due to limitations of cart
        cart.pack_stream(uncarted_content, carted_file)
        # Seek to the end of the file to determine length
        carted_file.seek(0)
        # Ignore if the blob already exists.
        with contextlib.suppress(ResourceExistsError):
            self._client.upload_blob(f"{sha256}{self._sha256_suffix}", carted_file)

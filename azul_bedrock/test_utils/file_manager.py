"""Class that handles the downloading and caching of files for tests.

Downloading and caching files is necessary to prevent copyright issues.
Files are downloaded from Virustotal and then cached on the local file system.

NOTE - cached files are stored as carts for security reasons.
       (prevent accidental execution and to avoid AV scanners in blob storage)
"""

import logging
import os
import pathlib
import struct
import tempfile
import typing
from io import BytesIO
from typing import Callable, TypeVar

import cart
import httpx
from cachetools import LRUCache, cached

from azul_bedrock.test_utils.bucket_storage import AzureBlobStorage
from azul_bedrock.test_utils.errors import AzulFileNotFoundError
from azul_bedrock.test_utils.file_manager_settings import FileManagerSettings

# Cart header length
MANDATORY_CART_HEADER_LEN = struct.calcsize(cart.MANDATORY_HEADER_FMT)

ZERO_BYTE_FILE_SHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

# Maximum number of Virustotal download URLs to cache.
MAX_CACHED_VT_URLS = 1000
# Chunk size to read from a file. (5MB)
FILE_CHUNK_SIZE = 5 * 1024 * 1024

logger = logging.getLogger("TestFileManager")
# Global list of downloaded virustotal urls


@cached(cache=LRUCache(maxsize=MAX_CACHED_VT_URLS))
def get_vt_download_url(*, sha256: str, virustotal_url: str, api_key: str, timeout: int) -> str | None:
    """Get the download URL from virustotal to download a file with.

    The downloaded URL is used with a cache to reduce the risk of using the VT download quota twice.
    """
    resp = httpx.get(
        url=f"{virustotal_url}/files/{sha256}/download_url",
        headers={"x-Apikey": api_key},
        timeout=timeout,
        follow_redirects=True,
    )
    if resp.status_code == 404:
        return None

    resp.raise_for_status()

    download_url = resp.json().get("data", "")
    return download_url


_T = TypeVar("_T")


class FileManager:
    """Mange the downloading and caching of test files."""

    # Singleton instance
    _instance = None

    def __init__(self):
        """Setup the settings and temporary directory for the class."""
        self._settings = FileManagerSettings()
        self._temp_directory = tempfile.TemporaryDirectory(prefix="azul_file_manager_")
        if self._settings.azure_blob_cache_enabled:
            self._storage_client = AzureBlobStorage(self._settings)
            self._storage_client.init_client()

        if self._settings.file_caching_enabled:
            cache_dir = pathlib.Path(self._settings.file_cache_dir)
            try:
                if not cache_dir.exists():
                    cache_dir.mkdir(parents=True, exist_ok=True)
            except PermissionError:
                raise PermissionError(
                    f"Python doesn't have permissions to access the file cache in directory '{cache_dir}'"
                )
            canary_file = cache_dir.joinpath("permissionCanaryFile")
            with open(canary_file, "w") as f:
                f.write("Check to see if process can write to cache file location.")
            canary_file.unlink()

    def __new__(cls):
        """Override new to implement singleton to prevent excessive recreations of this class."""
        if cls._instance is None:
            cls._instance = super(FileManager, cls).__new__(cls)
            cls._instance.__init__()
        return cls._instance

    def _get_cache_file_path(self, sha256: str) -> pathlib.Path:
        """Get the file path to a cached file."""
        return pathlib.Path(os.path.join(self._settings.file_cache_dir, f"{sha256}.cart"))

    # --- Download handlers

    def _download_file_from_vt(
        self, sha256: str, bytes_handler: Callable[[str, typing.Iterator[bytes]], _T]
    ) -> _T | None:
        """Downloads a file from VT and passes a bytes iterator into a handler function.

        This function should be used with a `with` statement to ensure the underlying network connection closes.
        """
        try:
            logger.info("downloading test file from VT")
            vt_download_url = get_vt_download_url(
                sha256=sha256,
                virustotal_url=self._settings.virustotal_api_url,
                api_key=self._settings.virustotal_api_key,
                timeout=self._settings.request_timeout,
            )
            if not vt_download_url:
                return None

            with httpx.stream(
                "GET",
                vt_download_url,
                headers={"x-Apikey": self._settings.virustotal_api_key},
                timeout=self._settings.request_timeout,
                follow_redirects=True,
            ) as resp:
                resp.raise_for_status()
                return bytes_handler(sha256, resp.iter_bytes(FILE_CHUNK_SIZE))
        except httpx.HTTPStatusError as e:
            if e.response.status_code == 401:
                raise httpx.HTTPStatusError(
                    "Virustotal API key is invalid, with status code 401 "
                    + f"{e.response.reason_phrase} for url {e.response.url}",
                    request=e.request,
                    response=e.response,
                )
            raise
        except httpx.ConnectError as e:
            raise httpx.ConnectError(f"Failed to connect to virustotal server at url {e.request.url}")

    def _get_local_file_bytes(self, sha256: str) -> bytes | None:
        """Read the entire file into memory if it's found in the local cache."""
        local_file = self._get_cache_file_path(sha256)
        if not local_file.exists():
            return None
        unpacked = BytesIO()
        with local_file.open(mode="rb") as f:
            cart.unpack_stream(f, unpacked)
            unpacked.seek(0)
            return unpacked.getvalue()

    def _copy_local_file_to_temp(self, sha256: str) -> pathlib.Path | None:
        """Locate the file in the local cache and copy it to temp returning the path if it's present."""
        local_file = self._get_cache_file_path(sha256)
        if not local_file.exists():
            return None
        dest_path = pathlib.Path(os.path.join(self._temp_directory.name, f"{sha256}"))
        # Remove the file if it already exists
        if dest_path.exists():
            dest_path.unlink(missing_ok=True)

        cart.unpack_file(local_file, dest_path)
        return dest_path

    def _get_bytes_handler(self, _sha256: str, bytes_iterator: typing.Iterator[bytes]) -> bytes:
        """Get the raw bytes of the file from the iterator and read it all into memory.

        _sha256 is for compatibility with _get_tempfile_handler
        """
        raw_file = b"".join(bytes_iterator)
        if cart.is_cart(raw_file):
            raw_file_uncarted = BytesIO()
            cart.unpack_stream(BytesIO(raw_file), raw_file_uncarted)
            raw_file_uncarted.seek(0)
            return raw_file_uncarted.getvalue()
        return raw_file

    def _get_tempfile_handler(self, sha256: str, bytes_iterator: typing.Iterator[bytes]) -> pathlib.Path:
        """Get the raw bytes or carted bytes of the file from and save them to a temporary file."""
        dest_path = pathlib.Path(os.path.join(self._temp_directory.name, f"{sha256}"))
        # Remove the file if it already exists
        if dest_path.exists():
            dest_path.unlink(missing_ok=True)

        with dest_path.open(mode="wb") as in_file:
            for cur_bytes in bytes_iterator:
                in_file.write(cur_bytes)

        is_cart = False
        with dest_path.open(mode="rb") as temp_file:
            is_cart = cart.is_cart(temp_file.read(MANDATORY_CART_HEADER_LEN + 1))

        if is_cart:
            dest_path_temp = pathlib.Path(os.path.join(self._temp_directory.name, f"{sha256}_temp"))
            cart.unpack_file(dest_path, dest_path_temp)
            # Delete the cart
            dest_path.unlink()
            # Move this file over the cart.
            dest_path_temp.rename(dest_path)

        return dest_path

    # --- Cache handlers

    def _save_to_local_cache(self, sha256: str, uncarted_content: BytesIO):
        if not self._settings.file_caching_enabled:
            return
        cache_path: pathlib.Path = self._get_cache_file_path(sha256)
        with cache_path.open("wb") as out_path:
            cart.pack_stream(uncarted_content, out_path)

    @staticmethod
    def check_for_zero_byte_file(sha256: str):
        """Raise an exception if given a zero byte file."""
        if sha256 == ZERO_BYTE_FILE_SHA256:
            raise Exception("The provided sha256 provides to an empty file with zero bytes of content!")

    # --- Main public interface

    def download_file_bytes(self, sha256: str) -> bytes:
        """Download a file and uncart if necessary providing the output as raw bytes."""
        self.check_for_zero_byte_file(sha256)
        raw_bytes: bytes | None = None
        if self._settings.file_caching_enabled:
            raw_bytes = self._get_local_file_bytes(sha256)
            if raw_bytes:
                return raw_bytes
        if self._settings.azure_blob_cache_enabled:
            raw_bytes = self._storage_client.download_file(sha256, self._get_bytes_handler)
            if raw_bytes:
                self._save_to_local_cache(sha256, BytesIO(raw_bytes))
                return raw_bytes
        if self._settings.virustotal_enabled:
            raw_bytes = self._download_file_from_vt(sha256, self._get_bytes_handler)
            if raw_bytes:
                self._storage_client.save_file(sha256, BytesIO(raw_bytes))
                self._save_to_local_cache(sha256, BytesIO(raw_bytes))
                return raw_bytes

        raise AzulFileNotFoundError(sha256, str(self._get_cache_file_path(sha256)), self._settings)

    def download_file_path(self, sha256: str) -> pathlib.Path:
        """Download a file to the temporary directory and provide the file path.

        This allows streaming of the file and is useful for downloading large files.
        The directory full of temporary files is deleted at the end of testing.
        Note - the temporary file is not carted.
        """
        self.check_for_zero_byte_file(sha256)
        out_file_path: pathlib.Path | None = None
        if self._settings.file_caching_enabled:
            out_file_path = self._copy_local_file_to_temp(sha256)
            # If the file exists return it.
            if out_file_path:
                return out_file_path

        if self._settings.azure_blob_cache_enabled:
            out_file_path = self._storage_client.download_file(sha256, self._get_tempfile_handler)
            if out_file_path:
                with open(out_file_path, "rb") as f:
                    self._save_to_local_cache(sha256, f)
                return out_file_path

        if self._settings.virustotal_enabled:
            out_file_path = self._download_file_from_vt(sha256, self._get_tempfile_handler)
            if out_file_path:
                with open(out_file_path, "rb") as f:
                    self._storage_client.save_file(sha256, f)
                with open(out_file_path, "rb") as f:
                    self._save_to_local_cache(sha256, f)
                return out_file_path

        raise AzulFileNotFoundError(sha256, str(self._get_cache_file_path(sha256)), self._settings)

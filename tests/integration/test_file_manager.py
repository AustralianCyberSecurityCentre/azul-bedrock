import os
import unittest
from typing import Callable
from unittest import mock

import httpx

from azul_bedrock.test_utils.errors import AzureAuthError, AzureBadContainerUrlError
from azul_bedrock.test_utils.file_manager import (
    ZERO_BYTE_FILE_SHA256,
    AzulFileNotFoundError,
    FileManager,
)


class TestTestFileManager(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.known_vt_sha256 = "011dfdc48be1f58b8dee36c1098700e0154dac96688c1e38b6c45fda6a032fb7"
        cls.known_vt_sha256_length = 992
        cls.non_existent_url = "localhost:9180"
        super().setUpClass()

    def setUp(self):
        self._recreate_file_manager()
        return super().setUp()

    def _recreate_file_manager(self):
        """Manually clear the instance to prevent singleton behaviour for tests."""
        if hasattr(self, "file_manager"):
            self.file_manager._instance = None
        self.file_manager = FileManager()

    def __revert_env(self, key: str):
        """Set the env back to the original value after a test run."""
        old_value = os.environ.get(key)

        def inner_revert():
            if old_value is not None:
                os.environ[key] = old_value
            else:
                del os.environ[key]

        return inner_revert

    def set_bad_env(self, key: str, value: str):
        """Set an environment variable to a bad value so it can be loaded within a test."""
        if not os.environ.get(key) and os.environ.get(key.upper()):
            key = key.upper()

        self.addCleanup(self.__revert_env(key))
        os.environ[key] = value

    # --- Test normal cases.

    def test_file_manager_is_singleton(self):
        """Verify that a new instance of FileManager is in fact equal to an old instance of file manager."""
        current_file_manager = self.file_manager
        self.assertIs(FileManager(), current_file_manager)

    def test_getfile_from_all(self):
        """Test downloading the file and getting the path to the raw file."""
        # Arbitrary benign file in Virus Total.
        sha256 = self.known_vt_sha256
        file_length = self.known_vt_sha256_length
        # Note not a cart as no cart header
        file_first_thirty_bytes = b"\x7fELF\x02\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00LIBRARY_CODELI"

        def download_and_verify_file():
            file_path = self.file_manager.download_file_path(sha256)
            with file_path.open("rb") as f:
                self.assertEqual(len(f.read()), file_length)
                f.seek(0)
                self.assertEqual(f.read(30), file_first_thirty_bytes)

        self.common_tests(download_and_verify_file)

    def test_get_raw_bytes_from_all(self):
        """Test downloading the file and getting the raw bytes of the file."""
        # Arbitrary benign file in Virus Total.
        sha256 = "3777cd83e38bb39e7c16bc63ecdf3d1732eb7b2d9d7ce912f4fe55f9a859a020"
        file_length = 1716
        # Note not a cart as no cart header
        file_first_thirty_bytes = b"\x7fELF\x02\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00LIBRARY_CODELI"

        def download_and_verify_file():
            raw_bytes = self.file_manager.download_file_bytes(sha256)
            self.assertEqual(len(raw_bytes), file_length)
            self.assertEqual(raw_bytes[:30], file_first_thirty_bytes)

        self.common_tests(download_and_verify_file)

    def test_file_not_found_anywhere_raw_bytes(self):
        """Test failing to find the file anywhere when downloading and expecting to find the raw bytes."""

        def download_and_verify_files():
            with self.assertRaises(AzulFileNotFoundError) as e:
                self.file_manager.download_file_bytes(
                    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
                )

        self.common_tests(download_and_verify_files)

    def test_file_not_found_anywhere_file_path(self):
        """Test failing to find the file anywhere when downloading and expecting the file path."""

        def download_and_verify_files():
            with self.assertRaises(AzulFileNotFoundError) as e:
                self.file_manager.download_file_path(
                    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
                )

        self.common_tests(download_and_verify_files)

    def test_download_zero_byte_file(self):
        """Test downloading a zero byte file."""

        def download_and_verify_files():
            with self.assertRaises(Exception) as e:
                self.file_manager.download_file_bytes(ZERO_BYTE_FILE_SHA256)
            with self.assertRaises(Exception) as e:
                self.file_manager.download_file_bytes(ZERO_BYTE_FILE_SHA256)

        self.common_tests(download_and_verify_files)

    def common_tests(self, download_and_verify_file: Callable):
        """Common tests to ensure all the caching is working for a given file."""
        # Ensure file is cached in azure blob
        self.file_manager._settings.virustotal_enabled = True
        self.file_manager._settings.azure_blob_cache_enabled = True
        self.file_manager._settings.file_caching_enabled = False
        download_and_verify_file()

        # Ensure file is cached in local
        self.file_manager._settings.virustotal_enabled = False
        self.file_manager._settings.azure_blob_cache_enabled = True
        self.file_manager._settings.file_caching_enabled = True
        download_and_verify_file()

        # Verify the file is in each location. now
        # Exists in VT
        self.file_manager._settings.virustotal_enabled = True
        self.file_manager._settings.azure_blob_cache_enabled = False
        self.file_manager._settings.file_caching_enabled = False
        download_and_verify_file()

        # Exists in local file system
        self.file_manager._settings.virustotal_enabled = False
        self.file_manager._settings.azure_blob_cache_enabled = False
        self.file_manager._settings.file_caching_enabled = True
        download_and_verify_file()

        # Exists in S3
        self.file_manager._settings.virustotal_enabled = False
        self.file_manager._settings.azure_blob_cache_enabled = True
        self.file_manager._settings.file_caching_enabled = False
        download_and_verify_file()

    # --- Test error cases:

    def test_bad_vt_api_key(self):
        """Attempt to download from VT with a bad API key."""
        self.set_bad_env("file_manager_virustotal_api_key", "not-a-real-api-key-fail-auth")
        self._recreate_file_manager()
        # Enable only Virustotal requests
        self.file_manager._settings.virustotal_enabled = True
        self.file_manager._settings.azure_blob_cache_enabled = False
        self.file_manager._settings.file_caching_enabled = False
        with self.assertRaises(httpx.HTTPStatusError) as e:
            self.file_manager.download_file_bytes(self.known_vt_sha256)
        self.assertIn("Virustotal API key is invalid", str(e.exception))
        with self.assertRaises(httpx.HTTPStatusError) as e:
            self.file_manager.download_file_path(self.known_vt_sha256)
        self.assertIn("Virustotal API key is invalid", str(e.exception))

    def test_bad_vt_url(self):
        """Attempt to download from VT with a URL that doesn't point to any host."""
        self.set_bad_env("file_manager_virustotal_api_url", "https://" + self.non_existent_url)
        self.set_bad_env("file_request_timeout", "2")
        self._recreate_file_manager()
        # Enable only Virustotal requests
        self.file_manager._settings.virustotal_enabled = True
        self.file_manager._settings.azure_blob_cache_enabled = False
        self.file_manager._settings.file_caching_enabled = False
        with self.assertRaises(httpx.ConnectError) as e:
            self.file_manager.download_file_bytes(self.known_vt_sha256)
        self.assertIn("Failed to connect to virustotal", str(e.exception))
        with self.assertRaises(httpx.ConnectError) as e:
            self.file_manager.download_file_path(self.known_vt_sha256)
        self.assertIn("Failed to connect to virustotal", str(e.exception))

    def test_blob_storage_bad_credentials(self):
        """Attempt to create a file manager with bad azure blob credentials."""
        self.set_bad_env("file_manager_azure_storage_access_key", "invalidsecretkeyohdear")
        with self.assertRaises(AzureAuthError):
            self._recreate_file_manager()

    def test_no_blob_storage_credential(self):
        self.set_bad_env("file_manager_azure_storage_access_key", "")
        with self.assertRaises(AzureAuthError):
            self._recreate_file_manager()

    def test_blob_storage_bad_url(self):
        """Attempt to download and upload to azure blob storage with a bad URL that doesn't point to any server."""
        self.set_bad_env("file_manager_azure_storage_account_address", self.non_existent_url)
        self.set_bad_env("file_request_timeout", "2")
        with self.assertRaises(AzureBadContainerUrlError):
            self._recreate_file_manager()

    @mock.patch("pathlib.Path")
    def test_no_permissions_to_write(self, fake_path):
        """Attempt to create the FileManager while the cache directory is set to a location that should have permissions rejected."""
        self.set_bad_env("file_manager_file_cache_dir", "/tmp/azul")

        # Mock a guaranteed fail.
        def raise_error(*args, **kwargs):
            raise PermissionError()

        fake_path_obj = mock.MagicMock()
        # Guarantee folder doesn't exist
        fake_path_obj.exists.return_value = False
        fake_path_obj.mkdir.side_effect = raise_error
        fake_path.return_value = fake_path_obj

        with self.assertRaises(PermissionError) as e:
            self._recreate_file_manager()
        self.assertIn("Python doesn't have", str(e.exception))

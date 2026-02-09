"""Test suite to verify language_catalog is correctly configured and ready."""

import unittest
from azul_bedrock.exception_enums import ExceptionCodeEnum
from azul_bedrock.language_catalogs import catalog_creation
from azul_bedrock.settings import LanguageCatalogsEnum


class TestBasic(unittest.TestCase):
    def test_catalog_updated(self):
        """Check to verify the catalog's and the code are in sync."""
        for lang, catalog_from_code in catalog_creation._get_all_unchecked_catalogs().items():
            catalog_from_disk = catalog_creation.get_catalog(lang)

            # For each language verify all the enum values exist and are in equal in both versions of the catalog.
            for error_enum in ExceptionCodeEnum:
                self.assertEqual(
                    catalog_from_code.get(error_enum),
                    catalog_from_disk.get(error_enum),
                    "The catalog on disk is not equal to the catalog definied in code.\n"
                    + "Run the script `python azul_bedrock/language_catalogs/catalog_creation.py` to refresh the catalog.",
                )

    def test_catalogs_are_still_valid(self):
        """Check all the catalogs are considered valid."""
        for lang in LanguageCatalogsEnum:
            print(f"Checking the catalog `{lang.value}` is still valid.")
            loaded_catalog = catalog_creation.get_catalog(lang)
            # Verify the language catalog is valid.
            catalog_creation._validate_catalog(loaded_catalog, lang)

import io
import os
import unittest

import cart
import pydantic

# Because unittest doesn't have parameterised tests use pytest.
import pytest
import yaml

from azul_bedrock import identify
from azul_bedrock.test_utils import file_manager

BASE_FILE_DIR = os.path.join(os.path.dirname(__file__), "..", "..", "testdata", "binaries")


class TestBasic(unittest.TestCase):
    def test_import1(self):
        path = os.path.join(os.path.dirname(__file__), "test_identify.py")
        magic, mime, file_format, file_format_legacy, file_extension = identify.from_file(path)
        self.assertEqual(magic, "Python script text executable")
        self.assertEqual(mime, "text/x-script.python")
        self.assertEqual(file_format, "code/python")
        self.assertEqual(file_format_legacy, "Python Code")
        self.assertEqual(file_extension, "py")

    def test_import2(self):
        fm = file_manager.FileManager()
        # Benign Microsoft Word document with RC4.
        file_bytes = fm.download_file_bytes("5f94858a80328bec92a0508ce3a9f4d4b088eb4f80a14569f856e7e01b72d642")
        magic, mime, file_format, file_format_legacy, file_extension = identify.from_buffer(file_bytes)
        self.assertEqual(
            magic,
            "Composite Document File V2 Document, Little Endian, Os: Windows, Version 5.2, Code page: 1252, Author: DLEBLANC-DEV11B, Template: Normal.dot, Last Saved By: DLEBLANC-DEV11B, Revision Number: 1, Name of Creating Application: Microsoft Office Word, Total Editing Time: 08:00, Create Time/Date: Fri Feb  6 02:33:00 2009, Last Saved Time/Date: Fri Feb  6 02:41:00 2009, Number of Pages: 1, Number of Words: 0, Number of Characters: 0, Security: 1",
        )
        self.assertEqual(mime, "application/msword")
        self.assertEqual(file_format, "document/office/word")
        self.assertEqual(file_format_legacy, "MS Word Document")
        self.assertEqual(file_extension, "doc")


class IdentifyTestData(pydantic.BaseModel):
    """TestData spec."""

    class TestCase(pydantic.BaseModel):
        """Test case data spec."""

        sha256: str
        file_format: str
        file_format_legacy: str
        file_extension: str
        mime: str = ""
        magic: str = ""

    identify_tests: list[TestCase]


found_test_cases = []
loc = os.path.join(os.path.dirname(__file__), "identify_test.yaml")
with open(loc, "r") as f:
    try:
        raw = yaml.safe_load(f)
        test_yaml = IdentifyTestData(**raw)
        for tc in test_yaml.identify_tests:
            found_test_cases.append(
                (tc.sha256, tc.file_format, tc.file_format_legacy, tc.file_extension, tc.magic, tc.mime)
            )
    except Exception:
        print("Exception when attempting to load identifyTest.yaml data.")
        raise


if len(found_test_cases) == 0:
    raise Exception("Failed to find any test cases from identify_test.yaml")

active_file_manager = file_manager.FileManager()


# All files in the test-files are currently listed here
@pytest.mark.parametrize(
    "sha256,file_format,file_format_legacy,file_extension,magic,mime",
    found_test_cases,
)
def test_file_format_legacys(
    sha256: str, file_format: str, file_format_legacy: str, file_extension: str, magic: str, mime: str
):
    """Parameterised test to allow quick bulk testing of file types."""
    # Force dos_ident to run on all win32 exe's to ensure function is working.
    if identify.cfg is None:
        identify.cfg = identify._load_config()
        for rule in identify.cfg.refine_rules:
            if rule.function_name == "dos_ident":
                rule.trigger_on.append("executable/windows/pe32")
                break
    if sha256 == file_manager.ZERO_BYTE_FILE_SHA256:
        file_bytes = b""
    else:
        file_bytes = active_file_manager.download_file_bytes(sha256)
    magic_calc, mime_calc, file_format_calc, file_format_legacy_calc, file_extension_calc = identify.from_buffer(
        file_bytes
    )
    msg_fn = (
        lambda var_type, calc, expected, sha256: f"!!! --- {var_type} are not equal (calculated) {calc} != {expected} "
        + f"(expected) for file '{sha256}' --- !!!"
    )
    print(
        f"Buffered values {magic_calc=}, {mime_calc=}, {file_format_calc=}, {file_format_legacy_calc=}, {file_extension_calc=}"
    )
    assert file_format_calc == file_format, msg_fn("file_format", file_format_calc, file_format, sha256)
    assert file_format_legacy_calc == file_format_legacy, msg_fn(
        "file_format_legacy", file_format_legacy_calc, file_format_legacy, sha256
    )
    assert file_extension_calc == file_extension, msg_fn("file_ext", file_extension_calc, file_extension, sha256)
    if magic:
        assert magic_calc == magic, msg_fn("file_ext", file_extension_calc, file_extension, sha256)
    if mime:
        assert mime_calc == mime, msg_fn("file_extension", file_extension_calc, file_extension, sha256)


""" Python code to allow you to format a whole directory read for processing.
import os
from collections import Counter


paths = []
for dir, _, file_name_list in os.walk("."):
    for file_name in file_name_list:
        path = os.path.join("dir-name", file_name)
        if path.endswith(".cart"):
            paths.append(path)
        #else:
        #    # Delete non-cart files
        #    os.remove(path)

paths.sort()
for p in paths:
    print(f'(t, "{p}", "", "", ""),')
"""

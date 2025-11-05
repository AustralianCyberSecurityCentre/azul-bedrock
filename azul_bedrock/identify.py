"""Python implementation of identify code.

It is intended for use during plugin tests only.
"""

import io
import os
import re
import struct
import tempfile
import zipfile
from functools import cached_property
from typing import Any, Callable

import magic
import pydantic
import yaml
import yara

cfg = None
cmagic = magic.Magic(keep_going=True, raw=True)
cmime = magic.Magic(keep_going=True, raw=True, mime=True)

POINTS_STRONG = 15
POINTS_WEAK = 1
MIN_POINTS = 15

# Max size of the start of the file to buffer for identification (32kB)
MAX_INDICATOR_BUFFERED_BYTES_SIZE = 32000


def zip_ident(buf_start_of_file: bytes, file_path: str, fallback: str = "unknown", **kwargs) -> tuple[str, str, str]:
    """Extract filenames of a zipfile and attempt to identify a file type."""
    file_list = []
    try:
        with zipfile.ZipFile(file_path, "r") as zf:
            file_list = [zfname for zfname in zf.namelist()]
    except Exception:
        return fallback

    tot_files = 0
    tot_class = 0
    tot_jar = 0

    is_ipa = False
    is_jar = False
    is_word = False
    is_excel = False
    is_ppt = False
    doc_props = False
    doc_rels = False
    doc_types = False
    android_manifest = False
    android_dex = False
    nuspec = False
    psmdcp = False

    for file_name in file_list:
        if file_name.startswith("META-INF/"):
            is_jar = True
        elif file_name == "AndroidManifest.xml":
            android_manifest = True
        elif file_name == "classes.dex":
            android_dex = True
        elif file_name.startswith("Payload/") and file_name.endswith(".app/Info.plist"):
            is_ipa = True
        elif file_name.endswith(".nuspec"):
            nuspec = True
        elif file_name.startswith("package/services/metadata/core-properties/") and file_name.endswith(".psmdcp"):
            psmdcp = True
        elif file_name.endswith(".class"):
            tot_class += 1
        elif file_name.endswith(".jar"):
            tot_jar += 1
        elif file_name.startswith("word/"):
            is_word = True
        elif file_name.startswith("xl/"):
            is_excel = True
        elif file_name.startswith("ppt/"):
            is_ppt = True
        elif file_name.startswith("docProps/"):
            doc_props = True
        elif file_name.startswith("_rels/"):
            doc_rels = True
        elif file_name == "[Content_Types].xml":
            doc_types = True

        tot_files += 1

    if 0 < tot_files < (tot_class + tot_jar) * 2:
        is_jar = True

    if is_jar and android_manifest and android_dex:
        return "android/apk"
    elif is_ipa:
        return "ios/ipa"
    elif is_jar:
        return "java/jar"
    elif (doc_props or doc_rels) and doc_types:
        if is_word:
            return "document/office/word"
        elif is_excel:
            return "document/office/excel"
        elif is_ppt:
            return "document/office/powerpoint"
        elif nuspec and psmdcp:
            # It is a nupkg file. Identify as archive/zip for now.
            return "archive/zip"
        else:
            return "document/office/unknown"
    else:
        return "archive/zip"


def cart_ident(buf_start_of_file: bytes, file_path: str, fallback: str = "unknown", **kwargs) -> tuple[str, str, str]:
    """Identify cart files (done this way to make it easy to replicate to GO)."""
    if len(buf_start_of_file) > 38:
        cart_magic = buf_start_of_file[:4]
        # Everything in cart header is little endian.
        version = int.from_bytes(buf_start_of_file[4 : 4 + 2], "little")
        reserved = int.from_bytes(buf_start_of_file[6 : 6 + 8], "little")
        if cart_magic == b"CART" and version == 1 and reserved == 0:
            return "archive/cart"
    return fallback


def dos_ident(buf_start_of_file: bytes, file_path: str, **kwargs) -> tuple[str, str, str]:
    """Identify what type of windows binary a file is (dll vs pe and 32 vs 64bit)."""
    # Data is too small to be any windows executable, so label it as unknown.
    if len(buf_start_of_file) < 48:
        return "unknown"
    # noinspection PyBroadException
    try:
        file_header = io.BytesIO(buf_start_of_file)
        if buf_start_of_file[0:2] != b"MZ":
            raise ValueError()

        (header_pos,) = struct.unpack("<I", buf_start_of_file[:0x40][-4:])
        file_header.seek(header_pos)
        if file_header.read(4) != b"PE\x00\x00":
            raise ValueError()
        (machine_id,) = struct.unpack("<H", file_header.read(2))
        if machine_id == 0x014C:
            width = 32
        elif machine_id == 0x8664:
            width = 64
        else:
            raise ValueError()
        val = file_header.read(18)[-2:]
        (characteristics,) = struct.unpack("<H", val)
        if characteristics & 0x2000:
            pe_type = "dll"
        elif characteristics & 0x0002:
            pe_type = "pe"
        else:
            raise ValueError()
        return "executable/windows/%s%i" % (pe_type, width)
    except Exception:  # nosec B110
        pass
    return "executable/windows/dos"


# FUTURE identify password protected office docs.
# No simple 1:1 tool for golang ignoring for now
# def office_ident(data: bytes):
#     if data["type"] in [
#         "document/office/word",
#         "document/office/excel",
#         "document/office/powerpoint",
#         "document/office/unknown",
#     ]:
#         try:
#             msoffcrypto_obj = msoffcrypto.OfficeFile(open(path, "rb"))
#             if msoffcrypto_obj and msoffcrypto_obj.is_encrypted():
#                 data["type"] = "document/office/passwordprotected"
#         except Exception:
#             # If msoffcrypto can't handle the file to confirm that it is/isn't password protected,
#             # then it's not meant to be. Moving on!
#             pass


def pdf_ident(buf_start_of_file: bytes, file_path: str, fallback: str = "unknown", **kwargs) -> tuple[str, str, str]:
    """Determine if a pdf is password protected or a portfolio."""
    # Password protected documents typically contain '/Encrypt'
    regex_encrypt = re.compile(b"/Encrypt")
    regex_collection = re.compile(b"/Type/Catalog/Collection")
    with open(file_path, "rb") as f:
        for cur_line in f.readlines():
            if regex_encrypt.search(cur_line):
                return "document/pdf/passwordprotected"
            # Portfolios typically contain '/Type/Catalog/Collection
            elif regex_collection.search(cur_line):
                return "document/pdf/portfolio"
        return fallback


def yara_ident(
    buf_start_of_file: bytes,
    file_path: str,
    magic: str = "",
    mime: str = "",
    current_type: str = "",
    fallback: str = "unknown",
) -> str:
    """Runs yara rules on files that mime and magic couldn't identify."""
    # FUTURE - remove if no issue with json identification is noted.
    # Check if the file is a misidentified json first before running the yara rules - may not be required.
    # try:
    #     json.loads(data)
    #     return "text/json"
    # except Exception:  # nosec B110
    #     pass
    # Run yara rules.
    externals = {"magic": magic, "mime": mime, "current_type": current_type}
    try:
        matches = cfg.yara_rules.match(file_path, externals=externals, fast=True)
        matches.sort(key=lambda x: x.meta.get("score", 0), reverse=True)
        for match in matches:
            ftype = match.meta.get("type", None)
            if ftype:
                return ftype
    except Exception as e:
        print(f"Yara file identifier failed with error: {str(e)}")
        matches = []

    return fallback


class Config(pydantic.BaseModel):
    """Identify config spec."""

    class IdMapping(pydantic.BaseModel):
        """A mapping between id, extension and legacy type."""

        id: str
        legacy: str = ""
        extension: str = ""

    class Rule(IdMapping):
        """Identify config spec."""

        magic: str | None = None
        magicC: re.Pattern | None = None
        mime: str | None = None
        mimeC: re.Pattern | None = None

    class Indicator(pydantic.BaseModel):
        """Identify config spec."""

        id: str
        trigger_on: list[str]
        re_strong: list[str] = []
        re_strongC: list[re.Pattern] = []
        re_weak: list[str] = []
        re_weakC: list[re.Pattern] = []

    class RefineRules(pydantic.BaseModel):
        """Rules to run once a type is identified."""

        function_name: str
        trigger_on: list[str]
        run_on_func_output: bool = False

    id_mappings: list[IdMapping]

    @pydantic.computed_field
    @cached_property
    def id_mapping_dict(self) -> dict[str, IdMapping]:
        """Get the id_mappings in a dictionary with the id's as the keys."""
        return {x.id: x for x in self.id_mappings}

    rules: list[Rule]
    indicators: list[Indicator]

    refine_rules: list[RefineRules]
    refine_rules_mapping: dict[str, Callable[[bytes, dict[str, int]], tuple[str, str, str]]] = {
        "zip_ident": zip_ident,
        "dos_ident": dos_ident,
        "cart_ident": cart_ident,
        "pdf_ident": pdf_ident,
        "yara_ident": yara_ident,
    }

    trusted_mimes: dict[str, str]

    yara_rules: Any | None = None


def _load_config():
    """Load the identify config file."""
    loc = os.path.join(os.path.dirname(__file__), "identify.yaml")
    with open(loc, "r") as f:
        raw = yaml.safe_load(f)

    loc = os.path.join(os.path.dirname(__file__), "trusted_mime.yaml")
    with open(loc, "r") as f:
        raw_mime = yaml.safe_load(f)

    _cfg = Config(**raw, **raw_mime)

    for rule in _cfg.rules:
        if rule.magic:
            rule.magicC = re.compile(rule.magic.lower())
        if rule.mime:
            rule.mimeC = re.compile(rule.mime.lower())

    for indicator in _cfg.indicators:
        for reraw in indicator.re_strong:
            indicator.re_strongC.append(re.compile(reraw.encode()))
        for reraw in indicator.re_weak:
            indicator.re_weakC.append(re.compile(reraw.encode()))

    YARA_RULES_PATH = os.path.join(os.path.dirname(__file__), "yara_rules.yar")
    _cfg.yara_rules = yara.compile(
        filepaths={"default": YARA_RULES_PATH}, externals={"mime": "", "magic": "", "type": ""}
    )

    return _cfg


def _apply_rules(magics: list[str], mimes: list[str]) -> tuple[str, str]:
    """Apply identification rules over magic and mime."""
    for rule in cfg.rules:
        if rule.magicC:
            for mgc in magics:
                if rule.magicC.search(mgc.lower()):
                    return rule.id, rule.legacy, rule.extension
        if rule.mimeC:
            for mime in mimes:
                if rule.mimeC.search(mime.lower()):
                    return rule.id, rule.legacy, rule.extension


def _apply_indicators(id: str, buf_start_of_file: bytes) -> str:
    """Apply content indicators over file content."""
    ret_id = id
    best_score = 0
    best_id = ""

    for indicator in cfg.indicators:
        tally = 0
        if id not in indicator.trigger_on:
            continue
        for pattern in indicator.re_strongC:
            tally += len(pattern.findall(buf_start_of_file)) * POINTS_STRONG
        for pattern in indicator.re_weakC:
            tally += len(pattern.findall(buf_start_of_file)) * POINTS_WEAK
        if best_score < tally:
            best_id = indicator.id
            best_score = tally

    if best_score >= MIN_POINTS:
        ret_id = best_id
    return ret_id


def from_buffer(data: bytes) -> tuple[str, str, str, str, str]:
    """Identify via bytes buffer."""
    with tempfile.NamedTemporaryFile() as f:
        f.write(data)
        f.flush()
        return from_file(f.name)


def from_file(path: str) -> tuple[str, str, str, str, str]:
    """Identify via file path."""
    with open(path, "rb") as f:
        buf_start_of_file = f.read(MAX_INDICATOR_BUFFERED_BYTES_SIZE)
        global cfg
        if not cfg:
            cfg = _load_config()
        try:
            magics = cmagic.from_file(path).split("\n")
        except magic.MagicException as e:
            magics = ["error - " + e.message.decode()]
        try:
            mimes = cmime.from_file(path).split("\n")
        except magic.MagicException as e:
            mimes = ["error - " + e.message.decode()]

        for i, _ in enumerate(magics):
            # Strip ensures removal of any excess white space
            magics[i] = magics[i].removeprefix("-").strip()
        for i, _ in enumerate(mimes):
            # Strip ensures removal of any excess white space
            mimes[i] = mimes[i].removeprefix("-").strip()

        # First attempt at identifying types
        file_format, legacy_override, extension_override = _apply_rules(magics, mimes)
        file_format_with_overrides = file_format

        # Find fist good mime type and use it.
        best_mime = ""
        for mime in mimes:
            if mime != "":
                best_mime = mime
                break
        # Try to identify off mime if magic couldn't find the type
        if file_format == "unknown" or file_format == "text/plain":
            for mime in mimes:
                mime = _dotdump(mime)
                if mime in cfg.trusted_mimes.keys():
                    file_format = cfg.trusted_mimes[mime]
                    break

        file_format = _apply_indicators(file_format, buf_start_of_file)

        fn_ran = False
        for r_fn in cfg.refine_rules:
            # Skip all functions that don't run on other functions output once a function has been run.
            if fn_ran and r_fn.run_on_func_output is False:
                continue
            if file_format in r_fn.trigger_on:
                fn_ran = True
                file_format = cfg.refine_rules_mapping[r_fn.function_name](
                    buf_start_of_file,
                    path,
                    magic=magics[0],
                    mime=best_mime,
                    current_type=file_format,
                    fallback=file_format,
                )

        if file_format_with_overrides == file_format and legacy_override != "":
            file_format_legacy = legacy_override
            extension = extension_override
        else:
            found_type = cfg.id_mapping_dict.get(file_format, None)
            if found_type is None:
                print(f"Error: the assemblyline type '{file_format}' doesn't have a mapping and should")
                raise Exception(f"Assemblyline type found was '{file_format}' doesn't have a mapping and should")

            file_format_legacy = found_type.legacy
            extension = found_type.extension

        return magics[0], best_mime, file_format, file_format_legacy, extension


def _dotdump(s):
    """Replace all non-ascii characters with '.' (required for assemblyline trusted mime types)."""
    if isinstance(s, str):
        s = s.encode()
    return "".join(["." if x < 32 or x > 126 else chr(x) for x in s])


if __name__ == "__main__":
    import sys

    if not len(sys.argv) > 1:
        print("must supply file path")
        sys.exit(1)
    path = sys.argv[1]
    magic, mime, file_format, file_format_legacy, extension = from_file(path)
    print(f"'{magic}' + '{mime}' -> {file_format} {file_format_legacy} {extension}")

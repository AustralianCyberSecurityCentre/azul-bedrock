"""Models for content endpoints."""

from enum import StrEnum

from pydantic import BaseModel, ConfigDict

from azul_bedrock import models_network as azm
from azul_bedrock.models_restapi.basic import BaseModelRepr


class BinaryData(azm.FileInfo):
    """Bedrock FileInfo with further enrichment."""

    model_config = ConfigDict(extra="ignore")

    filename: str | None = None
    track_source_references: str | None = None
    label: azm.DataLabel | None = None


class BinaryHexView(BaseModel):
    """Binary Hexview return format."""

    class BinaryHexHeader(BaseModel):
        """Header for hex data."""

        address: str
        hex: list[str] | str
        ascii: str

    class BinaryHexValue(BaseModel):
        """Row of hex data."""

        address: int
        hex: list[str] | str
        ascii: str

    hex_strings: list[BinaryHexValue]
    header: BinaryHexHeader
    has_more: bool
    next_offset: int
    content_length: int


class SearchResultType(StrEnum):
    """The original encoding of a search result in a binary's data."""

    ASCII = "ASCII"
    UTF16 = "UTF-16"
    Hex = "hex"


class SearchResult(BaseModelRepr):
    """A discovered instance of a particular string in a file."""

    string: str
    offset: int
    length: int
    encoding: SearchResultType


class BinaryStrings(BaseModel):
    """Binary string return format."""

    strings: list[SearchResult]
    has_more: bool
    next_offset: int
    time_out: bool = False  # timeout field for if ai string filter takes too long to process.

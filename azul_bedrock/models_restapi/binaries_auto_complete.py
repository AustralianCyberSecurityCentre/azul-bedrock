"""Data models for binary search auto-complete."""

from typing import Annotated, Literal, Optional, Union

from pydantic import BaseModel, Field


class AutocompleteNone(BaseModel):
    """Autocomplete mode where nothing is autocompleted."""

    type: Literal["None"] = "None"


class AutocompleteInitial(BaseModel):
    """Autocomplete mode where no input has been made yet."""

    type: Literal["Initial"] = "Initial"


class AutocompleteError(BaseModel):
    """Autocomplete mode where the current input is invalid."""

    type: Literal["Error"] = "Error"
    column: int
    message: str


PrefixType = (
    Literal["empty"] | Literal["case-sensitive"] | Literal["case-insensitive"] | Literal["numeric"] | Literal["range"]
)


class AutocompleteFieldName(BaseModel):
    """Autocomplete mode where the name of a field is autocompleted."""

    type: Literal["FieldName"] = "FieldName"
    prefix: str
    """The current input for the field's name."""
    prefix_type: PrefixType
    """The type of prefix that was identified."""
    has_value: bool
    """If the user has already specified a value for this field."""


class AutocompleteFieldValue(BaseModel):
    """Autocomplete mode where nothing is autocompleted."""

    type: Literal["FieldValue"] = "FieldValue"
    key: Optional[str]
    """The key the user has specified for this field."""
    prefix: str
    """The current input for the field's value."""
    prefix_type: PrefixType
    """The type of prefix that was identified."""


AutocompleteContext = Annotated[
    Union[AutocompleteNone, AutocompleteInitial, AutocompleteFieldName, AutocompleteFieldValue, AutocompleteError],
    Field(discriminator="type"),
]

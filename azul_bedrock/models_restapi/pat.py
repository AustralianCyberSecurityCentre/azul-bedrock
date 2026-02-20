"""PAT restapi related models."""

from enum import StrEnum
from typing import Annotated

from pydantic import AwareDatetime, BaseModel, ConfigDict, PlainSerializer, StringConstraints


class PATRequest(BaseModel):
    """Request for a PAT."""

    name: Annotated[str, StringConstraints(min_length=4, max_length=100)]
    roles: list[str]


class PATView(BaseModel):
    """The PAT view without the PAT itself."""

    # Extra is ignored to ensure the PAT is dropped if a dict with the PAT were validated against this model.
    model_config = ConfigDict(extra="ignore")

    id: str
    pat_name: str
    owner_username: str
    roles: list[str]
    creation_date: Annotated[AwareDatetime, PlainSerializer(lambda v: v.isoformat(), return_type=str)]
    last_used_date: Annotated[AwareDatetime, PlainSerializer(lambda v: v.isoformat(), return_type=str)]


class PATIssue(PATView):
    """Issuing of a PAT token."""

    pat: Annotated[str, StringConstraints(min_length=4, max_length=100)]
    # Base64 encoded pat_name:PAT put into the X-API-Key header for authentication to work.
    ready_api_key: str


class ListOfPAT(BaseModel):
    """Listing of PAT values."""

    pats: list[PATView]
    warnings: str = ""


class PATDeleteEnum(StrEnum):
    """Possible results for deletion of a PAT."""

    failed = "failed"
    not_found = "not found can't delete"
    success = "success"


class PATDeleteResponse(BaseModel):
    """Response to a request to delete a PAT."""

    result: PATDeleteEnum

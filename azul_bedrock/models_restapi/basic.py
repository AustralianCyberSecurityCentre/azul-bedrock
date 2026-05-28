"""Data models for basic data structures."""

from pydantic import BaseModel

from azul_bedrock.models_network import BaseModelStrict


class BaseModelRepr(BaseModelStrict):
    """Replace default import string in repr."""

    # expected to be imported as 'from azul_bedrock import models_restapi'
    _DEFAULT_IMPORT = "models_restapi"


class QueryInfo(BaseModel):
    """Information about a query performed in Opensearch."""

    query_type: str
    index: str
    query: dict | list[dict]
    run_time_ms: int | None = None
    args: list | None = None
    kwargs: dict | None = None
    response: dict | None = None


class Meta(BaseModelRepr):
    """Meta is where non-data goes (interesting things about the query)."""

    security: str | None = None
    sec_filter: str | None = None
    queries: list[QueryInfo] | None = None
    complete: bool = False


class Response(BaseModelRepr):
    """Generic metastore response."""

    data: dict | BaseModel | list[dict] | list[BaseModel]
    meta: Meta


class Author(BaseModelRepr):
    """A single author."""

    security: str | None = None
    category: str
    name: str
    version: str | None = None
    stream: str | None = None

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


class UserSecurity(BaseModel):
    """Contain common properties relating to user access."""

    # set of security labels accessible by user
    labels: list[str] = []
    # list of all available inclusive/exclusive/markings security labels
    labels_inclusive: list[str] = []
    labels_exclusive: list[str] = []
    labels_markings: list[str] = []
    # md5 of all security labels
    unique: str = ""
    # users max security as a string
    max_access: str = ""
    # azul-security presets the user is able to use
    allowed_presets: list[str] = []


class UserAccess(BaseModel):
    """Opensearch information for user."""

    # raw account info from Opensearch
    account_info: dict = {}
    # is security enabled for deployment?
    security_enabled: bool = False
    # does account bypass dls roles for access?
    privileged: bool = False
    # set of internal Opensearch roles associated with user
    roles: list[str] = []

    security: UserSecurity = None

"""Models representing user and service authentication."""

from enum import StrEnum

from pydantic import BaseModel, ConfigDict


class ApiAccessEnum(StrEnum):
    """Different access enums."""

    # Grants user to all API endpoints (does not include admin level endpoints)
    # Makes the rest of the options pointless.
    All = "all"
    # Allow for source based uploads
    BinarySourceUpload = "binary-source-upload"
    # Allows user to insert a child binary.
    BinaryChildUpload = "binary-child-upload"
    # Allows user to access the binary download request interface.
    BinaryDownloadRequest = "binary-download-request"
    # Allow user to download content and augmented streams.
    BinaryDownloadStreams = "binary-download-streams"
    # Allow user to use the hex and strings viewing APIs.
    BinaryHexAndStrings = "binary-hex-and-strings"
    # Allow user to expedite a binaries processing.
    BinaryExpedite = "binary-expedite"
    # Allow user to create and delete binary tags.
    BinaryModifyTags = "binary-modify-tags"
    # Allow user to search through binary metadata.
    BinarySearch = "binary-search"
    # Allow user to search for features, pivot on features etc.
    FeatureSearch = "features-search"
    # Allows user to create and delete feature tags.
    FeatureModifyTags = "feature-modify-tags"
    # Allows user to search through plugins and view their statuses.
    PluginSearch = "plugin-search"
    # Allow users to search through the list of available sources.
    SourcesSearch = "sources-search"


class CredentialFormat(StrEnum):
    """Allowed credential variants."""

    none = "none"
    basic = "basic"
    jwt = "jwt"
    oauth = "oauth"


class Credentials(BaseModel):
    """Credentials for user access to systems."""

    model_config = ConfigDict(use_enum_values=True)

    format: CredentialFormat
    unique: str  # unique identifier for credentials (for caching)
    username: str | None = None  # for basic
    password: str | None = None  # for basic
    token: str | None = None  # for jwt or oauth


class UserInfo(BaseModel):
    """A user of the system."""

    username: str = "unknown"
    org: str = "unknown"
    roles: list[str] = []
    api_access: list[ApiAccessEnum] = []
    email: str | None = None

    credentials: Credentials | None = None
    decoded: dict | None = None
    # Unique Identifier for the user if OIDC auth is used this will be the subject `sub` claim.
    unique_id: str

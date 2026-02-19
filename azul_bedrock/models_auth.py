"""Models representing user and service authentication."""

from enum import StrEnum

from pydantic import BaseModel


class CredentialFormat(StrEnum):
    """Allowed credential variants."""

    none = "none"
    basic = "basic"
    jwt = "jwt"
    oauth = "oauth"


class Credentials(BaseModel):
    """Credentials for user access to systems."""

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
    email: str | None = None

    credentials: Credentials | None = None
    decoded: dict | None = None
    # Unique Identifier for the user if OIDC auth is used this will be the subject `sub` claim.
    unique_id: str

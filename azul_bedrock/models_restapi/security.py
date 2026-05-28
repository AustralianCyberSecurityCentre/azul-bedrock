"""Security API models.

NOTE - it's important that this stays in a seperate file to enable lazy imports in azul-security.
"""

from azul_bedrock.models_restapi.basic import BaseModel


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
    # users max security string formatted for display rather than database usage.
    max_access_display: str = ""
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

    security: UserSecurity

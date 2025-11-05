"""Data models for plugins."""

from pydantic import ConfigDict

from azul_bedrock.models_restapi.basic import Author, BaseModelRepr
from azul_bedrock.models_restapi.binaries import StatusEvent


class PluginFeature(BaseModelRepr):
    """Feature a plugin can produce."""

    name: str
    desc: str | None = None
    tags: list[str] | None = None
    type: str | None = None


class PluginEntity(BaseModelRepr):
    """More detailed plugin info."""

    category: str
    name: str
    version: str | None = None
    contact: str | None = None
    last_completion: str | None = None
    stats: dict | None = None
    description: str | None = None
    features: list[PluginFeature] | None = None
    config: dict | None = None
    security: str


class Plugin(BaseModelRepr):
    """Plugin information from registration event."""

    # drop a bunch of fields
    model_config = ConfigDict(extra="ignore")

    author: Author
    entity: PluginEntity
    security: str
    timestamp: str


class LatestPluginWithVersions(BaseModelRepr):
    """Plugin Entity details for the latest version of a plugin, and a list of all the versions of that plugin."""

    versions: list[str] = []
    newest_version: PluginEntity | None = None


class PluginStatusSummary(LatestPluginWithVersions):
    """Plugin Entity details for the latest version of a plugin, and a list of all the versions of that plugin."""

    last_completion: str | None = None
    success_count: int = 0
    error_count: int = 0


class StatusGroup(BaseModelRepr):
    """Info for events matching a status type of a plugin."""

    status: str
    num_items: int
    items: list[StatusEvent]


class PluginInfo(BaseModelRepr):
    """Info for specific plugin."""

    num_entities: int
    plugin: PluginEntity
    status: list[StatusGroup]

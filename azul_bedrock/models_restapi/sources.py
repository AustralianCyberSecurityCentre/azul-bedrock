"""Data models for sources."""

from azul_bedrock.models_restapi.basic import BaseModelRepr


class Source(BaseModelRepr):
    """Compilation of source information."""

    name: str
    newest: str | None = None
    num_entities: int


class EventSource(BaseModelRepr):
    """A submission of a binary."""

    security: str | None = None

    name: str
    timestamp: str | None = None
    references: dict = {}

    settings: dict[str, str] = dict()

    # unique combination of submission info
    track_source_references: str | None = None


class ReferenceSet(BaseModelRepr):
    """Summary of a reference field set for a source in metastore."""

    track_source_references: str
    timestamp: str  # most recent entity
    num_entities: int
    # If true it means num_entities is the minimum number of entities not necessarily the full count.
    # This is triggered when num_entities is greater than the number of values Opensearch is queried for.
    num_entities_min: bool

    values: dict[str, str]


class References(BaseModelRepr):
    """Source references."""

    items: list[ReferenceSet]

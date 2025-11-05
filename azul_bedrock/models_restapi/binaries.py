"""Data models for binary_find queries."""

from enum import StrEnum
from typing import Literal

from pydantic import ConfigDict

from azul_bedrock import models_network as azm
from azul_bedrock.models_restapi.basic import Author, BaseModelRepr
from azul_bedrock.models_restapi.features import FeatureValueTag
from azul_bedrock.models_restapi.sources import EventSource


class FindBinariesSortEnum(StrEnum):
    """Possible options for sorting results."""

    score = "_score"
    source_timestamp = "source.timestamp"
    timestamp = "timestamp"


class IncludeCousinsEnum(StrEnum):
    """Enum to decide how wide the cousin search should be for nearby searches."""

    No = "no"
    Standard = "yes"
    Large = "yes_large"


class ReadTagsTag(BaseModelRepr):
    """Entity tag in system."""

    tag: str
    num_entities: int
    num_entities_approx: bool


class ReadTags(BaseModelRepr):
    """Collection of tags in the system, including entity counts."""

    tags: list[ReadTagsTag]
    num_tags: int
    num_tags_approx: bool


class EntityFindItemSource(BaseModelRepr):
    """Source of found entity."""

    depth: int
    name: str
    timestamp: str
    references: dict


class EntityTag(BaseModelRepr):
    """A tag attached to an entity."""

    sha256: str
    tag: str
    type: str

    owner: str | None = None
    timestamp: str
    security: str

    num_entities: int | None = None


class EntityFindItem(BaseModelRepr):
    """Found entity."""

    # the hash that was used to look up the binary, if no hash was used it will be the found binary id (sha256)
    key: str
    exists: bool  # when find is supplied with hashes, this might be false if the hash is not found
    # if this is true, it should be possible to retrieve content for the binary
    has_content: bool
    # The binary was found by multiple keys and this return value only holds the id and the key that had the hit.
    is_duplicate_find: bool | None = None

    sources: list[EntityFindItemSource] | None = None

    file_size: int | None = None
    file_format_legacy: str | None = None
    file_format: str | None = None
    file_extension: str | None = None
    magic: str | None = None
    mime: str | None = None
    filenames: list[str] | None = None

    # info about what document fields were a match with search parameters
    highlight: dict[str, list[str]] | None = None

    tags: list[EntityTag] | None = None

    md5: str | None = None
    sha1: str | None = None
    sha256: str | None = None
    sha512: str | None = None
    ssdeep: str | None = None
    tlsh: str | None = None


class EntityFind(BaseModelRepr):
    """Package results of find query with metadata about the query."""

    items_count: int = 0  # number of binaries that match the search
    items: list[EntityFindItem]  # limited number of entities that match the query


class EntityFindSimpleItem(BaseModelRepr):
    """Found entity."""

    sha256: str | None = None


class EntityFindSimpleFamilyItem(BaseModelRepr):
    """Found family entity."""

    sha256: str | None = None
    track_link: str | None = None
    author_name: str | None = None
    author_category: str | None = None
    timestamp: str | None = None


class EntityFindSimple(BaseModelRepr):
    """Package results of find query with metadata about the query."""

    items: list[EntityFindSimpleItem]  # limited number of entities that match the query
    after: str | None = None  # pagination key for next request
    total: int | None = None  # on first request only, expected number of items if all pages are collected


class EntityFindSimpleFamily(BaseModelRepr):
    """Package results of find query with metadata about the query."""

    items: list[EntityFindSimpleFamilyItem]  # limited number of entities that match the query
    after: str | None = None  # pagination key for next request
    total: int | None = None  # on first request only, expected number of items if all pages are collected


class EntityModel(BaseModelRepr):
    """A representation of the model used for binaries."""

    keys: dict[str, str]


class BinaryDocuments(BaseModelRepr):
    """Result of checking if newer documents exist."""

    count: int
    newest: str | None = None


class BinarySource(BaseModelRepr):
    """Metadata collecting various sources into direct and indirect groups."""

    source: str
    direct: list[EventSource] = []
    indirect: list[EventSource] = []


class EntityInstanceAuthor(BaseModelRepr):
    """A single plugin author."""

    security: str | None = None
    category: str
    name: str
    version: str | None = None


class PathNode(azm.PathNode):
    """Relationship link from a parent entity to a child, or the top level link in a path (no parent)."""

    _DEFAULT_IMPORT: str = "models_restapi"

    timestamp: str
    relationship: dict | None = None
    # optional tracking info needed to delete a parent-child relationship
    # depending on context, this node may either the 'parent' or 'child' in the tracked link
    track_link: str | None = None


class EntityInstance(BaseModelRepr):
    """Info to identify a unique set of features, streams, info, etc. for an entity."""

    key: str
    author: EntityInstanceAuthor
    action: azm.BinaryAction
    stream: str | None = None

    num_feature_values: int


class DatastreamInstances(azm.Datastream):
    """Stream + embedded authors info."""

    # required as this inherits directly from BaseModelStrict not Repr
    _DEFAULT_IMPORT = "models_restapi"

    instances: list[str]
    label: list[azm.DataLabel]


class FeatureValuePart(BaseModelRepr):
    """Sub elements of a feature value."""

    # offset where feature occurred within primary stream of entity (byte gte, byte lte)
    location: list[list[int]] | None = None

    # primitive values
    integer: str | None = None
    float: str | None = None
    datetime: str | None = None
    binary_string: str | None = None

    # additional meaning
    scheme: str | None = None
    netloc: str | None = None
    filepath: str | None = None
    params: str | None = None
    query: str | None = None
    fragment: str | None = None
    username: str | None = None
    password: str | None = None
    hostname: str | None = None
    ip: str | None = None
    port: int | None = None

    # further meaning
    filepath_unix: list[str] | None = None
    filepath_unixr: list[str] | None = None

    model_config = ConfigDict(coerce_numbers_to_str=True)


class BinaryFeatureValue(azm.FeatureValue):
    """Feature value that has been aggregated over binary events."""

    model_config = ConfigDict(coerce_numbers_to_str=True)

    # required as this inherits directly from BaseModelStrict not Repr
    _DEFAULT_IMPORT = "models_restapi"

    # there can be multiple labels for a feature
    label: list[str] = []
    # feature value may be divided into pieces
    parts: FeatureValuePart | None = None

    instances: list[str]
    location: list[list[int]] = []  # offset in entity content stream of this feature value
    tags: list[FeatureValueTag] = []  # all tags associated with feature value


class BinaryInfo(BaseModelRepr):
    """Binary info with instance of origin."""

    info: dict
    instance: str


class BinaryDiagnostic(BaseModelRepr):
    """A diagostic message for a binary.

    This may indicate that processing issues may be encountered for this file,
    or if otherwise some special state exists for this binary.
    """

    severity: Literal["info"] | Literal["warning"] | Literal["error"]
    # Identifier for the type of diagnostic
    id: str
    # Human readable title
    title: str
    # Human readable body, plain-text
    body: str


class BinaryMetadataDetail(StrEnum):
    """Set of valid details that can be retrieved from binary metadata."""

    total_hits = "total_hits"
    documents = "documents"
    security = "security"
    sources = "sources"
    features = "features"
    info = "info"
    datastreams = "datastreams"
    instances = "instances"
    parents = "parents"
    children = "children"
    tags = "tags"
    feature_tags = "feature_tags"


class BinaryMetadata(BaseModelRepr):
    """Data for a specific entity, individual properties may not be set based on query parameters (for performance)."""

    documents: BinaryDocuments
    security: list[str] = []
    sources: list[BinarySource] = []
    parents: list[PathNode] = []
    children: list[PathNode] = []
    instances: list[EntityInstance] = []
    features: list[BinaryFeatureValue] = []
    streams: list[DatastreamInstances] = []
    info: list[BinaryInfo] = []
    diagnostics: list[BinaryDiagnostic] = []
    tags: list[EntityTag] = []


class SimilarFuzzyMatchRow(BaseModelRepr):
    """Similarity result row."""

    sha256: str
    score: int | float


class SimilarFuzzyMatch(BaseModelRepr):
    """Ssdeep similarity calculation result."""

    matches: list[SimilarFuzzyMatchRow]


class SimilarMatchRow(BaseModelRepr):
    """Similarity result row."""

    sha256: str
    score_sum: int
    score_percent: int = 0
    contributions: list[list[str | int]]


class SimilarMatch(BaseModelRepr):
    """Similarity calculation result."""

    num_feature_values: int
    matches: list[SimilarMatchRow]
    timestamp: str
    status: str


class ReadNearbyLink(BaseModelRepr):
    """Link between a child and a parent or source."""

    id: str

    child: str
    parent: str | None = None

    child_node: PathNode

    source: EventSource | None = None


class ReadNearby(BaseModelRepr):
    """Holds links and nodes surrounding a focus entity."""

    id_focus: str
    links: list[ReadNearbyLink]


class ReadAllEntityTags(BaseModelRepr):
    """Tags for a binary."""

    items: list[EntityTag]


class AnnotationUpdated(BaseModelRepr):
    """Useful key value pairs derived from an Opensearch update (using painless script) on the annotation index."""

    total: int
    updated: int
    deleted: int


class StatusInputEntity(BaseModelRepr):
    """Entity input for status."""

    # drop a bunch of fields
    model_config = ConfigDict(extra="ignore")

    sha256: str


class StatusInput(BaseModelRepr):
    """Entity input for status."""

    # drop a bunch of fields
    model_config = ConfigDict(extra="ignore")

    entity: StatusInputEntity


class StatusEntity(BaseModelRepr):
    """Entity info for status."""

    status: str
    error: str | None = None
    message: str | None = None
    runtime: float | None = None
    input: StatusInput


class StatusEvent(BaseModelRepr):
    """Status information to return from the api."""

    # drop a bunch of fields
    model_config = ConfigDict(extra="ignore")

    timestamp: str
    author: Author
    entity: StatusEntity
    completed: int = 0
    security: str


class Status(BaseModelRepr):
    """Statuses of a binary."""

    items: list[StatusEvent]


class OpensearchDocuments(BaseModelRepr):
    """Documents from opensearch for a binary.

    Note: total_docs may be greater than the length of items if the total_hits exceeded the size allowed.
    """

    items: list[dict]
    total_docs: int

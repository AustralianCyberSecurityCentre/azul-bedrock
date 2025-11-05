"""Models representing network format of azul events."""

from __future__ import annotations

import base64
import datetime
from enum import StrEnum
from typing import Annotated, Any

from pydantic import (
    AwareDatetime,
    BaseModel,
    ConfigDict,
    NonNegativeInt,
    PlainSerializer,
)

# decoded feature value types
VALUE_DECODED = int | float | str | datetime.datetime | bytes
# generally this module will be imported with this rename
STD_IMPORT = "azm"
# the current model version expected on events - must match events/event.go
CURRENT_MODEL_VERSION: int = 5


def repr_reproduce(instance: BaseModelStrict, name: str | None = None, required: list[str] | None = None):
    """Pydantic repr output constructs valid python code to create a minimal instance of the target.

    This makes unit tests less hellish to write.
    """
    if not name:
        # default to class name
        name = instance.__class__.__name__
    if not required:
        # list of properties that do not require key=value format as they are always present.
        required = []

    def g(prop: str):
        """Returns the needed property on the instance."""
        return getattr(instance, prop)

    # this code isn't triggered under normal conditions, so its ok to do a model dump
    setted = instance.model_dump(exclude=set(required), exclude_defaults=True).keys()
    if instance._DEFAULT_IMPORT:
        ret = f"{instance._DEFAULT_IMPORT}.{name}("
    else:
        ret = f"{name}("
    ret += ", ".join([repr(g(x)) for x in required])
    if setted:
        if required:
            ret += ", "
        ret += ", ".join([f"{k}={repr(g(k))}" for k in setted])
    ret += ")"
    return ret


class BaseModelStrict(BaseModel):
    """Base model which provides extra restrictions and capabilities.

    * forbids extra properties to prevent subtle issues.
    * minimal copy+pasteable repr output to make printing results nice.
    """

    # expected to be imported as 'from models_network import azm'
    _DEFAULT_IMPORT: str = STD_IMPORT

    model_config = ConfigDict(extra="forbid")

    def __repr__(self):
        """Custom pydantic repr."""
        return repr_reproduce(self)

    def __str__(self) -> str:
        """Use our repr function for pretty printing."""
        return self.__repr__()


#
# Azul common components for events
#


class BinaryAction(StrEnum):
    """Valid event types."""

    def __repr__(self):
        """Return simple repr."""
        return f"{STD_IMPORT}.BinaryAction.{self.name}"

    #
    # binary events
    #
    # Binary that has been uploaded to Azul
    # Must have entity.datastreams
    # Must not be child of other event types
    Sourced = "sourced"
    # A binary has been extracted from another binary, which should be further processed
    # For example, an encoded PE in a PE was extracted
    # Must have entity.datastreams with a label=content stream
    Extracted = "extracted"
    # An alternative binary representation of this file has been produced
    # For example, a screenshot or decompiled source code
    # Must have entity.datastreams with a label!=content stream
    Augmented = "augmented"
    # Metadata for a binary has been mapped into Azul from an external database
    # For example, metadata for a given binary has been found and copied from VirusTotal
    # Must not have entity.datastreams
    # Must not be parent of other event types
    # Must not be a child of other event types
    Mapped = "mapped"
    # A binary has been enriched with additional metadata
    # For example, features have been identified by a plugin for this binary
    # Must not have entity.datastreams
    # Must not be parent of other event types
    Enriched = "enriched"


class ModelType(StrEnum):
    """Known model types."""

    def __repr__(self):
        """Return simple repr."""
        return f"{STD_IMPORT}.ModelType.{self.name}"

    Binary = "binary"
    Delete = "delete"
    Download = "download"
    Insert = "insert"
    Plugin = "plugin"
    Status = "status"
    Retrohunt = "retrohunt"


class FeatureType(StrEnum):
    """Valid feature types."""

    def __repr__(self):
        """Return simple repr."""
        return f"{STD_IMPORT}.FeatureType.{self.name}"

    Integer = "integer"
    Float = "float"
    String = "string"
    Binary = "binary"
    Datetime = "datetime"
    Filepath = "filepath"
    Uri = "uri"


def value_encode(val: VALUE_DECODED) -> str:
    """Encode a feature value into a string."""
    if isinstance(val, bytes):
        val = base64.b64encode(val).decode("ASCII")
    elif isinstance(val, datetime.datetime):
        val = val.isoformat()
    elif issubclass(type(val), str):
        val = str(val).encode(errors="backslashreplace").decode()
    elif isinstance(val, (int, float)):
        val = str(val)
    elif not isinstance(val, str):
        raise Exception(f"cannot encode value {val} with type {type(val)}")
    return val


def value_decode(_type: FeatureType, val: str) -> VALUE_DECODED:
    """With type and encoded value, return decoded value."""
    if _type == FeatureType.Binary:
        return base64.b64decode(val.encode("ASCII"))
    elif _type == FeatureType.Datetime:
        return datetime.datetime.fromisoformat(val.replace("Z", "+00:00"))
    elif _type == FeatureType.Integer:
        return int(val)
    elif _type == FeatureType.Float:
        return float(val)
    elif _type in [FeatureType.String, FeatureType.Filepath, FeatureType.Uri]:
        return val
    else:
        raise ValueError(f"could not decode type {_type} for value {val}")


class FeatureValue(BaseModelStrict):
    """Used by the comms API, in which features are passed as a simple list of (feat_name, value, <other meta>)."""

    # feature name
    name: str
    # type of the value, after decoding
    type: FeatureType
    # encoded value - so that it is json compatible
    value: str
    # label the feature-value with custom text
    label: str | None = None
    # offset in binary content where the feature value occurred
    offset: NonNegativeInt | None = None
    # size of the feature value from the offset where the feature value occurred
    size: NonNegativeInt | None = None

    def decode_value(self) -> VALUE_DECODED:
        """Return decoded value."""
        return value_decode(self.type, self.value)


# NOTE - when this is updated ensure you update the golang equivalent (gosrc/models/data_labels.go)
class DataLabel(StrEnum):
    """Labels used to identify different types of streams being passed throughout Azul."""

    def __repr__(self):
        """Return simple repr."""
        return f"{STD_IMPORT}.DataLabel.{self.name}"

    # Metadata from assemblyline.
    ASSEMBLYLINE = "assemblyline"
    # Full json formatted cape report.
    CAPE_REPORT = "cape_report"
    # Content is the default for a binary that is being examined by Azul.
    CONTENT = "content"
    # A C# call tree holding all the functions a C# application can make.
    CS_CALL_TREE = "cs_call_tree"
    # Decompiled C# content.
    DECOMPILED_CS = "decompiled_cs"
    # Decompiled C content.
    DECOMPILED_C = "decompiled_c"
    # Deobfuscated Javascript content.
    DEOB_JS = "deob_js"
    # Words extracted from a file that can potentially be used for extracting related zip files.
    PASSWORD_DICTIONARY = "password_dictionary"  # nosec B105
    # Network PCAP capture data.
    PCAP = "pcap"
    # Report about a binary found from an external source.
    REPORT = "report"
    # A safe version of an image with all metadata stripped out, ensuring it can't execute.
    SAFE_PNG = "safe_png"
    # A screenshot of actions taken by a malware sandbox (e.g cape).
    SCREENSHOT = "screenshot"
    # Test stream label type just used for testing stream files.
    TEST = "test"
    # Plain text file that provides a large amount of information about a binary.
    TEXT = "text"


class FileInfo(BaseModelStrict):
    """Common file info used between FileInfo, BinaryEvent.Entity and in azul_runner events."""

    # file hard hashes
    sha256: str | None = None
    sha512: str | None = None
    sha1: str | None = None
    md5: str | None = None
    # file fuzzy hashes
    ssdeep: str | None = None
    tlsh: str | None = None
    # bytes in file
    size: NonNegativeInt | None = None
    # azul type identification
    file_format_legacy: str | None = None
    # assemblyline type identification
    file_format: str | None = None
    # expected file type extension
    file_extension: str | None = None

    # generic file identification
    mime: str | None = None
    magic: str | None = None

    def to_input_entity(self) -> "BinaryEvent.Entity":
        """Convert to an input event entity."""
        ret = BinaryEvent.Entity(
            sha256=self.sha256,
            sha512=self.sha512,
            sha1=self.sha1,
            md5=self.md5,
            size=self.size,
            file_format_legacy=self.file_format_legacy,
            file_format=self.file_format,
            tlsh=self.tlsh,
            ssdeep=self.ssdeep,
            mime=self.mime,
            magic=self.magic,
            file_extension=self.file_extension,
            features=[
                FeatureValue(name="file_format", type="string", value=self.file_format or ""),
                FeatureValue(name="file_format_legacy", type="string", value=self.file_format_legacy or ""),
                FeatureValue(name="file_extension", type="string", value=self.file_extension or ""),
                FeatureValue(name="magic", type="string", value=self.magic or ""),
                FeatureValue(name="mime", type="string", value=self.mime or ""),
            ],
        )
        # remove features with no value
        ret.features = [x for x in ret.features if x.value]
        return ret


class Datastream(FileInfo):
    """Struct containing data expected to be returned from dispatcher after uploading a file via the Streams API."""

    # Ensure enum values are made into strings
    model_config = ConfigDict(use_enum_values=True)

    # note - default value should not be changed from '0'
    # only reason default is set is to account for events before version number was introduced
    identify_version: NonNegativeInt = 0

    # content vs pcap vs anything
    # Note - Not in the Base class because BaseClass needs to support BinaryEvent.Entity
    # which can accept data-less submissions
    label: DataLabel

    # if text, the programming language
    language: str | None = None

    def to_input_entity(self) -> "BinaryEvent.Entity":
        """Convert to an input event entity.

        This only makes sense for content files.
        Additional functionality added because the base class needs to support
        azul-runner's BaseEvent type and BinaryEvent.Entity's data-less submissions.
        """
        if self.label != DataLabel.CONTENT:
            raise ValueError("only content fileinfo can become entity")
        ent = super().to_input_entity()
        # Not in BaseClass to support data-less submissions and overrides from azul-runner plugin post-processing.
        ent.datastreams = [self]
        return ent


class Author(BaseModelStrict):
    """Author struct."""

    # plugin or user or something, kinda irrelevant but might be used in metastore
    category: str | None = None
    # plugin name
    name: str
    # version of the plugin, should be a YYYY.MM.DD string but can be anything
    # not evaluated as a date, the newest event.timestamp will win on collisions
    version: str | None = None
    # security string for the author
    security: str | None = None


class PathNode(BaseModelStrict):
    """A node on the source path - the relationship hierarchy to the original source-event."""

    # binary identifier
    sha256: str
    # what event produced this sha256 from parent node
    action: BinaryAction
    # when the event was generated from parent node
    timestamp: Annotated[AwareDatetime, PlainSerializer(lambda v: v.isoformat(), return_type=str)]
    # the plugin that produced the sha256 from parent node
    author: Author
    # name/value pairs describing the relationship from parent node
    relationship: dict[str, str] = {}
    # important binary info
    file_format_legacy: str | None = None
    file_format: str | None = None
    size: NonNegativeInt | None = None
    filename: str | None = None
    language: str | None = None


class Source(BaseModelStrict):
    """Source struct."""

    # original source-event security string
    security: str | None = None
    # name of the source
    name: str
    # original source-event creation time
    timestamp: Annotated[AwareDatetime, PlainSerializer(lambda v: v.isoformat(), return_type=str)]
    # original source-event source references
    # e.g. 'incident number', 'user', 'description'
    references: dict[str, str] = {}
    # source path that describes relationship hierarchy to the original source-event
    path: list[PathNode]
    settings: dict[str, str] = dict()


#
# Azul Events
#


class BaseEvent(BaseModelStrict):
    """Foundation of an event message."""

    # version of the events structure
    model_version: NonNegativeInt
    # deduplication id of this event
    # usually only the latest event with this id will be stored/processed
    # combination of entity id + source info
    kafka_key: str
    # time that this event was produced
    timestamp: Annotated[AwareDatetime, PlainSerializer(lambda v: v.isoformat(), return_type=str)]
    # the plugin/user that produced this event
    author: Author
    # in the base event, the structure of entity is unknown
    entity: Any = None


class BinaryFlags(BaseModelStrict):
    """Flags that can appear on a binary event."""

    # always process, do not look in cache for results
    bypass_cache: bool = False
    # put event in expedite topic for faster analysis
    expedite: bool = False
    # put event in retry topic for another attempt
    retry: bool = False


class BinaryEvent(BaseEvent):
    """Binary event message from plugin execution or data source."""

    class Entity(FileInfo):
        """Details for binary after being processed by a plugin or data source."""

        # key value pairs + metadata
        features: list[FeatureValue] = []
        # information about streams, including 'content' stream
        datastreams: list[Datastream] = []
        # dictionary with non specific key definitions
        info: dict = {}

        def to_file_info(self) -> FileInfo:
            """Convert a BinaryEvent.Entity into a FileInfo object."""
            file_info_dict = self.model_dump()
            del file_info_dict["features"]
            del file_info_dict["datastreams"]
            del file_info_dict["info"]
            return FileInfo(**file_info_dict)

    entity: Entity
    # describes some kind of model variant
    action: BinaryAction
    # information about where the event came from
    source: Source
    # dequeued must be used as id for status messages produced from this event
    dequeued: str | None = None
    # Optional retries variable
    retries: NonNegativeInt | None = None
    # flags control various aspects of processing
    flags: BinaryFlags = BinaryFlags()

    # tracking set by dispatcher after PostEvent
    # track source references
    track_source_references: str = ""
    # track parent-child links
    track_links: list[str] = []
    # track authors in source path
    track_authors: list[str] = []


class StatusEnum(StrEnum):
    """Valid 'status' entries for StatusEvent.Entity.status."""

    def __repr__(self):
        """Return simple repr."""
        return f"{STD_IMPORT}.StatusEnum.{self.name}"

    # Successfully completed
    COMPLETED = "completed"
    # Successfully completed but no features or augmented streams were produced
    COMPLETED_EMPTY = "completed-empty"
    # Successfully completed but errors occurred which means the plugin might not have gotten all data.
    COMPLETED_WITH_ERRORS = "completed-with-errors"
    # Entity not suitable for this plugin (eg wrong size, type, ...)
    OPT_OUT = "opt-out"
    # Plugin heartbeat
    HEARTBEAT = "heartbeat"
    # Event has been dequeued from kafka by dispatcher - not for use by plugins!
    DEQUEUED = "dequeued"

    # Errors
    # Plugin-specific code raised an unhandled exception
    # This is dedicated to errors that can only be resolved by the plugin author
    ERROR_EXCEPTION = "error-exception"
    # Plugin could not communicate with some required service
    ERROR_NETWORK = "error-network"
    # Generic error in plugin harness
    ERROR_RUNNER = "error-runner"
    # Error processing input entity (eg incorrect format, corrupted) - legacy "entity error"
    ERROR_INPUT = "error-input"
    # Plugin returned something that couldn't be understood by the runner
    ERROR_OUTPUT = "error-output"
    # Plugin exceeded its maximum execution time on a sample
    ERROR_TIMEOUT = "error-timeout"
    # Plugin execution was cancelled due to being out of memory
    ERROR_OOM = "error-out-of-memory"


StatusEnumSuccess: set[StatusEnum] = {
    StatusEnum.COMPLETED,
    StatusEnum.COMPLETED_EMPTY,
    StatusEnum.COMPLETED_WITH_ERRORS,
}
StatusEnumErrored: set[StatusEnum] = set(x for x in StatusEnum if x.value.startswith("error-"))
StatusEnumInProgress: set[StatusEnum] = {StatusEnum.HEARTBEAT, StatusEnum.DEQUEUED}


class StatusEvent(BaseEvent):
    """Status event."""

    class Entity(BaseModelStrict):
        """Status message."""

        # input event that was analysed
        input: BinaryEvent
        # description of state of the analysis of the input with the plugin
        status: StatusEnum
        # seconds the plugin has taken to execute
        runtime: float | None = None
        # short error description
        error: str | None = None
        # long error description?
        message: str | None = None
        # embedded results of job (extracted by dispatcher when received)
        results: list[BinaryEvent] = []

    entity: Entity


class PluginEvent(BaseEvent):
    """Author event."""

    class Entity(Author):
        """Entity."""

        class Feature(BaseModelStrict):
            """An output feature for a plugin."""

            # name of the feature
            name: str
            # friendly description of the feature
            desc: str
            # type that feature values will have to match
            type: FeatureType

        def summary(self):
            """Returns self as a simple Author."""
            return Author(name=self.name, category=self.category, version=self.version, security=self.security)

        # description of what this plugin is/does
        description: str | None = None
        # who is the point-of-contact for the plugin
        contact: str | None = None
        # list of features that the plugin can produce
        features: list[Feature] = []
        # plugin specific configuration
        # values should be json encoded strings
        config: dict[str, str] = {}

    entity: Entity


class InsertEvent(BaseEvent):
    """Child manual insert event.

    If a user manually unpacks a file, this event allows for injecting that
    new file into the dispatcher event processing framework.

    Events for the parent binary will be extended with the child history,
    to generate extracted events.
    """

    class Entity(BaseModelStrict):
        """Entity."""

        original_source: str
        parent_sha256: str
        child: BinaryEvent.Entity
        child_history: PathNode

    entity: Entity
    # tracking set by dispatcher after PostEvent
    # track link between parent and child
    track_link: str = ""
    # track author
    track_author: str = ""


class DownloadAction(StrEnum):
    """Valid event types."""

    def __repr__(self):
        """Return simple repr."""
        return f"{STD_IMPORT}.DownloadAction.{self.name}"

    Requested = "requested"
    Success = "success"
    Failed = "failed"


class DownloadEvent(BaseEvent):
    """Download event used by virustotal to download binary events."""

    class Entity(BaseModelStrict):
        """Entity for download event."""

        hash: str
        direct_url: str | None = None
        direct_expiry: Annotated[AwareDatetime, PlainSerializer(lambda v: v.isoformat(), return_type=str)] | None = (
            None
        )
        pcap: bool = False
        category: str | None = None
        category_quota: NonNegativeInt | None = None
        metadata: dict = {}

    entity: Entity
    source: Source
    action: DownloadAction


class DeleteAction(StrEnum):
    """Deletion types."""

    def __repr__(self):
        """Return simple repr."""
        return f"{STD_IMPORT}.DeleteAction.{self.name}"

    submission = "submission"
    link = "link"
    author = "author"


class DeleteEvent(BaseEvent):
    """Deletion event posted to dispatcher to track Kafka events that have been deleted."""

    class DeleteEntity(BaseModelStrict):
        """Deletion of kafka events."""

        class DeleteSubmission(BaseModelStrict):
            """Delete a whole submission."""

            track_source_references: str
            timestamp: str | None = None

        class DeleteLink(BaseModelStrict):
            """Delete a link between binaries."""

            track_link: str

        class DeleteAuthor(BaseModelStrict):
            """Delete all events from an author."""

            track_author: str
            timestamp: str

        class DeleteIDs(BaseModelStrict):
            """Delete all events with matching ids."""

            ids: list[str]

        reason: str
        submission: DeleteSubmission | None = None
        link: DeleteLink | None = None
        author: DeleteAuthor | None = None
        ids: DeleteIDs | None = None

    entity: DeleteEntity
    action: DeleteAction


class HuntState(StrEnum):
    """Strings for the possible states of a retrohunt."""

    SUBMITTED = "submitted"
    STARTING = "starting"
    PARSING_RULES = "parsing-rules"
    SEARCHING_WIDE = "searching-wide"
    SEARCHING_NARROW = "searching-narrow"
    COMPLETED = "completed"
    FAILED = "failed"
    CANCELLED = "cancelled"


class RetrohuntEvent(BaseEvent):
    """Job used by workers when updating retrohunt tasks."""

    class RetrohuntSource(BaseModel):
        """A source specific to retrohunt events."""

        submitter: str | None = None
        security: str | None = None
        timestamp: Annotated[AwareDatetime, PlainSerializer(lambda v: v.isoformat(), return_type=str)]

    class RetrohuntEntity(BaseModel):
        """Retrohunt Data Model."""

        # The Id of the retrohunt submission e.g "hunt_" + now.strftime("%Y%m%d%H%M%S")
        id: str = ""
        # The query type Yara or BigGrep
        search_type: str = ""
        # Search used to query yara or bigrep
        search: str = ""
        # Security string associated with the submission
        security: str | None = None
        # Status of the retrohunt query.
        status: HuntState = HuntState.SUBMITTED
        # User who submitted the query
        submitter: str = "unknown"
        # Time the retrohunt query was submitted.
        submitted_time: Annotated[
            AwareDatetime | None, PlainSerializer(lambda v: v.isoformat() if v else None, return_type=str)
        ] = None
        # Last time the query was updated.
        updated: Annotated[
            AwareDatetime | None, PlainSerializer(lambda v: v.isoformat() if v else None, return_type=str)
        ] = None
        # Time the processing of the query started.
        processing_start: Annotated[
            AwareDatetime | None, PlainSerializer(lambda v: v.isoformat() if v else None, return_type=str)
        ] = None
        # Time the processing of the query finished.
        processing_end: Annotated[
            AwareDatetime | None, PlainSerializer(lambda v: v.isoformat() if v else None, return_type=str)
        ] = None
        # How long the query took to run.
        duration: float | None = None
        logs: str = ""

        # Various counts
        index_searches_total: int = 0
        index_searches_done: int = 0

        rules_parsed_total: int = 0
        rules_parsed_done: int = 0

        atom_count: int = 0
        index_match_count: int = 0

        tool_matches_total: int = 0
        tool_matches_done: int = 0
        tool_match_count: int = 0

        # free form results
        results: dict[str, list[dict]] = {}
        error: str = ""

    class RetrohuntAction(StrEnum):
        """Strings for the possible actions of a retrohunt."""

        Submitted = "submitted"
        Starting = "starting"
        Running = "running"
        Completed = "completed"

    entity: RetrohuntEntity
    action: RetrohuntAction
    source: RetrohuntSource

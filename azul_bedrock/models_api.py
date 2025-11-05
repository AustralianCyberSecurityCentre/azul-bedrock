"""Models representing dispatcher api requests and responses."""

from . import models_network as azm
from .models_network import BaseModelStrict


class GetEventsInfo(BaseModelStrict):
    """Information about the event fetch."""

    filtered: int
    fetched: int
    ready: bool
    paused: bool = False
    consumers_not_ready: str = ""
    filters: dict[str, int] = {}


class GetEventsBinary(BaseModelStrict):
    """GetEvents returned by dispatcher when it responds with binary events."""

    events: list[azm.BinaryEvent]


class GetEventsDelete(BaseModelStrict):
    """GetEvents returned by dispatcher when it responds with binary events."""

    events: list[azm.DeleteEvent]


class GetEventsDownload(BaseModelStrict):
    """GetEvents returned by dispatcher when it responds with binary events."""

    events: list[azm.DownloadEvent]


class GetEventsInsert(BaseModelStrict):
    """GetEvents returned by dispatcher when it responds with binary events."""

    events: list[azm.InsertEvent]


class GetEventsPlugin(BaseModelStrict):
    """GetEvents returned by dispatcher when it responds with binary events."""

    events: list[azm.PluginEvent]


class GetEventsStatus(BaseModelStrict):
    """GetEvents returned by dispatcher when it responds with binary events."""

    events: list[azm.StatusEvent]


class DispatcherData(BaseModelStrict):
    """Data type returned by dispatcher when it receives bytes."""

    data: azm.Datastream
    info: dict = {}


class EventSimulateConsumer(BaseModelStrict):
    """Result of simulating event processing with consumer."""

    name: str
    version: str
    filter_out: bool
    filter_out_trigger: str


class EventSimulate(BaseModelStrict):
    """Results of simulating event processing with consumers."""

    consumers: list[EventSimulateConsumer] = []


# Note linked to models/restapi.go
class DispatcherApiErrorModel(BaseModelStrict):
    """Json error format returned from some of the dispatcher API endpoints."""

    status: str | None = None
    title: str | None = None
    detail: str | None = None


class ResponsePostEventFailure(BaseModelStrict):
    """Info on an invalid event submitted to dispatcher."""

    event: str
    error: str


class ResponsePostEvent(BaseModelStrict):
    """Response for submitting events to dispatcher."""

    total_ok: int
    total_failures: int
    failures: list[ResponsePostEventFailure]
    # optional actual events created after dispatcher enrichment
    # dict as we don't know what the types of the events are
    ok: list[dict] = []

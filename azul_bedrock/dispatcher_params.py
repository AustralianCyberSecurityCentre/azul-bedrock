"""Query parameters for dispatcher api.

gosrc/clients/getevents/parameters.go is source of truth.
"""

from enum import StrEnum

# constants used to determine URL endpoint
GET_EVENTS_ENDPOINT_ACTIVE = "active"
GET_EVENTS_ENDPOINT_PASSIVE = "passive"

# the multipart-form filetype for response info
GET_EVENTS_RESP_INFO = "info"
# the multipart-form filetype for response events
GET_EVENTS_RESP_EVENTS = "events"


class GetEvent(StrEnum):
    """Parameters for GET events endpoint."""

    AvroFormat = "avro-format"  # should be in avro format
    Name = "name"  # name of the client
    Version = "version"  # version of the client
    DeploymentKey = "deployment_key"  # deployment key uniquely identifying the plugin's deployment
    Count = "count"  # max number of events to return
    Deadline = "deadline"  # deadline for retrieving kafka events (seconds)
    IsTask = "is-task"  # client will post completion events
    DenyActions = "d-action"  # deny specified event types
    DenySelf = "d-self"  # filter out events published by this plugin
    RequireExpedite = "r-expedite"  # require the 'expedite' queue of data for binary events
    RequireLive = "r-live"  # require the 'live' queue of data for binary events
    RequireHistoric = "r-historic"  # require the 'historic' queue of data for binary events
    RequireContent = "r-content"  # require events to have underlying binary data
    RequireSources = "r-source"  # filter to only include events from one specific source
    RequireUnderContentSize = "r-under-content-size"  # only keep events that have 'content' stream and below this size
    RequireOverContentSize = "r-over-content-size"  # only keep events that have 'content' stream and above this size
    RequireActions = "r-action"  # allow only specified event types
    # filter data types
    # e.g. "content,executable/windows/pe32,executable/windows/dll32"
    # Multiple stream labels means each event must have all stream labels.
    RequireStreams = "r-streams"


class PostStream(StrEnum):
    """Parameters for POST stream endpoint."""

    SkipIdentify = "skip-identify"  # skips slow identification code - must also supply sha256
    ExpectedSha256 = "expected-sha256"  # this will be cross-checked with the server-side calculated sha256

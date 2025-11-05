package getevents

// parameters available for getevents restapi endpoint on dispatcher

const (
	EndpointActive  = "active"
	EndpointPassive = "passive"
)

const (
	RespInfo   = "info"
	RespEvents = "events"
)

const (
	AvroFormat              = "avro-format"          // should be in avro format
	Name                    = "name"                 // name of the client
	Version                 = "version"              // version of the client
	DeploymentKey           = "deployment_key"       // deployment key uniquely identifying the plugin's deployment
	Count                   = "count"                // max number of events to return
	Deadline                = "deadline"             // deadline for retrieving kafka events (seconds)
	IsTask                  = "is-task"              // client will post completion events
	DenyActions             = "d-action"             // deny specified event types
	DenySelf                = "d-self"               // filter out events published by this plugin
	RequireExpedite         = "r-expedite"           // consume the 'expedite' queue of data for binary events
	RequireLive             = "r-live"               // consume the 'live' queue of data for events
	RequireHistoric         = "r-historic"           // consume the 'historic' queue of data for events
	RequireContent          = "r-content"            // require events to have underlying binary data
	RequireSources          = "r-source"             // filter to only include events from one specific source
	RequireUnderContentSize = "r-under-content-size" // only keep events that have 'content' stream and below this size
	RequireOverContentSize  = "r-over-content-size"  // only keep events that have 'content' stream and above this size
	RequireActions          = "r-action"             // allow only specified event types
	// filter data types
	// e.g. "content,executable/windows/pe32,executable/windows/dll32"
	// Multiple stream labels means each event must have all stream labels.
	RequireStreams = "r-streams"
)

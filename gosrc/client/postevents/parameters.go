package postevents

// parameters available for getevents restapi endpoint on dispatcher

const (
	Model        = "model"         // supplied body is providing events with this model type
	AvroFormat   = "avro-format"   // supplied body is in bulk avro events format
	Sync         = "sync"          // wait for kafka to provide insert receipt before returning
	IncludeOk    = "include_ok"    // response body should contain events after processing
	PausePlugins = "pause_plugins" //  restore/reprocessing operation which should pause all processing
)

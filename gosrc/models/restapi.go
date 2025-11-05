package models

import "github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/events"

// Dispatcher response to a submitted binary blob?
type DataResponse struct {
	Data   events.BinaryEntityDatastream `json:"data,omitempty"`
	Errors []Error                       `json:"errors,omitempty"`
}

// Dispatcher error message (note linked to models_api.py)
type Error struct {
	Status string `json:"status,omitempty"`
	Title  string `json:"title,omitempty"`
	Detail string `json:"detail,omitempty"`
}

type EventResponseInfo struct {
	Filtered          int            `json:"filtered"`
	Fetched           int            `json:"fetched"`
	Ready             bool           `json:"ready"`
	Paused            bool           `json:"paused"`
	ConsumersNotReady string         `json:"consumers_not_ready,omitempty"`
	Filters           map[string]int `json:"filters,omitempty"`
}

// Result of simulating event processing with consumer
type EventSimulateConsumer struct {
	Name             string `json:"name"`
	Version          string `json:"version"`
	FilterOut        bool   `json:"filter_out"`
	FilterOutTrigger string `json:"filter_out_trigger"`
}

// Results of simulating event processing with consumers
type EventSimulate struct {
	Consumers []EventSimulateConsumer `json:"consumers"`
}

type ResponsePostEventFailure struct {
	Event string `json:"event"`
	Error string `json:"error"`
}

// Response from submitting events to dispatcher.
type ResponsePostEvent struct {
	TotalOk       int                        `json:"total_ok"`
	TotalFailures int                        `json:"total_failures"`
	Failures      []ResponsePostEventFailure `json:"failures"`
	Ok            []interface{}              `json:"ok,omitempty"`
}

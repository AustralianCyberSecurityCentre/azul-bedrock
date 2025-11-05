package events

import (
	"encoding/json"
	"fmt"
	"time"
)

const CurrentModelVersion uint32 = 5

// collection of all types of events
type EventStructs interface {
	BinaryEvent | DeleteEvent | InsertEvent | PluginEvent | StatusEvent | DownloadEvent | RetrohuntEvent
}

// Some resource that can publish events.
// The category is usually 'plugin'.
type EventAuthor struct {
	Name     string `json:"name" avro:"name"`
	Version  string `json:"version,omitempty" avro:"version"`
	Category string `json:"category,omitempty" avro:"category"`
	Security string `json:"security,omitempty" avro:"security"`
}

// A link on the path to the original source entity
type EventSourcePathNode struct {
	Author           EventAuthor       `json:"author" avro:"author"`
	Action           BinaryAction      `json:"action" avro:"action"`
	Sha256           string            `json:"sha256" avro:"sha256"`
	Relationship     map[string]string `json:"relationship,omitempty,omitzero" avro:"relationship"`
	Timestamp        time.Time         `json:"timestamp" avro:"timestamp"`
	FileFormatLegacy string            `json:"file_format_legacy,omitempty" avro:"file_format_legacy"`
	FileFormat       string            `json:"file_format,omitempty" avro:"file_format"`
	Size             uint64            `json:"size,omitempty" avro:"size"`
	Filename         string            `json:"filename,omitempty" avro:"filename"`
	Language         string            `json:"language,omitempty" avro:"language"`
}

// Describes how the current event was derived from a base event
type EventSource struct {
	Name       string                `json:"name" avro:"name"`
	References map[string]string     `json:"references,omitempty,omitzero" avro:"references"`
	Security   string                `json:"security,omitempty" avro:"security"`
	Path       []EventSourcePathNode `json:"path" avro:"path"`
	Timestamp  time.Time             `json:"timestamp" avro:"timestamp"`
	Settings   map[string]string     `json:"settings,omitempty,omitzero" avro:"settings"`
}

func (es *EventSource) DeepCopy() (EventSource, error) {
	jsonSource, err := json.Marshal(&es)
	if err != nil {
		return EventSource{}, fmt.Errorf("error when deep copying EventSource Error: %s", err.Error())
	}
	outputSource := EventSource{}
	err = json.Unmarshal(jsonSource, &outputSource)
	return outputSource, err
}

type BaseEvent struct {
	Model        Model
	ModelVersion *uint32
	KafkaKey     *string
	Timestamp    *time.Time
	Author       *EventAuthor
}

type EventInterface interface {
	AvroInterface
	GetBase() *BaseEvent // GetBase return common properties in a known struct.
	CheckValid() error   // CheckValid verify that contents of event are valid.
}

type BulkEventInterface interface {
	AvroInterface
	IsBulk() bool // only exists to make bulks an isolated interface
	GetModel() Model
}

type AvroInterface interface {
	ToAvro() ([]byte, error)    // ToAvro returns the event in avro serialised format.
	FromAvro(data []byte) error // FromAvro loads the event from avro serialised format.
}

package events

import (
	"errors"
	"fmt"
	"time"

	"github.com/goccy/go-json"
)

type RetrohuntAction string

const (
	// Submitted a retrohunt job
	ActionSubmitted RetrohuntAction = "submitted"
	// Retrohunt job has been accepted and is starting
	ActionStarting RetrohuntAction = "starting"
	// Retrohunt job is currently running
	ActionRunning RetrohuntAction = "running"
	// Retrohunt job has finished.
	ActionCompleted RetrohuntAction = "completed"
)

// Retrohunt source which is different to a normal source because it has no path.
type RetrohuntSource struct {
	Submitter string    `json:"submitter,omitempty" avro:"submitter"`
	Security  string    `json:"security,omitempty" avro:"security"`
	Timestamp time.Time `json:"timestamp" avro:"timestamp"`
}

// generic event with no special properties and unknown entity format
type RetrohuntEvent struct {
	ModelVersion uint32          `json:"model_version,omitempty" avro:"model_version"`
	KafkaKey     string          `json:"kafka_key,omitempty" avro:"kafka_key"`
	Timestamp    time.Time       `json:"timestamp" avro:"timestamp"`
	Author       EventAuthor     `json:"author" avro:"author"`
	Entity       json.RawMessage `json:"entity" avro:"entity"`
	Action       RetrohuntAction `json:"action,omitempty" avro:"action"`
	Source       RetrohuntSource `json:"source" avro:"source"`
}

type BulkRetrohuntEvent struct {
	ModelVersion uint32            `json:"model_version,omitempty" avro:"model_version"`
	Events       []*RetrohuntEvent `json:"events" avro:"events"`
}

// must be supplied by client
func (evs *BulkRetrohuntEvent) GetModel() Model {
	return ModelRetrohunt
}

func (evs *BulkRetrohuntEvent) IsBulk() bool {
	return true
}

func (evs *BulkRetrohuntEvent) GetModelVersion() uint32 {
	return evs.ModelVersion
}

func (evs *BulkRetrohuntEvent) SetModelVersion(newVersion uint32) {
	evs.ModelVersion = newVersion
	if evs.Events == nil {
		return
	}
	// Set the same model version for all the collected Events
	for _, curEv := range evs.Events {
		if curEv != nil {
			curEv.SetModelVersion(newVersion)
		}
	}
}

func (evs *BulkRetrohuntEvent) ToAvro() ([]byte, error) {
	return GenericToAvro(evs, SchemaBulkRetrohunt)
}

func (evs *BulkRetrohuntEvent) FromAvro(data []byte) error {
	err := GenericFromAvro(evs, data, SchemaBulkRetrohunt)
	// Avro sometimes drops lots of data but doesn't error when un-marshalling bulk schemas.
	if len(evs.Events) == 0 && len(data) > LENGTH_OF_BULK_HEADER_INFO {
		return fmt.Errorf("bulk event was not properly un-marshalled by avro")
	}
	return err
}

func (b *RetrohuntEvent) GetBase() *BaseEvent {
	return &BaseEvent{
		Model:        ModelRetrohunt,
		ModelVersion: &b.ModelVersion,
		KafkaKey:     &b.KafkaKey,
		Timestamp:    &b.Timestamp,
		Author:       &b.Author,
	}
}

// CheckValid returns errors in an event
func (b *RetrohuntEvent) CheckValid() error {
	if len(b.Author.Name) == 0 {
		return errors.New("event is missing 'author' field")
	}
	return nil
}

func (ev *RetrohuntEvent) GetModelVersion() uint32 {
	return ev.ModelVersion
}

func (ev *RetrohuntEvent) SetModelVersion(newVersion uint32) {
	ev.ModelVersion = newVersion
}

func (ev *RetrohuntEvent) ToAvro() ([]byte, error) {
	return GenericToAvro(ev, SchemaRetrohunt)
}

func (ev *RetrohuntEvent) FromAvro(data []byte) error {
	return GenericFromAvro(ev, data, SchemaRetrohunt)
}

// updates entity with the json compatible struct
func (ev *RetrohuntEvent) StoreEntity(entity any) error {
	var err error
	ev.Entity, err = json.Marshal(&entity)
	return err
}

func (ev *RetrohuntEvent) LoadEntity(entity any) error {
	return json.Unmarshal(ev.Entity, &entity)
}

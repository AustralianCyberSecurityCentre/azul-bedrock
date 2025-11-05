package events

import (
	"errors"
	"fmt"
	"time"
)

// Entity field for status event (can only trigger from and produce binary events)
type StatusEntity struct {
	Input   BinaryEvent   `json:"input" avro:"input"`
	Status  string        `json:"status" avro:"status"`
	RunTime float64       `json:"runtime,omitempty" avro:"runtime"`
	Error   string        `json:"error,omitempty" avro:"error"`
	Message string        `json:"message,omitempty" avro:"message"`
	Results []BinaryEvent `json:"results,omitempty,omitzero" avro:"results"`
}

type StatusEvent struct {
	ModelVersion uint32       `json:"model_version,omitempty" avro:"model_version"`
	KafkaKey     string       `json:"kafka_key,omitempty" avro:"kafka_key"`
	Timestamp    time.Time    `json:"timestamp" avro:"timestamp"`
	Author       EventAuthor  `json:"author" avro:"author"`
	Entity       StatusEntity `json:"entity" avro:"entity"`
}

type BulkStatusEvent struct {
	ModelVersion uint32         `json:"model_version,omitempty" avro:"model_version"`
	Events       []*StatusEvent `json:"events" avro:"events"`
}

func (evs *BulkStatusEvent) GetModel() Model {
	return ModelStatus
}

func (evs *BulkStatusEvent) IsBulk() bool {
	return true
}

func (evs *BulkStatusEvent) GetModelVersion() uint32 {
	return evs.ModelVersion
}

func (evs *BulkStatusEvent) SetModelVersion(newVersion uint32) {
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

func (evs *BulkStatusEvent) ToAvro() ([]byte, error) {
	return GenericToAvro(evs, SchemaBulkStatus)
}

func (evs *BulkStatusEvent) FromAvro(data []byte) error {
	err := GenericFromAvro(evs, data, SchemaBulkStatus)
	// Avro sometimes drops lots of data but doesn't error when un-marshalling bulk schemas.
	if len(evs.Events) == 0 && len(data) > LENGTH_OF_BULK_HEADER_INFO {
		return fmt.Errorf("bulk event was not properly un-marshalled by avro")
	}
	return err
}

func (b *StatusEvent) GetBase() *BaseEvent {
	return &BaseEvent{
		Model:        ModelStatus,
		ModelVersion: &b.ModelVersion,
		KafkaKey:     &b.KafkaKey,
		Timestamp:    &b.Timestamp,
		Author:       &b.Author,
	}
}

func (b *StatusEvent) CheckValid() error {
	if len(b.Author.Name) == 0 {
		return errors.New("event is missing 'author' field")
	}

	// The input binary event is not CheckValid()'ed, as it is likely to have
	// been 'minified' to remove unnecessary data.
	if len(b.Entity.Input.Dequeued) == 0 {
		return errors.New("entity status missing entity.input.dequeued")
	}
	for i, ev := range b.Entity.Results {
		err := ev.CheckValid()
		if err != nil {
			return fmt.Errorf("status result %d is not valid binary event - %w", i, err)
		}
	}
	return nil
}

func (ev *StatusEvent) GetModelVersion() uint32 {
	return ev.ModelVersion
}

// Set model version for status event and all nested events.
func (ev *StatusEvent) SetModelVersion(newVersion uint32) {
	ev.ModelVersion = newVersion
	ev.Entity.Input.SetModelVersion(newVersion)
	for i := range ev.Entity.Results {
		ev.Entity.Results[i].SetModelVersion(newVersion)
	}
}

func (ev *StatusEvent) ToAvro() ([]byte, error) {
	return GenericToAvro(ev, SchemaStatus)
}

func (ev *StatusEvent) FromAvro(data []byte) error {
	return GenericFromAvro(ev, data, SchemaStatus)
}

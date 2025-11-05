package events

import (
	"errors"
	"fmt"
	"time"

	"github.com/goccy/go-json"
)

const (
	DeleteActionSubmission string = "submission"
	DeleteActionAuthor     string = "author"
	DeleteActionLink       string = "link"
	DeleteActionIDs        string = "ids"
)

type DeleteEntitySubmission struct {
	TrackSourceReferences string    `json:"track_source_references,omitempty" avro:"track_source_references"`
	Timestamp             time.Time `json:"timestamp,omitempty" avro:"timestamp"`
}

type DeleteEntityLink struct {
	TrackLink string `json:"track_link,omitempty" avro:"track_link"`
}

type DeleteEntityAuthor struct {
	TrackAuthor string    `json:"track_author,omitempty" avro:"track_author"`
	Timestamp   time.Time `json:"timestamp,omitempty" avro:"timestamp"`
}

type DeleteEntityIDs struct {
	IDs []string `json:"ids,omitempty,omitzero" avro:"ids"`
}

// Entity struct for deletion event
type DeleteEntity struct {
	Reason     string                 `json:"reason,omitempty" avro:"reason"`
	Submission DeleteEntitySubmission `json:"submission,omitempty" avro:"submission"`
	Author     DeleteEntityAuthor     `json:"author,omitempty" avro:"author"`
	Link       DeleteEntityLink       `json:"link,omitempty" avro:"link"`
	IDs        DeleteEntityIDs        `json:"ids,omitempty" avro:"ids"`
}

type DeleteEvent struct {
	ModelVersion uint32       `json:"model_version,omitempty" avro:"model_version"`
	KafkaKey     string       `json:"kafka_key,omitempty" avro:"kafka_key"`
	Timestamp    time.Time    `json:"timestamp" avro:"timestamp"`
	Author       EventAuthor  `json:"author" avro:"author"`
	Entity       DeleteEntity `json:"entity" avro:"entity"`
	Action       string       `json:"action" avro:"action"` // the deletion type we expect
}

type BulkDeleteEvent struct {
	ModelVersion uint32         `json:"model_version,omitempty" avro:"model_version"`
	Events       []*DeleteEvent `json:"events" avro:"events"`
}

func (evs *BulkDeleteEvent) GetModel() Model {
	return ModelDelete
}

func (evs *BulkDeleteEvent) IsBulk() bool {
	return true
}

func (evs *BulkDeleteEvent) GetModelVersion() uint32 {
	return evs.ModelVersion
}

func (evs *BulkDeleteEvent) SetModelVersion(newVersion uint32) {
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

func (evs *BulkDeleteEvent) ToAvro() ([]byte, error) {
	return GenericToAvro(evs, SchemaBulkDelete)
}

func (evs *BulkDeleteEvent) FromAvro(data []byte) error {
	err := GenericFromAvro(evs, data, SchemaBulkDelete)
	// Avro sometimes drops lots of data but doesn't error when un-marshalling bulk schemas.
	if len(evs.Events) == 0 && len(data) > LENGTH_OF_BULK_HEADER_INFO {
		return fmt.Errorf("bulk event was not properly un-marshalled by avro")
	}
	return err
}

func (b *DeleteEvent) GetBase() *BaseEvent {
	return &BaseEvent{
		Model:        ModelDelete,
		ModelVersion: &b.ModelVersion,
		KafkaKey:     &b.KafkaKey,
		Timestamp:    &b.Timestamp,
		Author:       &b.Author,
	}
}

// CheckValid returns errors in an event
func (b *DeleteEvent) CheckValid() error {
	if len(b.Author.Name) == 0 {
		return errors.New("event is missing 'author' field")
	}
	return nil
}

// prevent empty timestamp in json
func (ms *DeleteEntitySubmission) MarshalJSON() ([]byte, error) {
	type Alias DeleteEntitySubmission
	if ms.Timestamp.IsZero() {
		return json.Marshal(&struct {
			Timestamp int64 `json:"timestamp,omitempty"`
			*Alias
		}{
			Timestamp: 0,
			Alias:     (*Alias)(ms),
		})
	} else {
		return json.Marshal(&struct {
			*Alias
		}{
			Alias: (*Alias)(ms),
		})
	}
}

// prevent empty timestamp in json
func (ms *DeleteEntityAuthor) MarshalJSON() ([]byte, error) {
	type Alias DeleteEntityAuthor
	if ms.Timestamp.IsZero() {
		return json.Marshal(&struct {
			Timestamp int64 `json:"timestamp,omitempty"`
			*Alias
		}{
			Timestamp: 0,
			Alias:     (*Alias)(ms),
		})
	} else {
		return json.Marshal(&struct {
			*Alias
		}{
			Alias: (*Alias)(ms),
		})
	}
}

func (ev *DeleteEvent) GetModelVersion() uint32 {
	return ev.ModelVersion
}

func (ev *DeleteEvent) SetModelVersion(newVersion uint32) {
	ev.ModelVersion = newVersion
}

func (ev *DeleteEvent) ToAvro() ([]byte, error) {
	return GenericToAvro(ev, SchemaDelete)
}

func (ev *DeleteEvent) FromAvro(data []byte) error {
	return GenericFromAvro(ev, data, SchemaDelete)
}

package events

import (
	"errors"
	"fmt"
	"time"
)

// Entity struct for insertion event
type InsertEntity struct {
	OriginalSource string              `json:"original_source" avro:"original_source"`
	ParentSha256   string              `json:"parent_sha256" avro:"parent_sha256"`
	Child          BinaryEntity        `json:"child" avro:"child"`
	ChildHistory   EventSourcePathNode `json:"child_history" avro:"child_history"`
}

type InsertEvent struct {
	ModelVersion uint32       `json:"model_version,omitempty" avro:"model_version"`
	KafkaKey     string       `json:"kafka_key,omitempty" avro:"kafka_key"`
	Timestamp    time.Time    `json:"timestamp" avro:"timestamp"`
	Author       EventAuthor  `json:"author" avro:"author"`
	Entity       InsertEntity `json:"entity" avro:"entity"`
	// tracking set by dispatcher once received
	TrackLink   string `json:"track_link,omitempty" avro:"track_link"`     // tracks the link that will be created
	TrackAuthor string `json:"track_author,omitempty" avro:"track_author"` // tracks the author that will be created
}

type BulkInsertEvent struct {
	ModelVersion uint32         `json:"model_version,omitempty" avro:"model_version"`
	Events       []*InsertEvent `json:"events" avro:"events"`
}

func (evs *BulkInsertEvent) GetModel() Model {
	return ModelInsert
}

func (evs *BulkInsertEvent) IsBulk() bool {
	return true
}

func (evs *BulkInsertEvent) GetModelVersion() uint32 {
	return evs.ModelVersion
}

func (evs *BulkInsertEvent) SetModelVersion(newVersion uint32) {
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

func (evs *BulkInsertEvent) ToAvro() ([]byte, error) {
	return GenericToAvro(evs, SchemaBulkInsert)
}

func (evs *BulkInsertEvent) FromAvro(data []byte) error {
	err := GenericFromAvro(evs, data, SchemaBulkInsert)
	// Avro sometimes drops lots of data but doesn't error when un-marshalling bulk schemas.
	if len(evs.Events) == 0 && len(data) > LENGTH_OF_BULK_HEADER_INFO {
		return fmt.Errorf("bulk event was not properly un-marshalled by avro")
	}
	return err
}

func (b *InsertEvent) GetBase() *BaseEvent {
	return &BaseEvent{
		Model:        ModelInsert,
		ModelVersion: &b.ModelVersion,
		KafkaKey:     &b.KafkaKey,
		Timestamp:    &b.Timestamp,
		Author:       &b.Author,
	}
}

// CheckValid returns errors in an event
func (b *InsertEvent) CheckValid() error {
	if len(b.Author.Name) == 0 {
		return errors.New("event is missing 'author' field")
	}
	for _, curData := range b.Entity.Child.Datastreams {
		if !IsDataLabelValid(curData.Label) {
			return fmt.Errorf("label for the inserted child '%s' is invalid", curData.Label)
		}
	}
	if len(b.Entity.ParentSha256) == 0 {
		return errors.New("entity insert missing parent_sha256")
	}
	if len(b.Entity.ChildHistory.Sha256) == 0 {
		return errors.New("entity insert missing child_history.sha256")
	}
	if len(b.Entity.ChildHistory.Action) == 0 {
		return errors.New("entity insert missing child_history.action")
	}
	if len(b.Entity.ChildHistory.Author.Name) == 0 {
		return errors.New("entity insert missing child_history.author.name")
	}
	if len(b.Entity.Child.FileFormatLegacy) == 0 {
		return errors.New("entity insert missing child.file_format_legacy")
	}
	return nil
}

func (b *InsertEvent) UpdateTrackingFields() error {
	b.TrackAuthor = genTrackAuthor(&b.Entity.ChildHistory)
	b.TrackLink = genTrackLink(&b.Entity.ChildHistory, b.Entity.ParentSha256)
	return nil
}

func (ev *InsertEvent) GetModelVersion() uint32 {
	return ev.ModelVersion
}

func (ev *InsertEvent) SetModelVersion(newVersion uint32) {
	ev.ModelVersion = newVersion
}

func (ev *InsertEvent) ToAvro() ([]byte, error) {
	return GenericToAvro(ev, SchemaInsert)
}

func (ev *InsertEvent) FromAvro(data []byte) error {
	return GenericFromAvro(ev, data, SchemaInsert)
}

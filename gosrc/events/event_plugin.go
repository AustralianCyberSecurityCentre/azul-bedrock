package events

import (
	"errors"
	"fmt"
	"time"
)

// Plugins must describe list of features they can publish
type PluginEntityFeature struct {
	Name        string      `json:"name" avro:"name"`
	Type        FeatureType `json:"type" avro:"type"`
	Description string      `json:"desc,omitempty" avro:"desc"`
	Tags        []string    `json:"tags,omitempty" avro:"tags"`
}

// Some resource that can publish events.
// The category is usually 'plugin'.
type PluginEntity struct {
	Name        string                `json:"name" avro:"name"`
	Version     string                `json:"version,omitempty" avro:"version"`
	Category    string                `json:"category,omitempty" avro:"category"`
	Contact     string                `json:"contact,omitempty" avro:"contact"`
	Description string                `json:"description,omitempty" avro:"description"`
	Features    []PluginEntityFeature `json:"features,omitempty" avro:"features"`
	Security    string                `json:"security,omitempty" avro:"security"`
	Config      map[string]string     `json:"config,omitempty" avro:"config"`
}

type PluginEvent struct {
	ModelVersion uint32       `json:"model_version,omitempty" avro:"model_version"`
	KafkaKey     string       `json:"kafka_key,omitempty" avro:"kafka_key"`
	Timestamp    time.Time    `json:"timestamp" avro:"timestamp"`
	Author       EventAuthor  `json:"author" avro:"author"`
	Entity       PluginEntity `json:"entity" avro:"entity"`
}

type BulkPluginEvent struct {
	ModelVersion uint32         `json:"model_version,omitempty" avro:"model_version"`
	Events       []*PluginEvent `json:"events" avro:"events"`
}

func (evs *BulkPluginEvent) GetModel() Model {
	return ModelPlugin
}

func (evs *BulkPluginEvent) IsBulk() bool {
	return true
}

func (evs *BulkPluginEvent) GetModelVersion() uint32 {
	return evs.ModelVersion
}

func (evs *BulkPluginEvent) SetModelVersion(newVersion uint32) {
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

func (evs *BulkPluginEvent) ToAvro() ([]byte, error) {
	return GenericToAvro(evs, SchemaBulkPlugin)
}

func (evs *BulkPluginEvent) FromAvro(data []byte) error {
	err := GenericFromAvro(evs, data, SchemaBulkPlugin)
	// Avro sometimes drops lots of data but doesn't error when un-marshalling bulk schemas.
	if len(evs.Events) == 0 && len(data) > LENGTH_OF_BULK_HEADER_INFO {
		return fmt.Errorf("bulk event was not properly un-marshalled by avro")
	}
	return err
}

// Summary summarises EntityAuthor into EventAuthor
func (a *PluginEntity) Summary() EventAuthor {
	return EventAuthor{Name: a.Name, Version: a.Version, Category: a.Category, Security: a.Security}
}

func (b *PluginEvent) GetBase() *BaseEvent {
	return &BaseEvent{
		Model:        ModelPlugin,
		ModelVersion: &b.ModelVersion,
		KafkaKey:     &b.KafkaKey,
		Timestamp:    &b.Timestamp,
		Author:       &b.Author,
	}
}

// CheckValid returns errors in an event
func (b *PluginEvent) CheckValid() error {
	if len(b.Author.Name) == 0 {
		return errors.New("event is missing 'author' field")
	}
	return nil
}

func (ev *PluginEvent) GetModelVersion() uint32 {
	return ev.ModelVersion
}

func (ev *PluginEvent) SetModelVersion(newVersion uint32) {
	ev.ModelVersion = newVersion
}

func (ev *PluginEvent) ToAvro() ([]byte, error) {
	return GenericToAvro(ev, SchemaPlugin)
}

func (ev *PluginEvent) FromAvro(data []byte) error {
	return GenericFromAvro(ev, data, SchemaPlugin)
}

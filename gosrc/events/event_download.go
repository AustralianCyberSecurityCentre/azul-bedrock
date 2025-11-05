package events

import (
	"errors"
	"fmt"
	"time"

	"github.com/goccy/go-json"
)

// Enumeration of entity subtypes that are available
type DownloadAction string

const (
	// virustotal plugin
	DownloadActionRequested DownloadAction = "requested"
	// for quota tracking
	DownloadActionSuccess DownloadAction = "success"
	// unable to download requested file
	DownloadActionFailed DownloadAction = "failed"
)

// Entity struct for download event
type DownloadEntity struct {
	Hash          string          `json:"hash" avro:"hash"`
	DirectURL     string          `json:"direct_url,omitempty" avro:"direct_url"`
	DirectExpiry  time.Time       `json:"direct_expiry,omitempty" avro:"direct_expiry"`
	PCAP          bool            `json:"pcap,omitempty" avro:"pcap"`
	Category      string          `json:"category,omitempty" avro:"category"`
	CategoryQuota uint32          `json:"category_quota,omitempty" avro:"category_quota"`
	Metadata      json.RawMessage `json:"metadata,omitempty" avro:"metadata"`
}

type DownloadEvent struct {
	ModelVersion uint32         `json:"model_version,omitempty" avro:"model_version"`
	KafkaKey     string         `json:"kafka_key,omitempty" avro:"kafka_key"`
	Timestamp    time.Time      `json:"timestamp" avro:"timestamp"`
	Author       EventAuthor    `json:"author" avro:"author"`
	Source       EventSource    `json:"source" avro:"source"`
	Action       DownloadAction `json:"action" avro:"action"`
	Entity       DownloadEntity `json:"entity" avro:"entity"`
}

type BulkDownloadEvent struct {
	ModelVersion uint32           `json:"model_version,omitempty" avro:"model_version"`
	Events       []*DownloadEvent `json:"events" avro:"events"`
}

func (evs *BulkDownloadEvent) GetModel() Model {
	return ModelDownload
}

func (evs *BulkDownloadEvent) IsBulk() bool {
	return true
}

func (evs *BulkDownloadEvent) GetModelVersion() uint32 {
	return evs.ModelVersion
}

func (evs *BulkDownloadEvent) SetModelVersion(newVersion uint32) {
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

func (evs *BulkDownloadEvent) ToAvro() ([]byte, error) {
	return GenericToAvro(evs, SchemaBulkDownload)
}

func (evs *BulkDownloadEvent) FromAvro(data []byte) error {
	err := GenericFromAvro(evs, data, SchemaBulkDownload)
	// Avro sometimes drops lots of data but doesn't error when un-marshalling bulk schemas.
	if len(evs.Events) == 0 && len(data) > LENGTH_OF_BULK_HEADER_INFO {
		return fmt.Errorf("bulk event was not properly un-marshalled by avro")
	}
	return err
}

func (b *DownloadEvent) GetBase() *BaseEvent {
	return &BaseEvent{
		Model:        ModelDownload,
		ModelVersion: &b.ModelVersion,
		KafkaKey:     &b.KafkaKey,
		Timestamp:    &b.Timestamp,
		Author:       &b.Author,
	}
}

// CheckValid returns errors in an event
func (b *DownloadEvent) CheckValid() error {
	if len(b.Author.Name) == 0 {
		return errors.New("event is missing 'author' field")
	}
	return nil
}

// prevent empty timestamp in json
func (ms *DownloadEntity) MarshalJSON() ([]byte, error) {
	type Alias DownloadEntity
	if ms.DirectExpiry.IsZero() {
		return json.Marshal(&struct {
			DirectExpiry int64 `json:"direct_expiry,omitempty"`
			*Alias
		}{
			DirectExpiry: 0,
			Alias:        (*Alias)(ms),
		})
	} else {
		return json.Marshal(&struct {
			*Alias
		}{
			Alias: (*Alias)(ms),
		})
	}
}

func (ev *DownloadEvent) GetModelVersion() uint32 {
	return ev.ModelVersion
}

func (ev *DownloadEvent) SetModelVersion(newVersion uint32) {
	ev.ModelVersion = newVersion
}

func (ev *DownloadEvent) ToAvro() ([]byte, error) {
	return GenericToAvro(ev, SchemaDownload)
}

func (ev *DownloadEvent) FromAvro(data []byte) error {
	return GenericFromAvro(ev, data, SchemaDownload)
}

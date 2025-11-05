// helpers for generating test events
package msginflight

import (
	"fmt"
	"log"
	"time"

	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/events"
)

func GenBinaryStream(id string) events.BinaryEntityDatastream {
	return events.BinaryEntityDatastream{
		IdentifyVersion:  1,
		Label:            events.DataLabelContent,
		Size:             1111,
		Sha512:           "data.Sha512",
		Sha256:           id,
		Sha1:             "data.Sha1",
		Md5:              "data.Md5",
		FileFormatLegacy: "data.FileFormatLegacy",
	}
}

func genEntityBinary(id string, noData bool) events.BinaryEntity {
	if id == "" {
		id = "data.Sha256"
	}
	b := events.BinaryEntity{
		Sha512:           "data.Sha512",
		Sha256:           id,
		Sha1:             "data.Sha1",
		Md5:              "data.Md5",
		Size:             1111,
		FileFormatLegacy: "data.FileFormatLegacy",
		Features: []events.BinaryEntityFeature{
			{Name: "file_format_legacy", Type: "string", Value: "data.FileFormatLegacy"},
			{Name: "magic", Type: "string", Value: "data.Magic"},
			{Name: "mime", Type: "string", Value: "data.Mime"},
		},
		Info: []byte(`{"test": 55}`),
	}
	if !noData {
		b.Datastreams = []events.BinaryEntityDatastream{
			{
				IdentifyVersion:  1,
				Label:            events.DataLabelContent,
				Size:             1111,
				Sha512:           "data.Sha512",
				Sha256:           id,
				Sha1:             "data.Sha1",
				Md5:              "data.Md5",
				FileFormatLegacy: "data.FileFormatLegacy",
			},
		}
	}
	return b
}

func genericTime() time.Time {
	now, err := time.Parse(time.RFC3339, "2023-03-02T00:37:18Z")
	if err != nil {
		log.Fatalf("%v", err)
	}
	return now
}

func genPluginEntity() events.PluginEntity {
	return events.PluginEntity{
		Name:        "FilePublisher",
		Version:     "1.0",
		Contact:     "azul@asd.gov.au",
		Category:    "Loader",
		Description: "Local file publishing plugin used as code example.",
		Features: []events.PluginEntityFeature{
			{Name: "filename", Type: "filepath", Description: "Filesystem path the file was sourced from", Tags: []string{"source"}},
			{Name: "file_format_legacy", Type: "string", Description: "System normalised file type format"},
			{Name: "magic", Type: "string", Description: "File magic description string"},
			{Name: "mime", Type: "string", Description: "File magic mime-type label"},
		},
	}

}

type BC struct {
	ID             string
	Entity         *events.BinaryEntity
	Action         events.BinaryAction
	Flags          *events.BinaryFlags
	PresetFeatures int
	NoData         bool
	Source         string
}

func GenEventBinary(opt *BC) *events.BinaryEvent {
	if opt == nil {
		opt = &BC{}
	}
	if opt.ID == "" {
		opt.ID = "data.Sha256"
	}
	if opt.Entity == nil {
		tmp := genEntityBinary(opt.ID, opt.NoData)
		opt.Entity = &tmp
	}
	if opt.Action == "" {
		opt.Action = events.ActionSourced
	}
	if opt.Flags == nil {
		opt.Flags = &events.BinaryFlags{}
	}
	if len(opt.Source) == 0 {
		opt.Source = "testing"
	}
	features := []events.BinaryEntityFeature{}
	if opt.PresetFeatures > 0 {
		for i := range opt.PresetFeatures {
			features = append(features, events.BinaryEntityFeature{
				Name:  "myfeature",
				Value: fmt.Sprintf("v-%d", i),
				Type:  "string",
			})
		}
		opt.Entity.Features = features
	}
	author := genPluginEntity()
	now := genericTime()
	ob := events.BinaryEvent{
		Author:       author.Summary(),
		ModelVersion: events.CurrentModelVersion,
		Timestamp:    now,
		Action:       opt.Action,
		Flags:        *opt.Flags,
		Entity:       *opt.Entity,
		Source: events.EventSource{
			Name:       opt.Source,
			Security:   "RESTRICTED",
			References: map[string]string{},
			Path: []events.EventSourcePathNode{
				{
					Author:           author.Summary(),
					Action:           opt.Action,
					Sha256:           opt.Entity.Sha256,
					Timestamp:        now,
					Filename:         "file.exe",
					FileFormatLegacy: "PE32+ executable",
					Size:             1111,
				},
			},
			Timestamp: now,
		},
	}
	return &ob
}

// generate an event with 2 binary events contained within
func GenEventStatus(id string) *events.StatusEvent {
	srcEvent := GenEventBinary(&BC{})
	// any input event triggered a plugin, so would already have tracking information
	err := srcEvent.UpdateTrackingFields()
	if err != nil {
		panic(err)
	}
	srcEvent.Dequeued = "dequeued-test"
	author := genPluginEntity()
	now := time.Now().Truncate(time.Microsecond)
	ev := events.StatusEvent{
		ModelVersion: events.CurrentModelVersion,
		Author:       author.Summary(),
		Timestamp:    now,
		Entity: events.StatusEntity{
			Status: events.StatusTypeCompleted,
			Input:  *srcEvent,
			Results: []events.BinaryEvent{
				*GenEventBinary(&BC{ID: id, Action: events.ActionEnriched, NoData: true}),
				*GenEventBinary(&BC{ID: id, Action: events.ActionExtracted}),
			},
		},
	}
	return &ev
}

func GenEventPlugin(id string) *events.PluginEvent {
	// Generate a basic plugin for testing.
	author := genPluginEntity()
	ev := events.PluginEvent{
		Author:       author.Summary(),
		ModelVersion: events.CurrentModelVersion,
		KafkaKey:     id,
		Entity:       genPluginEntity(),
	}
	return &ev
}

// GenEventInsert generates an insert event
func GenEventInsert(id string) *events.InsertEvent {
	author := genPluginEntity()
	ev := events.InsertEvent{
		Author:       author.Summary(),
		ModelVersion: events.CurrentModelVersion,
		KafkaKey:     id,
		Entity: events.InsertEntity{
			OriginalSource: id,
			ParentSha256:   "sha2561",
			Child: events.BinaryEntity{
				FileFormatLegacy: "exe",
			},
			ChildHistory: events.EventSourcePathNode{
				Sha256: "sha2562",
				Action: "extracted",
				Author: events.EventAuthor{
					Name:    "myauthor",
					Version: "1.2",
				},
			},
		},
	}
	return &ev
}

func GenEventDelete(id string) *events.DeleteEvent {
	author := genPluginEntity()
	ev := events.DeleteEvent{
		Author:       author.Summary(),
		ModelVersion: events.CurrentModelVersion,
		Entity: events.DeleteEntity{
			Reason: "junk",
			IDs:    events.DeleteEntityIDs{IDs: []string{"a", "b"}},
		},
		Action: events.DeleteActionIDs,
	}
	return &ev
}

func GetDequeuedBinaryEvent(id string, dequeued_id string) *events.BinaryEvent {
	author := genPluginEntity()
	top := events.BinaryEvent{
		ModelVersion: events.CurrentModelVersion,
		Author:       author.Summary(),
		Action:       events.ActionEnriched,
		Source:       events.EventSource{Name: "truck"},
		Entity: events.BinaryEntity{
			FileFormatLegacy: "ZIP",
			FileFormat:       "compression/zip",
			Sha256:           id,
			Size:             5,
		},
		Dequeued: dequeued_id,
	}
	return &top
}

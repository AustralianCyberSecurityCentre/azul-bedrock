package plugin

import (
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"slices"
	"testing"

	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/events"
	"github.com/sanity-io/litter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const MAX_RAW_STREAM_SIZE = uint64(1 * 1024 * 1024) // 1MB (arbitrary)

type TestBinaryEntityFeature struct {
	Value  string `json:"value" avro:"value"`
	Label  string `json:"label,omitempty" avro:"label"`
	Size   uint64 `json:"size,omitempty" avro:"size"`
	Offset uint64 `json:"offset,omitempty" avro:"offset"`
}

type TestJobEvent struct {
	ParentSha256   string                               `json:"parent_sha256,omitempty"`
	ChildrenSha256 []string                             `json:"children_sha256,omitempty,omitzero"`
	Features       map[string][]TestBinaryEntityFeature `json:"features,omitempty,omitzero"`
	Info           string                               `json:"info,omitempty"`
	// Augmented streams with their raw bytes if requested.
	AugmentedStreams []ResultStream `json:"augmented_streams,omitempty,omitzero"`
}

func compareAugmentedStreamsSort(a ResultStream, b ResultStream) int {
	sha256Compare := cmp.Compare(a.Sha256, b.Sha256)
	if cmp.Compare(a.Sha256, b.Sha256) != 0 {
		return sha256Compare
	}
	return cmp.Compare(a.Label, b.Label)
}

func NewTestJobEvent(jobEvent *JobEvent, testJobResultOptions *TestJobResultOptions) *TestJobEvent {
	// Create defaults
	parentSha256 := ""
	childrenSha256 := []string{}
	info := ""
	augmentedStreams := []ResultStream{}

	if jobEvent.parent != nil {
		resultStream, err := jobEvent.parent.getContentStream()
		// If there is no error assign the parent sha256 otherwise leave it as an empty string.
		if err == nil {
			parentSha256 = resultStream.Sha256
		}
	}

	for _, childEvent := range jobEvent.children {
		childStream, err := childEvent.getContentStream()
		if err == nil {
			childrenSha256 = append(childrenSha256, childStream.Sha256)
		}
	}
	// Sort for consistency
	slices.Sort(childrenSha256)
	// Sort the features for consistency
	slices.SortFunc(jobEvent.features, events.CompareFeaturesSort)

	features := map[string][]TestBinaryEntityFeature{}
	for _, jobFeat := range jobEvent.features {
		existingList, ok := features[jobFeat.Name]
		if !ok {
			existingList = []TestBinaryEntityFeature{}
		}
		features[jobFeat.Name] = append(existingList, TestBinaryEntityFeature{
			Label:  jobFeat.Label,
			Value:  jobFeat.Value,
			Size:   jobFeat.Size,
			Offset: jobFeat.Offset,
		})
	}

	// Convert info into a string for easier comparison
	stringyJson, err := jobEvent.info.MarshalJSON()
	if err != nil {
		log.Fatalf("Failed to convert info into json which is required.")
	}
	info = string(stringyJson)

	for _, augStream := range jobEvent.augmentedAndContentStreams {
		var rawAugmentedStreamBytes []byte
		if testJobResultOptions.IncludeRawBytes {
			if augStream.Size > MAX_RAW_STREAM_SIZE {
				log.Fatalf("Can't load file with label/sha256 '%s/%s' because it is of size %d which is greater than the max allowed of %d", augStream.Label, augStream.Sha256, augStream.Size, MAX_RAW_STREAM_SIZE)
			}
			if len(augStream.path) > 0 {
				rawFile, err := os.Open(augStream.path)
				if err != nil {
					log.Fatalf("Failed to open augmented stream (label/sha256) %s/%s", augStream.Label, augStream.Sha256)
				}
				defer rawFile.Close()
				rawAugmentedStreamBytes, err = io.ReadAll(rawFile)
				if err != nil {
					log.Fatalf("Failed to read augmented stream (label/sha256) %s/%s", augStream.Label, augStream.Sha256)
				}
			} else {
				rawAugmentedStreamBytes = augStream.RawBytes
			}
		}

		augmentedStreams = append(augmentedStreams, ResultStream{
			RawBytes:     rawAugmentedStreamBytes,
			Relationship: augStream.Relationship,
			Label:        augStream.Label,
			Sha256:       augStream.Sha256,
			Size:         augStream.Size,
		})

	}
	slices.SortFunc(augmentedStreams, compareAugmentedStreamsSort)

	return &TestJobEvent{
		ParentSha256:     parentSha256,
		ChildrenSha256:   childrenSha256,
		Features:         features,
		Info:             info,
		AugmentedStreams: augmentedStreams,
	}
}

type TestJobResult struct {
	Status  string         `json:"status"`
	Message string         `json:"message"`
	Events  []TestJobEvent `json:"events,omitempty,omitzero"`
}

type TestJobResultOptions struct {
	// Include the raw bytes of all the streams.
	IncludeRawBytes bool
}

func NewTestJobResult(job *Job, pluginError *PluginError, testJobResultOptions *TestJobResultOptions) *TestJobResult {
	var events []TestJobEvent
	// Non error case will have events that need to be converted to test equivalents.
	if pluginError == nil {
		allJobEvents := job.getAllJobEventsAsList()
		for _, jobEvent := range allJobEvents {
			// Don't add empty events
			if len(jobEvent.features) == 0 && len(jobEvent.children) == 0 && jobEvent.info == nil && len(jobEvent.augmentedAndContentStreams) == 0 {
				continue
			}
			events = append(events, *NewTestJobEvent(jobEvent, testJobResultOptions))
		}
	}

	return &TestJobResult{
		Status:  job.getEventResultStatus(pluginError),
		Message: job.getEventResultMessage(pluginError),
		Events:  events,
	}
}

func (jr *TestJobResult) GenerateActualEventCode(t *testing.T) {
	litterSettings := litter.Options{
		HidePrivateFields: true,
		HideZeroValues:    true,
		FieldFilter: func(structField reflect.StructField, val reflect.Value) bool {
			switch val.Kind() {
			case reflect.Map:
				fallthrough
			case reflect.Array:
				fallthrough
			case reflect.Slice:
				fallthrough
			case reflect.String:
				if val.Len() == 0 {
					return false
				}
			}
			return true
		},
		// Handle raw bytes specially to prevent them spanning lots of lines.
		DumpFunc: func(val reflect.Value, w io.Writer) bool {
			switch val.Type() {
			case reflect.TypeOf([]byte{}):
				_, err := fmt.Fprintf(w, "(%q)", val.Bytes())
				require.Nil(t, err)
				return true
			}
			return false
		},
	}
	t.Logf(`The below printout can form the base of your test, 
but you must double check the output is as expected.

%s
`, litterSettings.Sdump(jr))
}

func compareTestFeaturesSort(a TestBinaryEntityFeature, b TestBinaryEntityFeature) int {
	valueCompare := cmp.Compare(a.Value, b.Value)
	if valueCompare != 0 {
		return valueCompare
	}
	labelCompare := cmp.Compare(a.Label, b.Label)
	if labelCompare != 0 {
		return labelCompare
	}
	offsetCompare := cmp.Compare(a.Offset, b.Offset)
	if offsetCompare != 0 {
		return offsetCompare
	}
	return cmp.Compare(a.Size, b.Size)
}

func (jr *TestJobResult) AssertJobResultEqual(t *testing.T, expected *TestJobResult) {
	// Marshal and unmarshal json to drop all the omitted empty arrays.
	jsonActual, err := json.Marshal(jr)
	if err != nil {
		t.Fatalf("Couldn't marshal actual result with error %v", err)
	}
	jsonExpected, err := json.Marshal(expected)
	if err != nil {
		t.Fatalf("Couldn't marshal expected result with error %v", err)
	}
	var actualMarshal TestJobResult
	var expectedMarshal TestJobResult
	err = json.Unmarshal(jsonActual, &actualMarshal)
	if err != nil {
		t.Fatalf("Couldn't marshal expected result with error %v", err)
	}
	err = json.Unmarshal(jsonExpected, &expectedMarshal)
	if err != nil {
		t.Fatalf("Couldn't marshal expected result with error %v", err)
	}
	// Additional last minute sorting here (allows for changes in runner to not break tests)
	for _, evt := range expectedMarshal.Events {
		for _, valBatch := range evt.Features {
			slices.SortFunc(valBatch, compareTestFeaturesSort)
		}
	}
	for _, evt := range actualMarshal.Events {
		for _, valBatch := range evt.Features {
			slices.SortFunc(valBatch, compareTestFeaturesSort)
		}
	}
	// Perform assertion here.
	isEqual := assert.Equal(t, expectedMarshal, actualMarshal)
	if !isEqual {
		jr.GenerateActualEventCode(t)
	}
}

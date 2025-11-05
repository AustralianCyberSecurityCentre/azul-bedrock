package plugin

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strconv"
	"time"

	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/events"
)

type ResultStream struct {
	path         string
	RawBytes     []byte                 `json:"rawBytes,omitempty,omitzero"`
	Relationship map[string]string      `json:"relationship,omitempty,omitzero"`
	Label        events.DatastreamLabel `json:"label,omitempty"`
	Sha256       string                 `json:"sha256,omitempty"`
	Size         uint64                 `json:"size,omitempty"`
}

type JobEvent struct {
	parent   *JobEvent
	children []*JobEvent
	features []events.BinaryEntityFeature
	// Contains all the augmented and content streams.
	augmentedAndContentStreams map[string]*ResultStream
	info                       json.RawMessage
	temporaryDirPath           string
}

func NewJobEvent(temporaryDirPath string, parent *JobEvent) *JobEvent {
	return &JobEvent{
		parent:                     nil,
		children:                   []*JobEvent{},
		features:                   []events.BinaryEntityFeature{},
		augmentedAndContentStreams: map[string]*ResultStream{},
		info:                       json.RawMessage{},
		temporaryDirPath:           temporaryDirPath,
	}
}

/*Calcualte the sha256 of a file and return it's file size.*/
func calculateSha256OfFile(filePath string) (string, uint64, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", 0, fmt.Errorf("error calculating sha256 of a file while opening file with error %v", err)
	}
	sha256Sum := sha256.New()
	totalBytes, err := f.WriteTo(sha256Sum)
	if err != nil {
		return "", 0, fmt.Errorf("error calculating sha256 while calculating hash with error %v", err)
	}
	return fmt.Sprintf("%x", sha256Sum.Sum(nil)), uint64(totalBytes), nil
}

/*Calculate the sha256 of a bytes array and return the the sha256.*/
func calculateSha256OfBytes(data []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

// Add extra options to the feature such as label offset or size.
type AddFeatureOptions struct {
	Label  string
	Offset uint64
	Size   uint64
}

// Add a feature to the job event, the feature can be of type string, uint, int, float, bool or time.time.
func (je *JobEvent) AddFeature(name string, value interface{}) *PluginError {
	return je.AddFeatureWithExtra(name, value, &AddFeatureOptions{})
}

// Add a feature to the job event, and a label, offset and/or size as well.
func (je *JobEvent) AddFeatureWithExtra(name string, value interface{}, options *AddFeatureOptions) *PluginError {
	stringValue := ""
	valInput := reflect.ValueOf(value)
	switch valInput.Kind() {
	case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
		stringValue = strconv.FormatInt(valInput.Int(), 10)
	case reflect.Uint, reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8:
		stringValue = strconv.FormatUint(valInput.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		stringValue = strconv.FormatFloat(valInput.Float(), 'f', -1, 64)
	case reflect.Bool:
		stringValue = strconv.FormatBool(valInput.Bool())
	case reflect.String:
		stringValue = valInput.String()
	default:
		switch valInput.Type() {
		case reflect.TypeOf([]byte{}):
			stringValue = string(valInput.Bytes())
		case reflect.TypeOf(time.Time{}):
			stringValue = value.(time.Time).Format(time.RFC3339)
		default:
			return NewPluginError(
				ErrorRunner,
				"Invalid Feature Type",
				fmt.Sprintf("Provided feature value was an invalid type '%s'", valInput.Type()),
			)
		}
	}
	feat := events.BinaryEntityFeature{
		Name:  name,
		Value: stringValue,
	}
	var zeroString string
	var zeroUint64 uint64
	if options.Label != zeroString {
		feat.Label = options.Label
	}
	if options.Offset != zeroUint64 {
		feat.Offset = options.Offset
	}
	if options.Size != zeroUint64 {
		feat.Size = options.Size
	}

	je.features = append(je.features, feat)
	return nil
}

/*Add info.*/
func (je *JobEvent) AddInfo(info json.RawMessage) {
	je.info = info
}

/*Add child file as part of results with a relationship to the current binary.*/
func (je *JobEvent) AddChildBytes(data []byte, relationship map[string]string) *JobEvent {
	sha256 := calculateSha256OfBytes(data)
	childJobEvent := NewJobEvent(je.temporaryDirPath, je)
	childJobEvent.augmentedAndContentStreams[sha256] = &ResultStream{
		RawBytes:     data,
		Label:        events.DataLabelContent,
		Relationship: relationship,
		Sha256:       sha256,
		Size:         uint64(len(data)),
	}
	je.children = append(je.children, childJobEvent)
	return childJobEvent
}

/*Add augmented file as part of results with a relationship to the current binary.*/
func (je *JobEvent) AddAugmentedBytes(data []byte, label events.DatastreamLabel) error {
	if label == events.DataLabelContent {
		return fmt.Errorf("can't add a file with a content label as an augmented stream")
	}
	sha256 := calculateSha256OfBytes(data)
	je.augmentedAndContentStreams[sha256] = &ResultStream{
		RawBytes: data,
		Label:    label,
		Sha256:   sha256,
		Size:     uint64(len(data)),
	}
	return nil
}

/*Add child file as part of results with a relationship to the current binary (Plugin is expected to delete original file).*/
func (je *JobEvent) AddChild(dataPath string, relationship map[string]string) (*JobEvent, error) {
	childStream, err := je.addChildOrAugmentedAsPath(dataPath, relationship, events.DataLabelContent)
	if err != nil {
		return &JobEvent{}, err
	}
	childJobEvent := NewJobEvent(je.temporaryDirPath, je)
	childJobEvent.augmentedAndContentStreams[childStream.Sha256] = childStream
	je.children = append(je.children, childJobEvent)
	return childJobEvent, nil
}

/*Add augmented file as part of results with a relationship to the current binary (Plugin is expected to delete original file).*/
func (je *JobEvent) AddAugmented(dataPath string, label events.DatastreamLabel) error {
	if label == events.DataLabelContent {
		return fmt.Errorf("can't add a file with a content label as an augmented stream")
	}
	augmentedStream, err := je.addChildOrAugmentedAsPath(dataPath, map[string]string{}, label)
	if err != nil {
		return err
	}
	je.augmentedAndContentStreams[augmentedStream.Sha256] = augmentedStream
	return nil
}

/*Add child file from a file.*/
func (je *JobEvent) addChildOrAugmentedAsPath(
	dataPath string, relationship map[string]string, label events.DatastreamLabel,
) (*ResultStream, error) {
	newPath, err := je.copyFileToPath(dataPath)
	if err != nil {
		return &ResultStream{}, err
	}
	sha256, size, err := calculateSha256OfFile(newPath)
	if err != nil {
		os.Remove(newPath)
		return &ResultStream{}, err
	}
	stream := &ResultStream{
		path:   newPath,
		Label:  label,
		Sha256: sha256,
		Size:   size,
	}
	if len(relationship) > 0 {
		stream.Relationship = relationship
	}
	return stream, nil
}

/*Copy the provided source file to the destination file location.*/
func (je *JobEvent) copyFileToPath(path string) (string, error) {
	// Create location to save file to.
	tempFile, err := os.CreateTemp(je.temporaryDirPath, "runner-child-")
	if err != nil {
		return "", fmt.Errorf("failed to open provided file when with error: %v", err.Error())
	}
	defer tempFile.Close()

	// Save file to the specified path.
	f, err := os.Open(path)
	if err != nil {
		// Ensure tempfile is deleted if there is an error
		defer os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to open file to copy contents into in gorunner: %v", err.Error())
	}
	defer f.Close()
	_, err = f.WriteTo(tempFile)
	if err != nil {
		// Ensure tempfile is deleted if there is an error
		defer os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to copy contents of file from one to the other in gorunner: %v", err.Error())
	}

	return tempFile.Name(), nil
}

/*Shorthand way of getting content stream if it's present or an error if it isn't.*/
func (je *JobEvent) getContentStream() (*ResultStream, error) {
	for _, stream := range je.augmentedAndContentStreams {
		if stream.Label == events.DataLabelContent {
			return stream, nil
		}
	}
	return &ResultStream{}, fmt.Errorf("content stream not found")
}

/*Get the file name for the job event if it has one, used when generating child events*/
func (je *JobEvent) getFileName() string {
	fileNames := []string{}
	for _, feat := range je.features {
		if feat.Name == "filename" {
			fileNames = append(fileNames, feat.Value)
		}
	}
	slices.Sort(fileNames)
	if len(fileNames) > 0 {
		return fileNames[0]
	}
	return ""
}

/*Used to get the path to a child binary.*/
func (je *JobEvent) recursivelyGetExtractedPathNodes(
	authorSummary events.EventAuthor, now time.Time, uploadData map[string]*events.BinaryEntityDatastream,
) ([]*events.EventSourcePathNode, error) {
	if je.parent == nil {
		// Root event has no parent and isn't included in the path so simply return the list used for the return.
		return []*events.EventSourcePathNode{}, nil
	}
	// Recursively call parent.
	pathNodes, err := je.parent.recursivelyGetExtractedPathNodes(authorSummary, now, uploadData)
	if err != nil {
		return []*events.EventSourcePathNode{}, err
	}

	// Generate a path node and append it as the recursive call unwinds.
	contentStream, err := je.getContentStream()
	if err != nil {
		return []*events.EventSourcePathNode{}, err
	}
	binaryEntityDatastream, ok := uploadData[getStreamKey(contentStream)]
	if !ok {
		return []*events.EventSourcePathNode{},
			fmt.Errorf(
				"unexpectedly missing stream with label, sha256 (%s, %s) during extracted path node extraction",
				contentStream.Label,
				contentStream.Sha256,
			)
	}
	newPathNode := events.EventSourcePathNode{
		Author:           authorSummary,
		Action:           events.ActionExtracted,
		Sha256:           binaryEntityDatastream.Sha256,
		Relationship:     contentStream.Relationship,
		Timestamp:        now,
		FileFormatLegacy: binaryEntityDatastream.FileFormatLegacy,
		FileFormat:       binaryEntityDatastream.FileFormat,
		Size:             binaryEntityDatastream.Size,
		Filename:         je.getFileName(),
		Language:         binaryEntityDatastream.Language,
	}

	return append(pathNodes, &newPathNode), nil
}

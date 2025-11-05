package events

import (
	"cmp"
	"crypto/md5"
	"errors"
	"fmt"
	"log"
	"maps"
	"slices"
	"strconv"
	"time"

	"github.com/goccy/go-json"
)

// A single feature value pair, with additional context
type BinaryEntityFeature struct {
	Name   string      `json:"name" avro:"name"`
	Value  string      `json:"value" avro:"value"`
	Type   FeatureType `json:"type" avro:"type"`
	Label  string      `json:"label,omitempty" avro:"label"`
	Size   uint64      `json:"size,omitempty" avro:"size"`
	Offset uint64      `json:"offset,omitempty" avro:"offset"`
}

// Dispatcher calcualted information about binary blob
type BinaryEntityDatastream struct {
	IdentifyVersion  uint32          `json:"identify_version,omitempty" avro:"identify_version"`
	Label            DatastreamLabel `json:"label" avro:"label"`
	Size             uint64          `json:"size" avro:"size"`
	Sha512           string          `json:"sha512" avro:"sha512"`
	Sha256           string          `json:"sha256" avro:"sha256"`
	Sha1             string          `json:"sha1" avro:"sha1"`
	Md5              string          `json:"md5" avro:"md5"`
	Ssdeep           string          `json:"ssdeep,omitempty" avro:"ssdeep"`
	Tlsh             string          `json:"tlsh,omitempty" avro:"tlsh"`
	Mime             string          `json:"mime" avro:"mime"`
	Magic            string          `json:"magic" avro:"magic"`
	FileFormatLegacy string          `json:"file_format_legacy,omitempty" avro:"file_format_legacy"`
	FileFormat       string          `json:"file_format,omitempty" avro:"file_format"`
	FileExtension    string          `json:"file_extension,omitempty" avro:"file_extension"`
	Language         string          `json:"language,omitempty" avro:"language"`
}

// Binary Entity type
type BinaryEntity struct {
	Sha256           string                   `json:"sha256" avro:"sha256"`
	Sha512           string                   `json:"sha512,omitempty" avro:"sha512"`
	Sha1             string                   `json:"sha1,omitempty" avro:"sha1"`
	Md5              string                   `json:"md5,omitempty" avro:"md5"`
	Ssdeep           string                   `json:"ssdeep,omitempty" avro:"ssdeep"`
	Tlsh             string                   `json:"tlsh,omitempty" avro:"tlsh"`
	Size             uint64                   `json:"size,omitempty" avro:"size"`
	Mime             string                   `json:"mime,omitempty" avro:"mime"`
	Magic            string                   `json:"magic,omitempty" avro:"magic"`
	FileFormatLegacy string                   `json:"file_format_legacy,omitempty" avro:"file_format_legacy"`
	FileFormat       string                   `json:"file_format,omitempty" avro:"file_format"`
	FileExtension    string                   `json:"file_extension,omitempty" avro:"file_extension"`
	Features         []BinaryEntityFeature    `json:"features,omitempty,omitzero" avro:"features"`
	Datastreams      []BinaryEntityDatastream `json:"datastreams,omitempty,omitzero" avro:"datastreams"`
	Info             json.RawMessage          `json:"info,omitempty" avro:"info"` // info block
}

/*Return a copy of the binary entity without features or info but keeping datastreams.*/
func (be *BinaryEntity) CopyWithDataStreams() *BinaryEntity {
	returnVal := *be
	returnVal.Features = []BinaryEntityFeature{}
	returnVal.Info = json.RawMessage{}
	returnVal.Datastreams = be.Datastreams
	return &returnVal
}

type BinaryFlags struct {
	BypassCache bool `json:"bypass_cache,omitempty" avro:"bypass_cache"` // always process, do not look in cache for results
	Expedite    bool `json:"expedite,omitempty" avro:"expedite"`         // put event in expedite topic for faster analysis
	Retry       bool `json:"retry,omitempty" avro:"retry"`               // put event in retry topic for another attempt
}

type BinaryEvent struct {
	ModelVersion          uint32       `json:"model_version,omitempty" avro:"model_version"`
	KafkaKey              string       `json:"kafka_key,omitempty" avro:"kafka_key"`
	Timestamp             time.Time    `json:"timestamp" avro:"timestamp"`
	Author                EventAuthor  `json:"author" avro:"author"`
	Source                EventSource  `json:"source" avro:"source"`
	Entity                BinaryEntity `json:"entity" avro:"entity"`
	Action                BinaryAction `json:"action" avro:"action"`
	Dequeued              string       `json:"dequeued,omitempty" avro:"dequeued"`
	Retries               uint32       `json:"retries,omitempty" avro:"retries"`
	Flags                 BinaryFlags  `json:"flags,omitempty" avro:"flags"`
	TrackSourceReferences string       `json:"track_source_references,omitempty" avro:"track_source_references"`
	TrackLinks            []string     `json:"track_links,omitempty,omitzero" avro:"track_links"`
	TrackAuthors          []string     `json:"track_authors,omitempty,omitzero" avro:"track_authors"`
}

type BulkBinaryEvent struct {
	ModelVersion uint32         `json:"model_version,omitempty" avro:"model_version"`
	Events       []*BinaryEvent `json:"events" avro:"events"`
}

func (evs *BulkBinaryEvent) GetModel() Model {
	return ModelBinary
}

func (evs *BulkBinaryEvent) IsBulk() bool {
	return true
}
func (evs *BulkBinaryEvent) GetModelVersion() uint32 {
	return evs.ModelVersion
}

func (evs *BulkBinaryEvent) SetModelVersion(newVersion uint32) {
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

func (evs *BulkBinaryEvent) ToAvro() ([]byte, error) {
	return GenericToAvro(evs, SchemaBulkBinary)
}

func (evs *BulkBinaryEvent) FromAvro(data []byte) error {
	err := GenericFromAvro(evs, data, SchemaBulkBinary)
	// Avro sometimes drops lots of data but doesn't error when un-marshalling bulk schemas.
	if len(evs.Events) == 0 && len(data) > LENGTH_OF_BULK_HEADER_INFO {
		return fmt.Errorf("bulk event was not properly un-marshalled by avro")
	}
	return err
}

func (ev *BinaryEvent) GetModelVersion() uint32 {
	return ev.ModelVersion
}

func (ev *BinaryEvent) SetModelVersion(newVersion uint32) {
	ev.ModelVersion = newVersion
}

func (ev *BinaryEvent) ToAvro() ([]byte, error) {
	return GenericToAvro(ev, SchemaBinary)
}

func (ev *BinaryEvent) FromAvro(data []byte) error {
	return GenericFromAvro(ev, data, SchemaBinary)
}

// UpdateTrackingFields calculates important ids for tracking the event
func (es *BinaryEvent) UpdateTrackingFields() error {
	// calculate submission
	// source.<md5 of 'k1.v1.k2.v2.'>
	es.TrackSourceReferences = es.Source.Name + "."
	vals := ""
	for _, key := range sortedKeys(es.Source.References) {
		vals += key + "." + es.Source.References[key] + "."
	}
	es.TrackSourceReferences += fmt.Sprintf("%x", md5.Sum([]byte(vals)))

	// calculate links & authors
	es.TrackAuthors = []string{}
	es.TrackLinks = []string{}
	parent := -1
	for i, node := range es.Source.Path {
		es.TrackAuthors = append(es.TrackAuthors, genTrackAuthor(&node))
		if parent >= 0 {
			parentId := es.Source.Path[parent].Sha256
			es.TrackLinks = append(es.TrackLinks, genTrackLink(&node, parentId))
		}
		parent = i
	}
	return nil
}

func (b *BinaryEvent) GetBase() *BaseEvent {
	return &BaseEvent{
		Model:        ModelBinary,
		ModelVersion: &b.ModelVersion,
		KafkaKey:     &b.KafkaKey,
		Timestamp:    &b.Timestamp,
		Author:       &b.Author,
	}
}

/*Standard validation plus some additional binary specific validation.*/
func (b *BinaryEvent) CheckValid() error {
	if len(b.Author.Name) == 0 {
		return errors.New("event is missing 'author' field")
	}
	if len(b.Action) == 0 {
		return errors.New("event is missing 'action' field")
	}
	if len(b.Source.Path) == 0 {
		return errors.New("event is missing 'source.path' entries")
	}

	labels := map[DatastreamLabel]bool{}
	for _, curData := range b.Entity.Datastreams {
		if !IsDataLabelValid(curData.Label) {
			return fmt.Errorf("the provided label '%s' is invalid, please use a valid label", curData.Label)
		}
		labels[curData.Label] = true
	}

	// enriched, mapped means no entity.datastreams
	switch b.Action {
	case ActionEnriched:
		if len(b.Entity.Datastreams) > 0 {
			return fmt.Errorf("%s cannot have entity.datastreams entries", b.Action)
		}
	case ActionMapped:
		if labels["content"] {
			return fmt.Errorf("%s can only have labels that aren't content", b.Action)
		}
	default:
		// binary_* others means entity.datastreams with label=content
		if !labels["content"] {
			return fmt.Errorf("%s must have entity.datastreams with label=content", b.Action)
		}
		if b.Action == ActionAugmented && len(labels) < 2 {
			// augmented means entity.datastreams with label!=content as well
			return fmt.Errorf("%s must have entity.datastreams with another label!=content", b.Action)
		}
	}

	// check valid path
	totalNodes := len(b.Source.Path)
	for i, node := range b.Source.Path {
		// enriched and mapped can't produce other events
		if totalNodes-1 != i {
			// intermediate node
			if node.Action == ActionEnriched {
				return fmt.Errorf("found %s on non leaf node", node.Action)
			}
		}
	}

	return nil
}

func (data *BinaryEntityDatastream) ToInputEntity() *BinaryEntity {
	// mirrors logic in models_network.py FileInfo.to_input_entity()
	ret := &BinaryEntity{
		Sha256:           data.Sha256,
		Sha512:           data.Sha512,
		Sha1:             data.Sha1,
		Md5:              data.Md5,
		Ssdeep:           data.Ssdeep,
		Tlsh:             data.Tlsh,
		Size:             data.Size,
		Mime:             data.Mime,
		Magic:            data.Magic,
		FileFormatLegacy: data.FileFormatLegacy,
		FileFormat:       data.FileFormat,
		FileExtension:    data.FileExtension,
		Features: []BinaryEntityFeature{
			{Name: "file_format", Type: FeatureString, Value: data.FileFormat},
			{Name: "file_format_legacy", Type: FeatureString, Value: data.FileFormatLegacy},
			{Name: "file_extension", Type: FeatureString, Value: data.FileExtension},
			{Name: "magic", Type: FeatureString, Value: data.Magic},
			{Name: "mime", Type: FeatureString, Value: data.Mime},
		},
		Datastreams: []BinaryEntityDatastream{*data},
	}
	// remove features with no value
	presentFeatures := []BinaryEntityFeature{}
	for _, f := range ret.Features {
		if f.Value != "" {
			presentFeatures = append(presentFeatures, f)
		}
	}
	ret.Features = presentFeatures
	return ret
}

// sortedKeys returns keys in a map, but sorted
func sortedKeys[U cmp.Ordered, V any](m map[U]V) []U {
	keys := slices.Collect(maps.Keys(m))
	slices.Sort(keys)
	return keys
}

func genTrackAuthor(node *EventSourcePathNode) string {
	// plugin_category.plugin_name.plugin_version
	return node.Author.Category + "." + node.Author.Name + "." + node.Author.Version
}

func genTrackLink(node *EventSourcePathNode, parentId string) string {
	// parent.child.plugin_category.plugin_name.plugin_version
	return parentId + "." + node.Sha256 + "." + genTrackAuthor(node)
}

type ValidationOptions struct {
	/* Arguments that can be parsed into BinaryFeature validation, recommend using the constructor which sets defaults.*/
	MaxFeatureCount            int // Max number of features a plugin result can have default is 10000
	MaxValuesInOneFeatureCount int // Max values that one feature can have default is 1000
	MaxFVLength                int // Max feature value length default is 4000
}

// Set the option maximum number of features that can be in a plugins response.
func (vo *ValidationOptions) WithMaxFeatureCount(count int) *ValidationOptions {

	vo.MaxFeatureCount = count
	return vo
}

// Set the option maximum number of values in one feature (e.g if it has a list of values).
func (vo *ValidationOptions) WithMaxValuesInOneFeatureCount(count int) *ValidationOptions {

	vo.MaxValuesInOneFeatureCount = count
	return vo
}

// Set a validation option max feature value length (max length of a features string value)
func (vo *ValidationOptions) WithMaxFVLength(length int) *ValidationOptions {
	vo.MaxFVLength = length
	return vo
}

// Create a new validation options object and allows you to modify settings with fluent APIs.
func NewValidationOptions() *ValidationOptions {
	vo := &ValidationOptions{
		MaxFeatureCount:            10000,
		MaxValuesInOneFeatureCount: 10000,
		MaxFVLength:                4000,
	}
	return vo
}

func (e *BinaryEntity) ProcessAndValidateBinaryFeatures(options *ValidationOptions) ([]string, error) {
	feats, warnings, err := ProcessAndValidateBinaryFeatures(e.Features, options)
	e.Features = feats
	return warnings, err
}

// Sort function for features to ensure they are consistently sorted.
func CompareFeaturesSort(a BinaryEntityFeature, b BinaryEntityFeature) int {
	nameCompare := cmp.Compare(a.Name, b.Name)
	if nameCompare != 0 {
		return nameCompare
	}
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

func ProcessAndValidateBinaryFeatures(featsIn []BinaryEntityFeature, options *ValidationOptions) ([]BinaryEntityFeature, []string, error) {
	// Sort features for consistency.
	slices.SortFunc(featsIn, CompareFeaturesSort)

	warnings := []string{}
	// Drop excess features if there are any (pointless to do if the MaxFeatureCount is the MaxValuesInOneFeatureCount)
	if options.MaxFeatureCount != options.MaxValuesInOneFeatureCount {
		var indexesToRemove []int
		featureNameSet := make(map[string]int)
		// Go through all features and if the count exceeds the allowable number start dropping all subsequent features with the same name.
		for idx, feat := range featsIn {
			_, exists := featureNameSet[feat.Name]
			if exists {
				if featureNameSet[feat.Name] >= options.MaxValuesInOneFeatureCount {
					indexesToRemove = append(indexesToRemove, idx)
				}
				featureNameSet[feat.Name] += 1
			} else {
				featureNameSet[feat.Name] = 1
			}
		}
		// Create a warning summarising all the dropped features.
		for featName, count := range featureNameSet {
			if count >= options.MaxValuesInOneFeatureCount {
				featureWarning := fmt.Sprintf("too many values for feature %s (%d) max is %d, dropping feature", featName, count, options.MaxValuesInOneFeatureCount)
				warnings = append(warnings, featureWarning)
			}
		}

		// Drop excess values for the given features (in reverse to prevent potential index issues).
		slices.Reverse(indexesToRemove)
		for _, index := range indexesToRemove {
			featsIn = slices.Delete(featsIn, index, index+1)
		}
	}

	// Drop excess features if there are more than the max allowed.
	if len(featsIn) >= options.MaxFeatureCount {
		maxFeatCountWarning := fmt.Sprintf("There are %d features where a max of %d are allowed dropping excess features.\n", len(featsIn), options.MaxFeatureCount)
		log.Printf("Warning: %s", maxFeatCountWarning)
		warnings = append(warnings, maxFeatCountWarning)

		featsIn = featsIn[:options.MaxFeatureCount]
	}

	// Validate an modify the Feature values as necessary.
	indexesToRemove := []int{}
	for idx := range featsIn {
		keep, warning, err := featsIn[idx].ProcessAndValidateFeature(options)
		if err != nil {
			return featsIn, warnings, err
		}
		if warning != "" {
			warnings = append(warnings, warning)
		}
		if !keep {
			indexesToRemove = append(indexesToRemove, idx)
		}
	}

	// Drop excess values for the given features (in reverse to prevent potential index issues).
	slices.Reverse(indexesToRemove)
	for _, index := range indexesToRemove {
		featsIn = slices.Delete(featsIn, index, index+1)
	}
	return featsIn, warnings, nil
}

func (e *BinaryEntityFeature) ProcessAndValidateFeature(options *ValidationOptions) (bool, string, error) {
	keep := true
	clipped := false
	switch e.Type {
	case FeatureUri:
		fallthrough
	case FeatureFilepath:
		fallthrough
	case FeatureString:
		// clip Feature values that are too long
		if len(e.Value) > options.MaxFVLength {
			clipped = true
			e.Value = e.Value[:options.MaxFVLength]
		}
	case FeatureInteger:
		_, err := strconv.ParseInt(e.Value, 10, 64)
		if err != nil {
			return false, "", fmt.Errorf("integer %s must be a valid int value was %s and couldn't be parsed with error %v", e.Name, e.Value, err)
		}
	case FeatureFloat:
		_, err := strconv.ParseFloat(e.Value, 64)
		if err != nil {
			return false, "", fmt.Errorf("float %s must be a valid float value was %s and couldn't be parsed with error %v", e.Name, e.Value, err)
		}
	case FeatureDatetime:
		_, err := time.Parse(time.RFC3339, e.Value)
		if err != nil {
			return false, "", fmt.Errorf("dateTime %s must be a valid RFC3339 date value was %s and couldn't be parsed with error %v", e.Name, e.Value, err)
		}
	default:
		// no reliable way to clip without corrupting, so drop the feature value
		if len(e.Value) > options.MaxFVLength {
			clipped = true
			keep = false
		}
	}

	warning := ""
	if clipped {
		errMsgSample := ""
		if len(e.Value) < 200 {
			errMsgSample = e.Value
		} else {
			errMsgSample = e.Value[:200]
		}
		warning = fmt.Sprintf("Warning: The feature value %v had a value that was %d chars long and can only be %d chars long so it was clipped, value started with: '%v'.", e.Name, len(e.Value), options.MaxFVLength, errMsgSample)
	}
	return keep, warning, nil
}

package events

import (
	"log"
	"testing"

	testdata "github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/testdata"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventBinary1(t *testing.T) {
	// This is quite an old event I had to edit a bit
	data := testdata.GetBytes("events/binary/extracted.json")
	var ev BinaryEvent
	err := json.Unmarshal(data, &ev)
	require.Nil(t, err)
	require.Equal(t, ev.Entity.Sha256, "ee303d3c6d7cfa24d42e6348bdd1103a26de77a887e9dbee3dd1fe6304414f69")
	require.Equal(t, ev.Entity.Mime, "")
	require.Equal(t, ev.Entity.Features[0], BinaryEntityFeature{Name: "filename", Value: "ukr_kh1.gif", Type: FeatureFilepath, Label: "", Size: 0x0, Offset: 0x0})
	require.Equal(t, ev.TrackSourceReferences, "")
	require.Equal(t, ev.TrackLinks, []string(nil))
	require.Equal(t, ev.TrackAuthors, []string(nil))
	ev.UpdateTrackingFields()
	require.Equal(t, ev.TrackSourceReferences, "virustotal.5a0cc3b7a38feb282d8ae928eca68f5d")
	require.Equal(t, ev.TrackLinks, []string{
		"b9debe8afbdc6d0f552ca4d41fc8a5760778f2724d1560f6b4b0fc28bd837a82.ee303d3c6d7cfa24d42e6348bdd1103a26de77a887e9dbee3dd1fe6304414f69.plugin.MimeDecoder.2021.02.14",
	})
	require.Equal(t, ev.TrackAuthors, []string{
		"plugin.vtdownload.1.1",
		"plugin.MimeDecoder.2021.02.14",
	})
}

func createBinaryEvent() *BinaryEvent {
	data := testdata.GetBytes("events/binary/extracted.json")
	var ev BinaryEvent
	err := json.Unmarshal(data, &ev)
	if err != nil {
		log.Fatalf("Test setup for generic binary event failed when EventFromBytes with error %v", err)
	}
	ev.Entity.Features = []BinaryEntityFeature{}
	return &ev
}

func loadBinaryEvent(filename string) *BinaryEvent {
	data := testdata.GetBytes("events/binary/" + filename)
	var ev BinaryEvent
	err := json.Unmarshal(data, &ev)
	if err != nil {
		log.Fatalf("Test setup for generic binary event failed when EventFromBytes with error %v", err)
	}
	return &ev
}

func createGenericBinaryEntity() *BinaryEntity {
	ev := createBinaryEvent()
	return &ev.Entity
}

func TestDeepCopyOfBinaryEvent(t *testing.T) {
	binaryEvent := createBinaryEvent()
	entity := binaryEvent.Entity
	entity.Features = []BinaryEntityFeature{{Name: "abc", Value: "def", Type: FeatureString}}
	require.Equal(t, 1, len(entity.Datastreams))
	be := entity.CopyWithDataStreams()
	require.False(t, be == &entity, "Memory addresses are equal and shouldn't be after copy.")
	require.Equal(t, be.Datastreams, entity.Datastreams)
	require.Equal(t, be.Sha256, entity.Sha256)
	require.NotEqual(t, be.Info, entity.Info)
	require.NotEqual(t, be.Features, entity.Features)
}

type variousValueValidationTests struct {
	inputValue      []string
	expectedResults map[FeatureType][]string
}

// Validation tests.
func TestValid(t *testing.T) {
	var tmp *BinaryEvent

	// check ok extracted
	tmp = loadBinaryEvent("extracted.json")
	require.Nil(t, tmp.CheckValid())

	// check ok enriched
	tmp = loadBinaryEvent("enriched.json")
	require.Nil(t, tmp.CheckValid())

	// check ok enriched with info
	tmp = loadBinaryEvent("enriched_w_info_dict.json")
	require.Nil(t, tmp.CheckValid())
	require.JSONEq(t, string(tmp.Entity.Info), `{"this": "is some testing data to see whats going on"}`)

	// check ok enriched with info
	tmp = loadBinaryEvent("enriched_w_info_list.json")
	require.Nil(t, tmp.CheckValid())
	require.JSONEq(t, string(tmp.Entity.Info), `[111.0, 222.0, 333.0, "hello"]`)

	// check ok augmented
	tmp = loadBinaryEvent("augmented.json")
	require.Nil(t, tmp.CheckValid())

	// check ok mapped
	tmp = loadBinaryEvent("mapped_w_non_content_data.json")
	require.Nil(t, tmp.CheckValid())

	// check fail mapped with content
	tmp = loadBinaryEvent("mapped_w_non_content_data.json")
	tmp.Entity.Datastreams[0].Label = DataLabelContent
	require.ErrorContains(t, tmp.CheckValid(), "mapped can only have labels that aren't content")

	// check fail enriched with data
	tmp = loadBinaryEvent("enriched_w_data.json")
	require.ErrorContains(t, tmp.CheckValid(), "enriched cannot have entity.datastreams entries")

	// check fail non binary on path
	tmp = loadBinaryEvent("invalid_path.json")
	require.ErrorContains(t, tmp.CheckValid(), "found enriched on non leaf node")

	// check fail augmented with no extra stream
	tmp = loadBinaryEvent("augmented_no_extra.json")
	require.ErrorContains(t, tmp.CheckValid(), "augmented must have entity.datastreams with another label!=content")

	// Check Valid Label
	genericBinEv := createBinaryEvent()
	genericBinEv.Entity.Datastreams = append(genericBinEv.Entity.Datastreams, BinaryEntityDatastream{IdentifyVersion: 0, Label: DataLabelContent})
	require.Nil(t, genericBinEv.CheckValid())

	// Check Invalid Label
	genericBinEv.Entity.Datastreams = append(genericBinEv.Entity.Datastreams, BinaryEntityDatastream{IdentifyVersion: 0, Label: "bad-label-really-it-should-error"})
	require.ErrorContains(t, genericBinEv.CheckValid(), "the provided label 'bad-label-really-it-should-error' is invalid")
}

// Validation tests.
func TestToInputEntity(t *testing.T) {
	genericBinEv := createBinaryEvent()
	data := genericBinEv.Entity.Datastreams[0]
	input := data.ToInputEntity()
	require.Equal(t, input, &BinaryEntity{
		Sha256:           "ee303d3c6d7cfa24d42e6348bdd1103a26de77a887e9dbee3dd1fe6304414f69",
		Sha512:           "0e61e710aa6737129b6b42b4e5e18c0c66f2febe4277c9e1f85faaed155de966fd7cdbcc8a7b5dff7936acfbeeec9073e594137dcb3f1c30544148e753ef0314",
		Sha1:             "05ea1ec1fd241a09713d43cd70e51f934de94b34",
		Md5:              "9e4638d508c003f83f44f5d07748f33a",
		Ssdeep:           "",
		Tlsh:             "",
		Size:             0x88ee,
		Mime:             "image/gif",
		Magic:            "GIF image data, version 89a, 80 x 35",
		FileFormatLegacy: "GIF",
		FileFormat:       "image/gif",
		FileExtension:    "gif",
		Features: []BinaryEntityFeature{
			{Name: "file_format", Value: "image/gif", Type: FeatureString},
			{Name: "file_format_legacy", Value: "GIF", Type: FeatureString},
			{Name: "file_extension", Value: "gif", Type: FeatureString},
			{Name: "magic", Value: "GIF image data, version 89a, 80 x 35", Type: FeatureString},
			{Name: "mime", Value: "image/gif", Type: FeatureString},
		},
		Datastreams: []BinaryEntityDatastream{{IdentifyVersion: 1,
			Label:            "content",
			Size:             0x88ee,
			Sha512:           "0e61e710aa6737129b6b42b4e5e18c0c66f2febe4277c9e1f85faaed155de966fd7cdbcc8a7b5dff7936acfbeeec9073e594137dcb3f1c30544148e753ef0314",
			Sha256:           "ee303d3c6d7cfa24d42e6348bdd1103a26de77a887e9dbee3dd1fe6304414f69",
			Sha1:             "05ea1ec1fd241a09713d43cd70e51f934de94b34",
			Md5:              "9e4638d508c003f83f44f5d07748f33a",
			Ssdeep:           "",
			Tlsh:             "",
			Mime:             "image/gif",
			Magic:            "GIF image data, version 89a, 80 x 35",
			FileFormatLegacy: "GIF",
			FileFormat:       "image/gif",
			FileExtension:    "gif",
			Language:         ""},
		},
	})

	// check that empty file format is not present in features
	data.FileFormat = ""
	input = data.ToInputEntity()
	require.Equal(t, input.Features, []BinaryEntityFeature{
		{Name: "file_format_legacy", Value: "GIF", Type: FeatureString},
		{Name: "file_extension", Value: "gif", Type: FeatureString},
		{Name: "magic", Value: "GIF image data, version 89a, 80 x 35", Type: FeatureString},
		{Name: "mime", Value: "image/gif", Type: FeatureString},
	})

}

// Feature Validation tests.
func TestTypeValidation(t *testing.T) {
	genericBin := createGenericBinaryEntity()
	options := NewValidationOptions()

	genericBin.Features = append(genericBin.Features, BinaryEntityFeature{
		Name:  "GenericFeature",
		Value: "hello",
		Type:  FeatureInteger,
		Label: "GenericLabel",
	})

	testData := variousValueValidationTests{
		inputValue: []string{
			"hello",                              // string
			"10232.1232",                         // float
			"2021-03-29T14:24:08.54928032+11:00", // Time object
			"100000000",                          // integer
		},
		expectedResults: map[FeatureType][]string{
			FeatureInteger:  {"error", "error", "error", "apass"},
			FeatureFloat:    {"error", "apass", "error", "apass"},
			FeatureString:   {"apass", "apass", "apass", "apass"},
			FeatureUri:      {"apass", "apass", "apass", "apass"},
			FeatureFilepath: {"apass", "apass", "apass", "apass"},
			FeatureDatetime: {"error", "error", "apass", "error"},
		},
	}

	for typeValue, expectedValues := range testData.expectedResults {
		for idx, expectedValue := range expectedValues {
			inputValue := testData.inputValue[idx]

			genericBin.Features[0].Value = inputValue
			genericBin.Features[0].Type = FeatureType(typeValue)
			t.Logf("Starting test case for inputs %v (Testing parser)-%v (panic/no-panic expected) -%v (INPUT)", typeValue, expectedValue, inputValue)
			warnings, err := genericBin.ProcessAndValidateBinaryFeatures(options)
			if expectedValue == "error" {
				assert.NotNil(t, err)

			} else if expectedValue == "apass" {
				assert.Nil(t, err)
				assert.Len(t, warnings, 0)
			} else {
				t.Fatalf("Failed to find expected result '%v' with value for key '%v'", expectedValue, typeValue)
			}
		}
	}
}

func TestStringTooLong(t *testing.T) {
	genericBin := createGenericBinaryEntity()
	options := NewValidationOptions().WithMaxFVLength(20)
	alphabet := "abcdefghijklmnopqrstuvwxyz"

	genericBin.Features = append(genericBin.Features, BinaryEntityFeature{
		Name:  "GenericFeature",
		Value: alphabet,
		Type:  FeatureString,
		Label: "GenericLabel",
	})

	warnings, err := genericBin.ProcessAndValidateBinaryFeatures(options)
	t.Logf("WARNINGS IS: %v", warnings)
	assert.Len(t, warnings, 1)
	assert.Nil(t, err)
	assert.Equal(t, "abcdefghijklmnopqrst", genericBin.Features[0].Value)

	// Same test for URI
	genericBin.Features[0].Value = alphabet
	genericBin.Features[0].Type = FeatureUri
	assert.Equal(t, alphabet, genericBin.Features[0].Value)
	warnings, err = genericBin.ProcessAndValidateBinaryFeatures(options)
	assert.Len(t, warnings, 1)
	assert.Nil(t, err)
	assert.Equal(t, "abcdefghijklmnopqrst", genericBin.Features[0].Value)

	options = NewValidationOptions().WithMaxFVLength(5)
	// Same test for a int that is too long for a uint64
	genericBin.Features[0].Value = "12345678911111111111111111111110"
	genericBin.Features[0].Type = FeatureInteger
	warnings, err = genericBin.ProcessAndValidateBinaryFeatures(options)
	assert.NotNil(t, err)
	assert.Len(t, warnings, 0)
	assert.Equal(t, 1, len(genericBin.Features))
}

func TestTooManyFeatures(t *testing.T) {
	/*Test too many feature values and they should be clipped to 5 values.*/
	genericBin := createGenericBinaryEntity()
	options := NewValidationOptions().WithMaxFeatureCount(5)

	for _, val := range []string{"val1", "val2", "val3", "val4", "val5", "val6", "val7"} {
		genericBin.Features = append(genericBin.Features, BinaryEntityFeature{
			Name:  "GenericFeature" + val,
			Value: val,
			Type:  FeatureString,
			Label: "GenericLabel",
		})
	}

	warnings, err := genericBin.ProcessAndValidateBinaryFeatures(options)
	assert.Nil(t, err)
	assert.Len(t, warnings, 1)
	assert.Equal(t, 5, len(genericBin.Features))
}

func TestTooManyOfTheSameFeatureValue(t *testing.T) {
	/*Test too many feature values for the one feature. Max is set to 2 so only 2 values should still be present because they are all for the same feature.*/
	genericBin := createGenericBinaryEntity()
	options := NewValidationOptions().WithMaxValuesInOneFeatureCount(2)

	for _, val := range []string{"val1", "val2", "val3", "val4", "val5", "val6", "val7"} {
		genericBin.Features = append(genericBin.Features, BinaryEntityFeature{
			Name:  "GenericFeature",
			Value: val,
			Type:  FeatureString,
			Label: "GenericLabel",
		})
	}

	warnings, err := genericBin.ProcessAndValidateBinaryFeatures(options)
	assert.Nil(t, err)
	assert.Len(t, warnings, 1)
	assert.Equal(t, 2, len(genericBin.Features))
	// Ensure that just the trailing values were dropped.
	for _, feat := range genericBin.Features {
		assert.Contains(t, []string{"val1", "val2"}, feat.Value)
	}
}

func TestTooManyOfTheSameFeatureValueComplex(t *testing.T) {
	/*Test too many feature values for the one feature. Also have a second feature mixed in to make sure no index errors when deleting.*/
	genericBin := createGenericBinaryEntity()
	options := NewValidationOptions().WithMaxValuesInOneFeatureCount(2)

	genericBin.Features = append(genericBin.Features, BinaryEntityFeature{
		Name:  "AnotherFeature",
		Value: "CompletelySpecial",
		Type:  FeatureString,
		Label: "GenericLabel",
	})

	for _, val := range []string{"val1", "val2", "val3", "val4", "val5", "val6", "val7"} {
		genericBin.Features = append(genericBin.Features, BinaryEntityFeature{
			Name:  "GenericFeature",
			Value: val,
			Type:  FeatureString,
			Label: "GenericLabel",
		})
	}
	genericBin.Features[4] = BinaryEntityFeature{
		Name:  "AnotherFeature",
		Value: "CompletelySpecial2",
		Type:  FeatureString,
		Label: "GenericLabel",
	}

	warnings, err := genericBin.ProcessAndValidateBinaryFeatures(options)
	assert.Nil(t, err)
	// Dropping too many features.
	assert.Len(t, warnings, 2)
	assert.Equal(t, 4, len(genericBin.Features))
	// Ensure that just the trailing values were dropped.
	for _, feat := range genericBin.Features {
		assert.Contains(t, []string{"val1", "val2", "CompletelySpecial", "CompletelySpecial2"}, feat.Value)
	}
}

func TestDatastreamLabelConversion(t *testing.T) {
	tmp := DataLabelAssemblyline
	tmp2 := &tmp
	require.Equal(t, DataLabelContent.Str(), "content")
	require.Equal(t, DataLabelAssemblyline.Str(), "assemblyline")
	require.Equal(t, tmp.Str(), "assemblyline")
	require.Equal(t, tmp2.Str(), "assemblyline")
}

package events

import (
	"testing"
	"time"

	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/testdata"
	"github.com/goccy/go-json"

	"github.com/stretchr/testify/require"
)

// this takes two of the incoming interface else generics with pointers is real difficult to init
func testLoopStability[T AvroInterface](t *testing.T, ev1 T, ev2 T, filepath string) {
	raw := testdata.GetBytes(filepath)
	err := json.Unmarshal(raw, &ev1)
	require.Nil(t, err, filepath)
	// avro round trip testing
	asavro, err := ev1.ToAvro()
	require.Nil(t, err, filepath)
	err = ev2.FromAvro(asavro)
	require.Nil(t, err, filepath)

	raw2, err := json.Marshal(&ev2)
	require.Nil(t, err, filepath)
	require.JSONEq(t, string(raw), string(raw2), filepath)

	// test checked in avro files
	filepathAvro := filepath + ".avro"
	// to regenerate, edit this to true or delete the desired avro files
	if !testdata.ExistsBytes(filepathAvro) {
		testdata.DumpBytes(filepathAvro, asavro)
	}
	asavro = testdata.GetBytes(filepathAvro)
	err = ev2.FromAvro(asavro)
	require.Nil(t, err, filepath)

	raw2, err = json.Marshal(&ev2)
	require.Nil(t, err, filepath)
	require.JSONEq(t, string(raw), string(raw2), filepath)
}

func TestLoopStability(t *testing.T) {
	// binary
	testLoopStability(t, &BinaryEvent{}, &BinaryEvent{}, "events/binary/augmented_no_extra.json")
	testLoopStability(t, &BinaryEvent{}, &BinaryEvent{}, "events/binary/augmented.json")
	testLoopStability(t, &BinaryEvent{}, &BinaryEvent{}, "events/binary/enriched_w_data.json")
	testLoopStability(t, &BinaryEvent{}, &BinaryEvent{}, "events/binary/enriched_w_info_dict.json")
	testLoopStability(t, &BinaryEvent{}, &BinaryEvent{}, "events/binary/enriched_w_info_list.json")
	testLoopStability(t, &BinaryEvent{}, &BinaryEvent{}, "events/binary/enriched.json")
	testLoopStability(t, &BinaryEvent{}, &BinaryEvent{}, "events/binary/extracted.json")
	testLoopStability(t, &BinaryEvent{}, &BinaryEvent{}, "events/binary/invalid_path.json")
	// delete
	testLoopStability(t, &DeleteEvent{}, &DeleteEvent{}, "events/delete/author.json")
	testLoopStability(t, &DeleteEvent{}, &DeleteEvent{}, "events/delete/link.json")
	testLoopStability(t, &DeleteEvent{}, &DeleteEvent{}, "events/delete/submission-complex.json")
	testLoopStability(t, &DeleteEvent{}, &DeleteEvent{}, "events/delete/submission-simple.json")
	// download
	testLoopStability(t, &DownloadEvent{}, &DownloadEvent{}, "events/download/example1.json")
	// insert
	testLoopStability(t, &InsertEvent{}, &InsertEvent{}, "events/insert/example1.json")
	// plugin
	testLoopStability(t, &PluginEvent{}, &PluginEvent{}, "events/plugin/example1.json")
	// status
	testLoopStability(t, &StatusEvent{}, &StatusEvent{}, "events/status/example1.json")
	testLoopStability(t, &StatusEvent{}, &StatusEvent{}, "events/speed/status.json")
	// generic
	testLoopStability(t, &RetrohuntEvent{}, &RetrohuntEvent{}, "events/retrohunt/example1.json")
}

// check that lists of events can be un/marshalled
func TestLoopStabilityBulk(t *testing.T) {
	testLoopStability(t, &BulkBinaryEvent{}, &BulkBinaryEvent{}, "events/binary/bulk.json")
	testLoopStability(t, &BulkDeleteEvent{}, &BulkDeleteEvent{}, "events/delete/bulk.json")
	testLoopStability(t, &BulkDownloadEvent{}, &BulkDownloadEvent{}, "events/download/bulk.json")
	testLoopStability(t, &BulkInsertEvent{}, &BulkInsertEvent{}, "events/insert/bulk.json")
	testLoopStability(t, &BulkPluginEvent{}, &BulkPluginEvent{}, "events/plugin/bulk.json")
	testLoopStability(t, &BulkStatusEvent{}, &BulkStatusEvent{}, "events/status/bulk.json")
	testLoopStability(t, &BulkRetrohuntEvent{}, &BulkRetrohuntEvent{}, "events/retrohunt/bulk.json")
}

func testBulkLoadNonBulkErrors[TSingle AvroInterface, TBulk AvroInterface](t *testing.T, singleEv TSingle, bulkEv TBulk, filepath string) {
	asavro, err := singleEv.ToAvro()
	require.Nil(t, err, filepath)
	err = bulkEv.FromAvro(asavro)
	require.NotNil(t, err)
}

// Verify that when Avro attempts to load a non-bulk event as a bulk event an error occurs.
// This is needed because sometimes avro doesn't error even if the model is invalid when a model is missing various values.
func TestBulkLoadNonBulkErrors(t *testing.T) {
	testBulkLoadNonBulkErrors(t, &BinaryEvent{ModelVersion: CurrentModelVersion}, &BulkBinaryEvent{}, "events/binary/enriched_w_data.json")
	testBulkLoadNonBulkErrors(t, &DeleteEvent{ModelVersion: CurrentModelVersion}, &BulkDeleteEvent{}, "events/delete/author.json")
	testBulkLoadNonBulkErrors(t, &DownloadEvent{ModelVersion: CurrentModelVersion}, &BulkDownloadEvent{}, "events/download/example1.json")
	testBulkLoadNonBulkErrors(t, &InsertEvent{ModelVersion: CurrentModelVersion}, &BulkInsertEvent{}, "events/insert/example1.json")
	testBulkLoadNonBulkErrors(t, &PluginEvent{ModelVersion: CurrentModelVersion}, &BulkPluginEvent{}, "events/plugin/example1.json")
	testBulkLoadNonBulkErrors(t, &StatusEvent{ModelVersion: CurrentModelVersion}, &BulkStatusEvent{}, "events/status/example1.json")
	testBulkLoadNonBulkErrors(t, &RetrohuntEvent{ModelVersion: CurrentModelVersion}, &BulkRetrohuntEvent{}, "events/retrohunt/example1.json")
}

// Verify that the legacy Avro can upgrade to the expected json.
// this takes two of the incoming interface else generics with pointers is real difficult to init
// this takes two of the incoming interface else generics with pointers is real difficult to init
func testLegacyAvroUpgrades[T AvroInterface](t *testing.T, ev1 T, ev2 T, filepath string) {
	// Test the old avro can be successfully loaded.
	filepathAvro := filepath + ".avro"
	var asavro []byte
	// to regenerate, edit this to true or delete the desired avro files
	if !testdata.ExistsBytes(filepathAvro) {
		testdata.DumpBytes(filepathAvro, asavro)
	}
	asavro = testdata.GetBytes(filepathAvro)
	err := ev2.FromAvro(asavro)
	require.Nil(t, err, filepath)

	avroAsJson, err := json.Marshal(&ev2)
	require.Nil(t, err, filepath)

	// Expected Json for the loaded avro (defaults loaded correctly)
	expectedJson := testdata.GetBytes(filepath)
	require.JSONEq(t, string(expectedJson), string(avroAsJson), filepath)
}

// Verify that legacy version 4 models can be loaded into the newest version of the models.
func TestLegacyV4Stability(t *testing.T) {
	// binary
	testLegacyAvroUpgrades(t, &BinaryEvent{}, &BinaryEvent{}, "events/binary/v4/augmented_no_extra.json")
	testLegacyAvroUpgrades(t, &BinaryEvent{}, &BinaryEvent{}, "events/binary/v4/augmented.json")
	testLegacyAvroUpgrades(t, &BinaryEvent{}, &BinaryEvent{}, "events/binary/v4/enriched_w_data.json")
	testLegacyAvroUpgrades(t, &BinaryEvent{}, &BinaryEvent{}, "events/binary/v4/enriched_w_info_dict.json")
	testLegacyAvroUpgrades(t, &BinaryEvent{}, &BinaryEvent{}, "events/binary/v4/enriched_w_info_list.json")
	testLegacyAvroUpgrades(t, &BinaryEvent{}, &BinaryEvent{}, "events/binary/v4/enriched.json")
	testLegacyAvroUpgrades(t, &BinaryEvent{}, &BinaryEvent{}, "events/binary/v4/extracted.json")
	testLegacyAvroUpgrades(t, &BinaryEvent{}, &BinaryEvent{}, "events/binary/v4/invalid_path.json")
	// delete
	testLegacyAvroUpgrades(t, &DeleteEvent{}, &DeleteEvent{}, "events/delete/v4/author.json")
	testLegacyAvroUpgrades(t, &DeleteEvent{}, &DeleteEvent{}, "events/delete/v4/link.json")
	testLegacyAvroUpgrades(t, &DeleteEvent{}, &DeleteEvent{}, "events/delete/v4/submission-complex.json")
	testLegacyAvroUpgrades(t, &DeleteEvent{}, &DeleteEvent{}, "events/delete/v4/submission-simple.json")
	// download
	testLegacyAvroUpgrades(t, &DownloadEvent{}, &DownloadEvent{}, "events/download/v4/example1.json")
	// insert
	testLegacyAvroUpgrades(t, &InsertEvent{}, &InsertEvent{}, "events/insert/v4/example1.json")
	// plugin
	testLegacyAvroUpgrades(t, &PluginEvent{}, &PluginEvent{}, "events/plugin/v4/example1.json")
	// status
	testLegacyAvroUpgrades(t, &StatusEvent{}, &StatusEvent{}, "events/status/v4/example1.json")
	testLegacyAvroUpgrades(t, &StatusEvent{}, &StatusEvent{}, "events/speed/v4/status.json")
	// generic
	testLegacyAvroUpgrades(t, &RetrohuntEvent{}, &RetrohuntEvent{}, "events/retrohunt/v4/example1.json")
}

// Verify that legacy version 4 Bulk models can be loaded into the newest version of the models.
func TestLegacyV4StabilityBulk(t *testing.T) {
	testLegacyAvroUpgrades(t, &BulkBinaryEvent{}, &BulkBinaryEvent{}, "events/binary/v4/bulk.json")
	testLegacyAvroUpgrades(t, &BulkDeleteEvent{}, &BulkDeleteEvent{}, "events/delete/v4/bulk.json")
	testLegacyAvroUpgrades(t, &BulkDownloadEvent{}, &BulkDownloadEvent{}, "events/download/v4/bulk.json")
	testLegacyAvroUpgrades(t, &BulkInsertEvent{}, &BulkInsertEvent{}, "events/insert/v4/bulk.json")
	testLegacyAvroUpgrades(t, &BulkPluginEvent{}, &BulkPluginEvent{}, "events/plugin/v4/bulk.json")
	testLegacyAvroUpgrades(t, &BulkStatusEvent{}, &BulkStatusEvent{}, "events/status/v4/bulk.json")
	testLegacyAvroUpgrades(t, &BulkRetrohuntEvent{}, &BulkRetrohuntEvent{}, "events/retrohunt/v4/bulk.json")
}

func verifyEncodeDecode[T any](t *testing.T, pre string, expected T, post string) {
	var actual T
	err := json.Unmarshal([]byte(pre), &actual)
	require.Nil(t, err)
	require.Equal(t, expected, actual)
	crushed, err := json.Marshal(actual)
	nice_crushed := string(crushed)
	require.Nil(t, err)
	require.JSONEq(t, nice_crushed, post, "expected\n%v", nice_crushed)
}

func verifyEncodeDecodeBinaryEvent(t *testing.T, pre string, expected *BinaryEvent, post string) {
	var actual BinaryEvent
	err := json.Unmarshal([]byte(pre), &actual)
	require.Nil(t, err)

	// set tracking info
	err = actual.UpdateTrackingFields()
	require.Nil(t, err)
	require.Equal(t, expected, &actual)

	// standard marshal using custom function
	crushed, err := json.Marshal(&actual)
	nice_crushed := string(crushed)
	require.Nil(t, err)
	require.JSONEq(t, nice_crushed, post, "expected\n%v", nice_crushed)
}

func TestEntityData(t *testing.T) {
	verifyEncodeDecode(t,
		`{"label": "test"}`,
		BinaryEntityDatastream{Label: "test"},
		`{"label":"test","size":0,"sha512":"","sha256":"","sha1":"","md5":"","mime":"","magic":""}`,
	)
	verifyEncodeDecode(t,
		`{"label": "test", "size": 999, "sha256": "too long", "file_format": "text/plain"}`,
		BinaryEntityDatastream{Label: "test", Size: 999, Sha256: "too long", FileFormat: "text/plain"},
		`{"label":"test","size":999,"sha512":"","sha256":"too long","sha1":"","md5":"","mime":"","magic":"","file_format":"text/plain"}`,
	)
	verifyEncodeDecode(t,
		`{"identify_version":12, "label": "test", "size": 999, "sha256": "too long", "file_format": "text/plain"}`,
		BinaryEntityDatastream{IdentifyVersion: 12, Label: "test", Size: 999, Sha256: "too long", FileFormat: "text/plain"},
		`{"identify_version":12, "label":"test","size":999,"sha512":"","sha256":"too long","sha1":"","md5":"","mime":"","magic":"","file_format":"text/plain"}`,
	)
}

func TestBinary(t *testing.T) {
	verifyEncodeDecode(t,
		`{"sha256": "test"}`,
		BinaryEntity{Sha256: "test"},
		`{"sha256":"test"}`,
	)
	verifyEncodeDecode(t,
		`{"sha256": "test", "file_extension": "exe"}`,
		BinaryEntity{Sha256: "test", FileExtension: "exe"},
		`{"sha256":"test","file_extension":"exe"}`,
	)
}

func TestBinaryEvent(t *testing.T) {
	atime := time.Date(2012, time.January, 1, 0, 0, 0, 0, time.UTC)
	verifyEncodeDecode(t,
		`{
			"model_version":12,
			"kafka_key":"id",
			"action": "mapped",
			"timestamp": "2012-01-01T00:00:00Z",
			"source": {"name": "id", "path": [], "timestamp": "2012-01-01T00:00:00Z"},
			"author": {"name": "name"},
			"entity": {"sha256":"id"},
			"flags":{"expedite": true}
		}`,
		BinaryEvent{
			ModelVersion: 12,
			Author:       EventAuthor{Name: "name", Version: "", Category: "", Security: ""},
			KafkaKey:     "id",
			Timestamp:    atime,
			Action:       ActionMapped,
			Source:       EventSource{Name: "id", References: map[string]string(nil), Security: "", Path: []EventSourcePathNode{}, Timestamp: atime},
			Entity:       BinaryEntity{Sha256: "id"},
			Dequeued:     "",
			Flags:        BinaryFlags{Expedite: true},
		},
		`{
			"model_version":12,
			"kafka_key":"id",
			"action": "mapped",
			"timestamp": "2012-01-01T00:00:00Z",
			"source": {"name": "id", "path": [], "timestamp": "2012-01-01T00:00:00Z"},
			"author": {"name": "name"},
			"entity": {"sha256":"id"},
			"flags":{"expedite": true}
		}`,
	)
	// wonky time
	atime = time.Date(2012, time.January, 1, 0, 0, 0, 542543000, time.Local)
	verifyEncodeDecode(t,
		`{
			"model_version":12,
			"kafka_key":"id",
			"action": "mapped",
			"timestamp": "2012-01-01T00:00:00.542543+00:00",
			"source": {"name": "id", "path": [], "timestamp": "2012-01-01T00:00:00.542543+00:00"},
			"author": {"name": "name"},
			"entity": {"sha256":"id"}
		}`,
		BinaryEvent{
			ModelVersion: 12,
			Author:       EventAuthor{Name: "name", Version: "", Category: "", Security: ""},
			KafkaKey:     "id",
			Timestamp:    atime,
			Action:       ActionMapped,
			Source:       EventSource{Name: "id", References: map[string]string(nil), Security: "", Path: []EventSourcePathNode{}, Timestamp: atime},
			Entity:       BinaryEntity{Sha256: "id"},
			Dequeued:     "",
		}, `{
			"model_version":12,
			"kafka_key":"id",
			"action": "mapped",
			"timestamp": "2012-01-01T00:00:00.542543Z",
			"flags": {},
			"source": {"name": "id", "path": [], "timestamp": "2012-01-01T00:00:00.542543Z"},
			"author": {"name": "name"},
			"entity": {"sha256":"id"}
		}`,
	)
	// no version
	atime = time.Date(2012, time.January, 1, 0, 0, 0, 0, time.UTC)
	verifyEncodeDecode(t,
		`{
			"kafka_key":"id",
			"action": "mapped",
			"timestamp": "2012-01-01T00:00:00Z",
			"source": {"name": "id", "path": [], "timestamp": "2012-01-01T00:00:00Z"},
			"author": {"name": "name"},
			"entity": {"sha256":"id"}
		}`,

		BinaryEvent{
			ModelVersion: 0,
			Author:       EventAuthor{Name: "name", Version: "", Category: "", Security: ""},
			KafkaKey:     "id",
			Timestamp:    atime,
			Source:       EventSource{Name: "id", References: map[string]string(nil), Security: "", Path: []EventSourcePathNode{}, Timestamp: atime},
			Action:       ActionMapped,
			Entity:       BinaryEntity{Sha256: "id"},
			Dequeued:     "",
		}, `{
			"kafka_key":"id",
			"action": "mapped",
			"timestamp": "2012-01-01T00:00:00Z",
			"source": {"name": "id", "path": [], "timestamp": "2012-01-01T00:00:00Z"},
			"author": {"name": "name"},
			"flags": {},
			"entity": {"sha256":"id"}
		}`,
	)
}

func TestBinaryEvent2(t *testing.T) {
	atime := time.Date(2012, time.January, 1, 0, 0, 0, 0, time.UTC)
	verifyEncodeDecodeBinaryEvent(t,
		`{
			"model_version":12,
			"kafka_key":"myevent1",
			"action": "mapped",
			"timestamp": "2012-01-01T00:00:00Z",
			"source": {"name": "source", "references": {"taskid":"lemon"}, "path": [{"author":{"name":"apple"}, "sha256": "parent"},{"author":{"name":"apple"}, "sha256": "child"}], "timestamp": "2012-01-01T00:00:00Z"},
			"author": {"name": "name"},
			"entity": {"sha256":"child"}
		}`,
		&BinaryEvent{
			ModelVersion: 12,
			Author:       EventAuthor{Name: "name", Version: "", Category: "", Security: ""},
			KafkaKey:     "myevent1",
			Timestamp:    atime,
			Source: EventSource{
				Name:       "source",
				References: map[string]string{"taskid": "lemon"},
				Security:   "",
				Path:       []EventSourcePathNode{{Author: EventAuthor{Name: "apple"}, Sha256: "parent"}, {Author: EventAuthor{Name: "apple"}, Sha256: "child"}},
				Timestamp:  atime,
			},
			Action:                ActionMapped,
			Entity:                BinaryEntity{Sha256: "child"},
			Dequeued:              "",
			TrackSourceReferences: "source.5febce412be5ba9ce929ff9ad16bdd50",
			TrackLinks:            []string{"parent.child..apple."},
			TrackAuthors:          []string{".apple.", ".apple."},
		},
		`{
			"track_source_references":"source.5febce412be5ba9ce929ff9ad16bdd50",
			"track_links":["parent.child..apple."],
			"track_authors":[".apple.",".apple."],
			"kafka_key":"myevent1",
			"author":{"name":"name"},
			"action":"mapped",
			"flags": {},
			"model_version":12,
			"timestamp":"2012-01-01T00:00:00Z",
			"source":{"name":"source", "references": {"taskid":"lemon"},"path":[
				{"author":{"name":"apple"},"action":"","sha256":"parent","timestamp":"0001-01-01T00:00:00Z"}, 
				{"author":{"name":"apple"},"action":"","sha256":"child","timestamp":"0001-01-01T00:00:00Z"}
			],"timestamp":"2012-01-01T00:00:00Z"},
			"entity":{"sha256":"child"}
		}`,
	)
}

func TestSanity(t *testing.T) {
	data := testdata.GetBytes("events/sanity.json")
	var event BinaryEvent
	err := json.Unmarshal(data, &event)
	require.Nil(t, err)
	require.Equal(t, event.KafkaKey, "b76bbb2bf5a5cd78c1f85adf551bb90e")
}

func TestEventTypesFromStrings(t *testing.T) {
	var ret []BinaryAction
	var err error

	ret, err = ActionsFromStrings([]string{"status_update", "extracted", "plugin_started"})
	require.NotNil(t, err)
	require.Nil(t, ret)

	ret, err = ActionsFromStrings([]string{"mapped", "manual_insert"})
	require.NotNil(t, err)
	require.Nil(t, ret)

	ret, err = ActionsFromStrings([]string{})
	require.Nil(t, err)
	require.Equal(t, ret, []BinaryAction{})

	ret, err = ActionsFromStrings([]string{"extracted", "augmented", "mapped"})
	require.Nil(t, err)
	require.Equal(t, ret, []BinaryAction{ActionExtracted, ActionAugmented, ActionMapped})
}

// ensure that alterations of GetBase are reflected in regular model
func TestAlterThroughGetBase(t *testing.T) {
	be := createBinaryEvent()
	var ev EventInterface = be
	base := ev.GetBase()
	*base.KafkaKey = "tttt"
	require.Equal(t, be.KafkaKey, "tttt")
	require.Equal(t, *base.KafkaKey, "tttt")
	*base.KafkaKey = "test"
	require.Equal(t, be.KafkaKey, "test")
	require.Equal(t, *base.KafkaKey, "test")
}

// test that events are compatible with the event interface
func TestInterface(t *testing.T) {
	var ev EventInterface
	ev = &BinaryEvent{ModelVersion: 5}
	require.Equal(t, *ev.GetBase().ModelVersion, uint32(5))
	ev = &DeleteEvent{ModelVersion: 5}
	require.Equal(t, *ev.GetBase().ModelVersion, uint32(5))
	ev = &RetrohuntEvent{ModelVersion: 5}
	require.Equal(t, *ev.GetBase().ModelVersion, uint32(5))
	ev = &InsertEvent{ModelVersion: 5}
	require.Equal(t, *ev.GetBase().ModelVersion, uint32(5))
	ev = &PluginEvent{ModelVersion: 5}
	require.Equal(t, *ev.GetBase().ModelVersion, uint32(5))
	ev = &StatusEvent{ModelVersion: 5}
	require.Equal(t, *ev.GetBase().ModelVersion, uint32(5))
	ev = &RetrohuntEvent{ModelVersion: 5}
	require.Equal(t, *ev.GetBase().ModelVersion, uint32(5))
	// bulk
	var av AvroInterface
	av = &BulkBinaryEvent{}
	require.NotNil(t, av)
	av = &BulkDeleteEvent{}
	require.NotNil(t, av)
	av = &BulkRetrohuntEvent{}
	require.NotNil(t, av)
	av = &BulkInsertEvent{}
	require.NotNil(t, av)
	av = &BulkPluginEvent{}
	require.NotNil(t, av)
	av = &BulkStatusEvent{}
	require.NotNil(t, av)
	av = &BulkRetrohuntEvent{}
	require.NotNil(t, av)
}

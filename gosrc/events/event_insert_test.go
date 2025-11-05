package events

import (
	"testing"

	testdata "github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/testdata"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
)

func TestEventInsert1(t *testing.T) {
	data := testdata.GetBytes("events/insert/example1.json")
	var ev InsertEvent
	err := json.Unmarshal(data, &ev)
	require.Nil(t, err)

	require.Equal(t, ev.Entity.ParentSha256, "1111111111111111111111111111")
	require.Equal(t, ev.Entity.Child.Sha256, "2222222222222222222222222222222222222222222222222")
	require.Equal(t, ev.Entity.Child, BinaryEntity{Sha256: "2222222222222222222222222222222222222222222222222"})
	require.Equal(t, ev.TrackAuthor, "")
	require.Equal(t, ev.TrackLink, "")

	// check tracking fields
	ev.UpdateTrackingFields()
	require.Equal(t, ev.TrackAuthor, "plugin.some_enhancer.111")
	require.Equal(t, ev.TrackLink, "1111111111111111111111111111.2222222222222222222222222222222222222222222222222.plugin.some_enhancer.111")
}

// Data Label Validation tests.
func TestInsertEventDataLabelValid(t *testing.T) {
	data := testdata.GetBytes("events/insert/example1.json")
	var ev InsertEvent
	err := json.Unmarshal(data, &ev)
	require.Nil(t, err)

	ev.Entity.Child.FileFormatLegacy = "TEXT"
	ev.Entity.Child.FileFormat = "txt"
	// Check Valid Label
	ev.Entity.Child.Datastreams = append(ev.Entity.Child.Datastreams, BinaryEntityDatastream{IdentifyVersion: 0, Label: DataLabelContent})
	require.Nil(t, ev.CheckValid())

	// Check Invalid Label
	ev.Entity.Child.Datastreams = append(ev.Entity.Child.Datastreams, BinaryEntityDatastream{IdentifyVersion: 0, Label: "bad-label-really-it-should-error"})
	require.NotNil(t, ev.CheckValid())
}

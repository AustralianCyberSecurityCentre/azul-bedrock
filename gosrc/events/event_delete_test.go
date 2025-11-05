package events

import (
	"testing"
	"time"

	testdata "github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/testdata"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
)

func parseDelete(t *testing.T, filename string) *DeleteEvent {
	data := testdata.GetBytes(filename)
	var ev DeleteEvent
	err := json.Unmarshal(data, &ev)
	require.Nil(t, err)
	return &ev

}

func TestEventDelete1(t *testing.T) {
	testTime, err := time.Parse(time.RFC3339, "2021-03-29T00:24:08.54928Z")
	require.Nil(t, err)
	ev := parseDelete(t, "events/delete/author.json")
	require.Equal(t, ev.Entity, DeleteEntity{
		Reason: "don't like the author!",
		Author: DeleteEntityAuthor{TrackAuthor: "plugin.myplugin.1", Timestamp: testTime},
	})
	require.Equal(t, ev.Action, "author")

	ev = parseDelete(t, "events/delete/link.json")
	require.Equal(t, ev.Entity, DeleteEntity{
		Reason: "was a bad file",
		Link:   DeleteEntityLink{TrackLink: "parentid.childid.authorinfo"},
	})
	require.Equal(t, ev.Action, "link")

	ev = parseDelete(t, "events/delete/submission-simple.json")
	require.Equal(t, ev.Entity, DeleteEntity{
		Reason:     "was a bad file",
		Submission: DeleteEntitySubmission{TrackSourceReferences: "test.123912658fg78d679"},
	})
	require.Equal(t, ev.Action, "submission")

	ev = parseDelete(t, "events/delete/submission-complex.json")
	require.Equal(t, ev.Entity, DeleteEntity{
		Reason: "was a bad file",
		Submission: DeleteEntitySubmission{
			TrackSourceReferences: "test.123912658fg78d679",
			Timestamp:             testTime,
		},
	})
	require.Equal(t, ev.Action, "submission")
	require.Nil(t, ev.CheckValid())

}

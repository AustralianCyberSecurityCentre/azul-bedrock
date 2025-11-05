package events

import (
	"testing"

	testdata "github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/testdata"
	"github.com/goccy/go-json"

	"github.com/stretchr/testify/require"
)

func TestEventStatus1(t *testing.T) {
	data := testdata.GetBytes("events/status/example1.json")
	var ev StatusEvent
	err := json.Unmarshal(data, &ev)
	require.Nil(t, err)
	require.Equal(t, ev.Entity.Input.Dequeued, "xabzz")
	require.Equal(t, ev.Entity.Status, "done")
	require.Equal(t, ev.Entity.Error, "a bad thing happened, oh dear!")
	require.Nil(t, ev.CheckValid())
}

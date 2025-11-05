package events

import (
	"testing"

	testdata "github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/testdata"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
)

func TestEventPlugin1(t *testing.T) {
	data := testdata.GetBytes("events/plugin/example1.json")
	var ev PluginEvent
	err := json.Unmarshal(data, &ev)
	require.Nil(t, err)
	require.Equal(t, ev.Entity.Name, "myplugin")
	require.Equal(t, ev.Entity.Version, "99.9.9")
	require.Equal(t, ev.Entity.Features[0].Description, "a feature")
	require.Equal(t, ev.Entity.Config["deployment_key"], "adeploymentkey")
}

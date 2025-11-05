package events

import (
	"testing"

	testdata "github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/testdata"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
)

func TestEventDownload(t *testing.T) {
	data := testdata.GetBytes("events/download/example1.json")
	var ev DownloadEvent
	err := json.Unmarshal(data, &ev)
	require.Nil(t, err)
	require.Nil(t, ev.CheckValid())

	require.Nil(t, err)
	require.Equal(t, ev.Entity.Hash, "86795438796")
	require.Equal(t, ev.Entity.Category, "athing")
	require.Equal(t, ev.Entity.CategoryQuota, uint32(100))
}

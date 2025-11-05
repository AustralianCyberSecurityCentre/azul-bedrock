package models

import (
	"testing"

	testdata "github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/testdata"
	"github.com/stretchr/testify/require"
)

func TestSourcesLoad(t *testing.T) {
	data := string(testdata.GetBytes("sources/sources1.yaml"))
	parsed, err := ParseSourcesYaml(data)
	require.Nil(t, err)
	require.Equal(t, len(parsed.Sources), 7)
	allsrc := []string{}
	for x := range parsed.Sources {
		allsrc = append(allsrc, x)
	}
	require.ElementsMatch(t, allsrc, []string{"testing", "incidents", "reporting", "samples", "tasking", "virustotal", "watch"})
	require.Equal(t, parsed.Sources["testing"].Description, "Files submitted during testing of Azul")
	require.Equal(t, parsed.Sources["testing"].ExcludeFromBackup, true)
	require.Equal(t, parsed.Sources["virustotal"].ExcludeFromBackup, true)
	require.Equal(t, parsed.Sources["tasking"].ExcludeFromBackup, false)

	// Verify the ExpireEventsAfter data loads correctly.
	require.Equal(t, parsed.Sources["testing"].ExpireEventsAfter, "7 days")
	require.Equal(t, parsed.Sources["testing"].ExpireEventsAfterMs, int64(604800000))
	require.EqualValues(t, map[string]string{"retention.ms": "604800000", "cleanup.policy": "delete", "segment.bytes": "1073741824"}, parsed.Sources["testing"].Kafka.Config)

	require.Equal(t, parsed.Sources["incidents"].ExpireEventsAfter, "0")
	require.Equal(t, parsed.Sources["incidents"].ExpireEventsAfterMs, int64(-1))
	require.EqualValues(t, map[string]string{"retention.ms": "-1", "cleanup.policy": "compact"}, parsed.Sources["incidents"].Kafka.Config)

	require.Equal(t, parsed.Sources["reporting"].ExpireEventsAfter, "0")
	require.Equal(t, parsed.Sources["reporting"].ExpireEventsAfterMs, int64(-1))
	require.EqualValues(t, map[string]string{"retention.ms": "-1", "cleanup.policy": "compact"}, parsed.Sources["reporting"].Kafka.Config)

	require.Equal(t, parsed.Sources["virustotal"].ExpireEventsAfter, "3 months")
	require.Equal(t, parsed.Sources["virustotal"].ExpireEventsAfterMs, int64(7776000000))
	require.EqualValues(t, map[string]string{"retention.ms": "7776000000", "cleanup.policy": "delete"}, parsed.Sources["virustotal"].Kafka.Config)
}

func TestSourcesLoadBad(t *testing.T) {
	data := string(testdata.GetBytes("sources/sources_bad_expiry_inner_spaces.yaml"))
	parsed, err := ParseSourcesYaml(data)
	require.NotNil(t, err)
	require.Equal(t, parsed, SourcesConf{})

	data = string(testdata.GetBytes("sources/sources_bad_expiry_no_number.yaml"))
	parsed, err = ParseSourcesYaml(data)
	require.NotNil(t, err)
	require.Equal(t, parsed, SourcesConf{})

	data = string(testdata.GetBytes("sources/sources_bad_expiry_outer_spaces_lead.yaml"))
	parsed, err = ParseSourcesYaml(data)
	require.NotNil(t, err)
	require.Equal(t, parsed, SourcesConf{})

	data = string(testdata.GetBytes("sources/sources_bad_expiry_outer_spaces_trail.yaml"))
	parsed, err = ParseSourcesYaml(data)
	require.NotNil(t, err)
	require.Equal(t, parsed, SourcesConf{})
}

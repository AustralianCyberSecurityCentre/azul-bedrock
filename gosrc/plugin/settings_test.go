package plugin

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFluentSetSettingsApis(t *testing.T) {
	defaultSettings := PluginSettings{}
	// This is the most complex setter
	defaultSettings.WithContentFilterDataTypes([]string{"abc"})
	require.Equal(t, map[string][]string{"content": {"abc"}}, defaultSettings.FilterDataTypes)

	// Test a couple more
	defaultSettings.WithDeploymentKey("customkey")
	require.Equal(t, "customkey", defaultSettings.DeploymentKey)

	defaultSettings.WithRequireHistoric(false)
	require.False(t, defaultSettings.RequireHistoric)
	defaultSettings.WithRequireHistoric(true)
	require.True(t, defaultSettings.RequireHistoric)
}

// Test that verifies the plugin settings are converted to a map with their konaf keys.
func TestConvertSettingsToMap(t *testing.T) {
	defaultSettings := PluginSettings{}
	result := defaultSettings.convertToMap()
	require.Equal(t, map[string]string{
		"data_url":                 "\"\"",
		"deployment_key":           "\"\"",
		"depth_limit":              "0",
		"events_url":               "\"\"",
		"filter_allow_event_types": "null",
		"filter_data_types":        "null",
		"filter_max_content_size":  "0",
		"filter_min_content_size":  "0",
		"filter_self":              "false",
		"heartbeat_interval":       "0",
		"max_value_length":         "0",
		"max_values_per_feature":   "0",
		"not_ready_backoff":        "0",
		"request_retry_count":      "0",
		"request_timeout":          "0",
		"require_expedite":         "false",
		"require_historic":         "false",
		"require_live":             "false",
		"run_timeout":              "0",
	},
		result,
	)

}

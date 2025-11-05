package plugin

import (
	"encoding/json"
	"log"
	"reflect"
	"strings"

	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/events"
	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/settings"
	"github.com/fatih/structs"
	"github.com/go-viper/mapstructure/v2"
)

type PluginSettings struct {
	// dispatcher to use for event interaction
	PluginEventsUrl string `koanf:"plugin_events_url"`
	// dispatcher to use for file data interaction
	PluginDataUrl string `koanf:"plugin_data_url"`

	// Max seconds between heartbeat status messages.
	HeartbeatIntervalSeconds int `koanf:"plugin_heartbeat_interval"`
	// Seconds a plugin can run on a single sample before being aborted, 0 to never time out.
	PluginRunTimeout int `koanf:"plugin_run_timeout"`

	// Seconds to wait before timing out API requests.
	RequestTimeout int `koanf:"plugin_request_timeout"`
	// Times to retry API requests before worker dies.
	RequestRetryCount int `koanf:"plugin_request_retry_count"`
	// Hard limit of values for a single feature.
	MaxValuesPerFeature int `koanf:"plugin_max_values_per_feature"`
	// Hard limit of feature value length (opensearch limit is 32766)
	MaxValueLength int `koanf:"plugin_max_value_length"`
	// Max event processing depth before auto opting out, -1 to disable.
	DepthLimit int `koanf:"plugin_depth_limit"`
	// Seconds to sleep if plugin was not ready to receive events.
	NotReadyBackoffSeconds int `koanf:"plugin_not_ready_backoff"`
	// A unique key for all instances of this plugin, typically matching the parent deployment.
	DeploymentKey string `koanf:"plugin_deployment_key"`
	// Filter out historic events and only get live events.
	RequireExpedite bool `koanf:"plugin_require_expedite"`
	RequireLive     bool `koanf:"plugin_require_live"`
	RequireHistoric bool `koanf:"plugin_require_historic"`
	// Filter out input events above this size
	FilterMaxContentSize settings.HumanReadableBytes `koanf:"plugin_filter_max_content_size"`
	// Filter out input events below this size
	FilterMinContentSize settings.HumanReadableBytes `koanf:"plugin_filter_min_content_size"`
	// Filter allowed events such as only processing extracted events or only processing sourced events.
	FilterAllowEventTypes []events.BinaryAction `koanf:"plugin_filter_allow_event_types"`
	// Filter out input events published by this plugin
	FilterSelf bool `koanf:"plugin_filter_self"`
	// Filter to only accept data streams with specific labels.
	FilterDataTypes map[string][]string `koanf:"plugin_filter_data_types"`
}

var defaults = PluginSettings{
	PluginEventsUrl:          "",
	PluginDataUrl:            "",
	HeartbeatIntervalSeconds: 30,
	PluginRunTimeout:         600,
	RequestTimeout:           15,
	RequestRetryCount:        3,
	MaxValuesPerFeature:      1000,
	MaxValueLength:           4000,
	DepthLimit:               10,
	NotReadyBackoffSeconds:   5,
	DeploymentKey:            "",
	RequireExpedite:          true,
	RequireLive:              true,
	RequireHistoric:          true,
	FilterMaxContentSize:     0,
	FilterMinContentSize:     0,
	FilterAllowEventTypes:    []events.BinaryAction{events.ActionSourced, events.ActionExtracted},
	FilterSelf:               false,
	FilterDataTypes:          map[string][]string{},
}

// --- Most common selection of pluginSettings people modify.

// Max seconds between heartbeat status messages.
func (ps *PluginSettings) WithHeartbeatInterval(seconds int) *PluginSettings {
	ps.HeartbeatIntervalSeconds = seconds
	return ps
}

// Seconds a plugin can run on a single sample before being aborted, 0 to never time out.
func (ps *PluginSettings) WithPluginRunTimeout(seconds int) *PluginSettings {
	ps.PluginRunTimeout = seconds
	return ps
}

// Seconds to wait for dispatcher before timing out API requests.
func (ps *PluginSettings) WithDispatcherRequestTimeout(seconds int) *PluginSettings {
	ps.RequestTimeout = seconds
	return ps
}

// Hard limit of values for a single feature.
func (ps *PluginSettings) WithMaxValuesPerFeature(maxValue int) *PluginSettings {
	ps.MaxValuesPerFeature = maxValue
	return ps
}

// Hard limit of feature value length (opensearch limit is 32766)
func (ps *PluginSettings) WithMaxValueLength(maxValue int) *PluginSettings {
	ps.MaxValueLength = maxValue
	return ps
}

// Allows you to specify the deployment key setting although it is recommended to use the default which is derived from the plugin name.
func (ps *PluginSettings) WithDeploymentKey(deploymentKey string) *PluginSettings {
	ps.DeploymentKey = deploymentKey
	return ps
}

// Allows to receive expedite events
func (ps *PluginSettings) WithRequireExpedite(on bool) *PluginSettings {
	ps.RequireExpedite = on
	return ps
}

// Allows to receive live events
func (ps *PluginSettings) WithRequireLive(on bool) *PluginSettings {
	ps.RequireLive = on
	return ps
}

// Allows to receive historic events
func (ps *PluginSettings) WithRequireHistoric(on bool) *PluginSettings {
	ps.RequireHistoric = on
	return ps
}

// Filter out input events above this size
func (ps *PluginSettings) WithFilterMaxContentSize(maxSize settings.HumanReadableBytes) *PluginSettings {
	ps.FilterMaxContentSize = maxSize
	return ps
}

// Filter out input events below this size
func (ps *PluginSettings) WithFilterMinContentSize(minSize settings.HumanReadableBytes) *PluginSettings {
	ps.FilterMinContentSize = minSize
	return ps
}

// Enable or disable filter out input events published by this plugin
func (ps *PluginSettings) WithFilterSelf(filterOutSelf bool) *PluginSettings {
	ps.FilterSelf = filterOutSelf
	return ps
}

/*
Filter labels expected values is something like this:

	{
		"content": ["images/","executables/"],
		"safe_png": ["images/bmp"],
		"assemblyline": [],
	}

which for the given input label you only get the specified file formats.
*/
func (ps *PluginSettings) WithFilterDataTypes(labelToFileFormatFilter map[string][]string) *PluginSettings {
	ps.FilterDataTypes = labelToFileFormatFilter
	return ps
}

/*
Allows you to filter the allowed input file formats for just the Content stream.
If you want to filter for more than just the content stream use WithFilterDataTypes.
Expected value is something like (refer to bedrock identify for full list of file types):

	[]string{
		// Windows exe
		"executable/windows/",
		// Non windows exe
		"executable/dll32",
		"executable/pe32",
		// Linux elf
		"executable/linux/elf64",
		"executable/linux/elf32",
		"executable/mach-o",
	}

	[]string{"images/"}
*/
func (ps *PluginSettings) WithContentFilterDataTypes(fileFormatFilterList []string) *PluginSettings {
	filter := map[string][]string{
		events.DataLabelContent.Str(): fileFormatFilterList,
	}
	ps.FilterDataTypes = filter
	return ps
}

// Get a copy of the default plugin
func NewDefaultPluginSettings() *PluginSettings {
	copyOfDefaults := defaults
	return &copyOfDefaults
}

// Parse plugin settings with optional overrides
func parsePluginSettings(defaultSettings *PluginSettings) *PluginSettings {
	return settings.ParseSettings(*defaultSettings, "", []mapstructure.DecodeHookFunc{settings.HumanReadableBytesHookFunc()})
}

/*Convert setting into a map, similar to what azul-runner would have.*/
func (s *PluginSettings) convertToMap() map[string]string {
	result := map[string]string{}
	mappedSettings := structs.Map(s)
	pluginSettingType := reflect.TypeOf(*s)

	for key, val := range mappedSettings {
		val, err := json.Marshal(val)
		if err != nil {
			log.Fatalf("Failed to marshal setting %v into json! error: %v", val, err)
		}
		// Ensure the key value in the map is the konaf value not the variable name in the struct.
		// This is required for opensearch/azul-runner compatibility.
		// As metastore reads some of these config values in the form that azul-runner has them defined.
		field, ok := pluginSettingType.FieldByName(key)
		if !ok {
			log.Fatalf("Failed to convert setting key '%s' to appropriate koanf tag value", key)
		}
		configKeyVal := field.Tag.Get("koanf")
		if configKeyVal == "" {
			log.Fatalf("No konaf value found when converting setting key '%s' ensure the konaf value is set.", key)
		}
		keyWithoutPrefix := strings.Replace(configKeyVal, "plugin_", "", 1)
		result[keyWithoutPrefix] = string(val)

	}
	return result
}

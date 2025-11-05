package models

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Information for a particular data source for Azul binaries.
type SourceItem struct {
	IconClass  string `yaml:"icon_class"`
	References []struct {
		Name        string `yaml:"name"`
		Required    bool   `yaml:"required"`
		Description string `yaml:"description"`
		Highlight   bool   `yaml:"highlight"`
	}
	Kafka struct {
		NumPartitions     int               `yaml:"numpartitions"`
		Replicationfactor int               `yaml:"replicationfactor"`
		Config            map[string]string `yaml:"config"`
	}
	Description string `yaml:"description"`
	// time box indices
	PartitionUnit string `yaml:"partition_unit"`
	// Delete events out of Kafka after the provided duration
	ExpireEventsAfter string `yaml:"expire_events_after"`
	// Delete events out of kafka and Opensearch after the duration in ms.
	ExpireEventsAfterMs int64
	// override settings for the elastic indices associated with the source
	Elastic map[string]any `yaml:"elastic"`
	// if true, the source will not be backed up with azul-backup
	ExcludeFromBackup bool `yaml:"exclude_from_backup"`
}

type SourcesConf struct {
	Sources map[string]SourceItem
}

// Calculate how many ms for expiry if the input is 0 or less returns -1.
func calculateExpiryMs(expireEvents string) (int64, error) {
	if expireEvents == "0" {
		return -1, nil
	}
	unitDurationList := strings.Split(expireEvents, " ")
	var duration int64
	var err error
	// Split the string <number> <unit> e.g 4 days or 7 months
	if len(unitDurationList) != 2 {
		return 0, fmt.Errorf("expire_events_after did not split into '2' elements as expected actual number was '%d', split list was %v for input value %s", len(unitDurationList), unitDurationList, expireEvents)
	}
	if duration, err = strconv.ParseInt(unitDurationList[0], 10, 64); err != nil {
		return 0, fmt.Errorf("provided duration '%s' was not a valid integer (innerError: %v)", unitDurationList[0], err)
	}
	unit := unitDurationList[1]

	// Multiplier to convert days into milliseconds using: number_of_units * hours * minutes * seconds * milliseconds
	multiplier := duration * 24 * 60 * 60 * 1000
	// Converts any units into days
	switch unit {
	case "days":
		return int64(multiplier) * 1, nil
	case "weeks":
		return int64(multiplier) * 7, nil
	case "months":
		return int64(multiplier) * 30, nil
	case "years":
		return int64(multiplier) * 365, nil
	}
	return -1, fmt.Errorf("provided unit '%s' was not in the allowed values days, weeks, months, years  (innerError: %v)", unit, err)
}

func ParseSourcesYaml(yamlContent string) (SourcesConf, error) {
	var sourcesConf SourcesConf
	err := yaml.Unmarshal([]byte(yamlContent), &sourcesConf)
	for sourceName, sourceValue := range sourcesConf.Sources {
		if sourceValue.ExpireEventsAfter == "" {
			sourceValue.ExpireEventsAfter = "0"
		}
		expireAfterMs, err := calculateExpiryMs(sourceValue.ExpireEventsAfter)
		if err != nil {
			return SourcesConf{}, fmt.Errorf("couldn't parse the expire_events_after setting with value '%s' for source '%s' with error %v", sourceValue.ExpireEventsAfter, sourceName, err)
		}
		if sourceValue.Kafka.Config == nil {
			sourceValue.Kafka.Config = make(map[string]string)
		}
		if expireAfterMs <= 0 {
			expireAfterMs = -1
			sourceValue.Kafka.Config["cleanup.policy"] = "compact"
		} else {
			sourceValue.Kafka.Config["cleanup.policy"] = "delete"
		}

		sourceValue.Kafka.Config["retention.ms"] = fmt.Sprintf("%d", expireAfterMs)
		sourceValue.ExpireEventsAfterMs = expireAfterMs
		sourcesConf.Sources[sourceName] = sourceValue
	}
	return sourcesConf, err
}

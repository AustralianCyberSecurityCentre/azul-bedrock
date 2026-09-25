package models

import (
	"time"

	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v13/gosrc/events"
)

// All models here have to map to models_restapi/alert.py

// Redis key for the storing of the alerters configuration.
const ALERTER_CONFIG_KEY = "config"

// Redis key for the storing of active alerts in redis
const ALERTER_ALERT_KEY = "alerts"

// Redis Alerter database ID.
const ALERTER_DB_ID = 4

// Alert rule used to match against events, if a match is achieved an alert is raised.
type AlertRule struct {
	Id                       string              `json:"id"`
	WebhookId                string              `json:"webhook_id"`
	AlertMessage             string              `json:"alert_message"`
	Status                   events.StatusType   `json:"status,omitempty"`
	EventType                events.BinaryAction `json:"event_type,omitempty"`
	PluginName               string              `json:"plugin_name,omitempty"`
	PluginVersion            string              `json:"plugin_version,omitempty"`
	SourceName               string              `json:"source_name,omitempty"`
	SourceReferenceKeyValues map[string]string   `json:"source_reference_key_values,omitempty"`
	FeatureNameValues        map[string]string   `json:"feature_name_values,omitempty"`
}

// All loaded alert rules stored in Redis with the compile time so endpoints can check if there are any new rules.
type LoadedRules struct {
	Rules            []AlertRule `json:"rules"`
	RulesCompileTime time.Time   `json:"rules_compile_time"`
}

// Format for an alert hit that should be stored in redis when an alert hit is made.
type AlertHit struct {
	RuleId       string `json:"rule_id"`
	AlertMessage string `json:"alert_message"`
	AlertAttempt int    `json:"alert_attempt,omitempty"`
	Sha256       string `json:"sha256"`
}

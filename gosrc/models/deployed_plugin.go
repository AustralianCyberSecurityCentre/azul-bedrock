/*
Provides live metadata about a plugin that has been deployed.
*/
package models

// Given a deployment key, information about the latest deployment of a plugin.
type DeployedPlugin struct {
	KafkaPrefix string `json:"kafka_prefix" avro:"kafka_prefix"`
}

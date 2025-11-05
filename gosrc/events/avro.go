package events

import (
	"bytes"
	"embed"
	"fmt"
	"path"
	"strings"

	"github.com/hamba/avro/v2"
)

//go:embed schemas
var rawSchemas embed.FS

// avro schemas for various events
type AvroSchemaType string

// All avro schemas available
const (
	SchemaBinary    AvroSchemaType = "BinaryEvent"
	SchemaDelete    AvroSchemaType = "DeleteEvent"
	SchemaDownload  AvroSchemaType = "DownloadEvent"
	SchemaRetrohunt AvroSchemaType = "RetrohuntEvent"
	SchemaInsert    AvroSchemaType = "InsertEvent"
	SchemaPlugin    AvroSchemaType = "PluginEvent"
	SchemaStatus    AvroSchemaType = "StatusEvent"

	SchemaBulkBinary    AvroSchemaType = "BulkBinaryEvent"
	SchemaBulkDelete    AvroSchemaType = "BulkDeleteEvent"
	SchemaBulkDownload  AvroSchemaType = "BulkDownloadEvent"
	SchemaBulkRetrohunt AvroSchemaType = "BulkRetrohuntEvent"
	SchemaBulkInsert    AvroSchemaType = "BulkInsertEvent"
	SchemaBulkPlugin    AvroSchemaType = "BulkPluginEvent"
	SchemaBulkStatus    AvroSchemaType = "BulkStatusEvent"
)

// Full list of available schemas used when loading schemas into the system.
func getAllSchemaTypes() []AvroSchemaType {
	return []AvroSchemaType{
		SchemaBinary,
		SchemaDelete,
		SchemaDownload,
		SchemaRetrohunt,
		SchemaInsert,
		SchemaPlugin,
		SchemaStatus,
		SchemaBulkBinary,
		SchemaBulkDelete,
		SchemaBulkDownload,
		SchemaBulkRetrohunt,
		SchemaBulkInsert,
		SchemaBulkPlugin,
		SchemaBulkStatus,
	}
}

type AvroSchemaVersion uint32

// When adding a new Version add it here
const (
	SchemaVersionLatest AvroSchemaVersion = AvroSchemaVersion(CurrentModelVersion) // Ensure CurrentModelVersion is updated! (in python as well 'CURRENT_MODEL_VERSION')
	SchemaVersionV5     AvroSchemaVersion = 5
	SchemaVersionV4     AvroSchemaVersion = 4
)

// Full list of schema versions (ensure this is updated when adding a new schema version)
func getAllSchemaVersions() []AvroSchemaVersion {
	return []AvroSchemaVersion{SchemaVersionV5, SchemaVersionV4}
}

var avroVersionedSchemas map[string]avro.Schema

func loadAvroSchemaCache(version AvroSchemaVersion) (*avro.SchemaCache, int) {
	numLoaded := 0
	cache := avro.SchemaCache{}
	root := fmt.Sprintf("schemas/v%d", version)
	dir, err := rawSchemas.ReadDir(root)
	if err != nil {
		panic(fmt.Errorf("avro file read %s: %w", root, err))
	}
	for _, direntry := range dir {
		if direntry.IsDir() {
			// folders aren't files
			continue
		}
		if !strings.Contains(direntry.Name(), ".json") {
			// only load schemas
			continue
		}
		filepath := path.Join(root, direntry.Name())

		data, err := rawSchemas.ReadFile(filepath)
		if err != nil {
			panic(fmt.Errorf("avro file read %s: %w", filepath, err))
		}
		_, err = avro.ParseBytesWithCache(data, fmt.Sprintf("v%d", version), &cache)
		if err != nil {
			panic(fmt.Errorf("avro parse %s: %w", filepath, err))
		}
		numLoaded += 1
	}
	return &cache, numLoaded
}

// Get the latest schema, used for converting model data into avro data.
func GetAvroSchemaVersion(schemaType AvroSchemaType, version uint32) (avro.Schema, error) {
	schemaRef := fmt.Sprintf("v%d.%s", version, schemaType)

	schema, ok := avroVersionedSchemas[schemaRef]
	if !ok {
		return schema, fmt.Errorf("unable to load the avro schema %s", schemaRef)
	}
	return schema, nil
}

// Get the avro schema appropriate for the provided avro content.
// The avro version is determined by reading the first integer in the avro content which should be the model_version.
func GetAvroSchema(schemaType AvroSchemaType, avroRawBytes []byte) (avro.Schema, error) {
	// Read the first int to determine the model_version
	avroReader := avro.NewReader(bytes.NewReader(avroRawBytes), 20)
	version := avroReader.ReadInt()

	// Load the desired schema reference.
	schemaRef := fmt.Sprintf("v%d.%s", version, schemaType)
	schema, ok := avroVersionedSchemas[schemaRef]

	if !ok {
		// If version can't be determined it must be the oldest
		schemaRef := fmt.Sprintf("v%d.%s", SchemaVersionV4, schemaType)
		schema, ok := avroVersionedSchemas[schemaRef]
		if ok {
			return schema, nil
		}
	}
	if !ok {
		return schema, fmt.Errorf("unable to load the avro schema %s", schemaRef)
	}
	return schema, nil
}

func loadSchemaSet(currentVersion AvroSchemaVersion) int {
	schemas, numLoadedForVersion := loadAvroSchemaCache(currentVersion)
	var schemaRef string
	for _, schemaType := range getAllSchemaTypes() {
		schemaRef = fmt.Sprintf("v%d.%s", currentVersion, schemaType)
		avroVersionedSchemas[schemaRef] = schemas.Get(schemaRef)
	}
	return numLoadedForVersion
}

// Make compatibility schemas so all the old schemas load with the new defaults.
func makeSchemasCompatible(oldVersion AvroSchemaVersion) error {
	for _, schemaType := range getAllSchemaTypes() {
		// Locate the old schema
		schemaRefOld := fmt.Sprintf("v%d.%s", oldVersion, schemaType)
		schemaOld, ok := avroVersionedSchemas[schemaRefOld]
		if !ok {
			return fmt.Errorf("unable to find the schema %s", schemaRefOld)
		}

		// Locate the new schema
		schemaRefLatest := fmt.Sprintf("v%d.%s", SchemaVersionLatest, schemaType)
		schemaLatest, ok := avroVersionedSchemas[schemaRefLatest]
		if !ok {
			return fmt.Errorf("unable to find the latest schema %s", schemaRefLatest)
		}
		// Make it so the old schema can be read and all the defaults from the new schema applied.
		compatibleSchema, err := avro.NewSchemaCompatibility().Resolve(schemaLatest, schemaOld)
		if err != nil {
			fmt.Println()
			return fmt.Errorf("unable to find compatibility between schemas %s and %s with error %+v", schemaRefLatest, schemaRefOld, err)
		}
		// Store the schemas for later use.
		avroVersionedSchemas[schemaRefOld] = compatibleSchema
	}
	return nil
}

// initialise ALL avro schemas
func init() {
	avroVersionedSchemas = map[string]avro.Schema{}
	var numLoaded int

	allVersions := getAllSchemaVersions()

	// Load all the schemas
	for _, currentVersion := range allVersions {
		numLoaded += loadSchemaSet(currentVersion)
	}

	// Remove the current version from the list of versions
	for i, other := range allVersions {
		if other == SchemaVersionLatest {
			allVersions = append(allVersions[:i], allVersions[i+1:]...)
			break
		}
	}

	// Add all the schema conversions so all loaded data will be in the newest format.
	for _, historicalVersion := range allVersions {
		err := makeSchemasCompatible(historicalVersion)
		if err != nil {
			panic(fmt.Sprintf("error when making schemas compatible, %+v", err))
		}
	}

	if numLoaded == 0 {
		panic(fmt.Errorf("loaded %d schemas", numLoaded))
	}
}

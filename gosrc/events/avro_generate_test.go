package events

// requires github.com/wirelessr/avroschema so is disabled by default to remove from go.mod
// Generate avro schemas from struct definitions.
// These are not valid and require substantial hand editing to be correct.
// Still better than creating from scratch.

// func genAvroSchema[T any](filename string) {
// 	var dest T
// 	schema, err := avroschema.Reflect(&dest)
// 	if err != nil {
// 		panic(err)
// 	}
// 	testdata.DumpBytes("bad_schemas/"+filename, []byte(schema))
// }

// func TestAvroGenerate(t *testing.T) {
// 	generateAvroSchemas := false // disabled by default
// 	if generateAvroSchemas {
// 		genAvroSchema[BinaryEvent]("binary.json")
// 		genAvroSchema[DeleteEvent]("delete.json")
// 		genAvroSchema[DownloadEvent]("download.json")
// 		genAvroSchema[GenericEvent]("generic.json")
// 		genAvroSchema[InsertEvent]("insert.json")
// 		genAvroSchema[PluginEvent]("plugin.json")
// 		genAvroSchema[StatusEvent]("status.json")
// 	}
// }

package events

import (
	"github.com/hamba/avro/v2"
)

type Model string

// Constant to use as the threshold number of bytes for erroring if no events are found in a bulk binary event.
const LENGTH_OF_BULK_HEADER_INFO = 10

const (
	ModelBinary    Model = "binary"
	ModelDelete    Model = "delete"
	ModelDownload  Model = "download"
	ModelInsert    Model = "insert"
	ModelPlugin    Model = "plugin"
	ModelStatus    Model = "status"
	ModelRetrohunt Model = "retrohunt"
)

var modelMap = map[Model]bool{
	ModelBinary:    true,
	ModelDelete:    true,
	ModelDownload:  true,
	ModelInsert:    true,
	ModelPlugin:    true,
	ModelStatus:    true,
	ModelRetrohunt: true,
}

func (b Model) Str() string {
	return string(b)
}

// returns true if the supplied model is valid
func IsValidModel(model Model) bool {
	_, ok := modelMap[model]
	return ok
}

type EventModelInterface interface {
	GetModelVersion() uint32
	SetModelVersion(newVersion uint32)
}

func GenericToAvro[T EventModelInterface](ev T, schemaType AvroSchemaType) ([]byte, error) {
	if ev.GetModelVersion() == 0 {
		ev.SetModelVersion(uint32(SchemaVersionLatest))
	}
	schema, err := GetAvroSchemaVersion(schemaType, ev.GetModelVersion())
	if err != nil {
		return []byte{}, err
	}
	return avro.Marshal(schema, &ev)
}

func GenericFromAvro[T EventModelInterface](ev T, data []byte, schemaType AvroSchemaType) error {
	schema, err := GetAvroSchema(schemaType, data)
	if err != nil {
		return err
	}
	err = avro.Unmarshal(schema, data, &ev)
	if err == nil {
		// Ensure the model is promoted upon unmarshalling.
		ev.SetModelVersion(uint32(SchemaVersionLatest))
	}
	return err
}

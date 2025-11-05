package events

import (
	"testing"

	testdata "github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/testdata"
	goccyjson "github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
)

var jsonLarge []byte = testdata.GetBytes("events/speed/status.json")
var avroLarge []byte

func init() {
	// generate avro doc equivalent to status json
	var event StatusEvent
	err := goccyjson.Unmarshal(jsonLarge, &event)
	if err != nil {
		panic(err)
	}
	avroLarge, err = event.ToAvro()
	if err != nil {
		panic(err)
	}
	// dump to disk for comparison
	// testdata.DumpBytes("events/speed/status.avro", avroLarge)
}

func BenchmarkJsonGoccy(b *testing.B) {
	var event StatusEvent
	for n := 0; n < b.N; n++ {
		err := goccyjson.Unmarshal(jsonLarge, &event)
		require.Nil(b, err)
		if event.ModelVersion == 0 {
			panic("no ModelVersion")
		}
		raw, err := goccyjson.Marshal(&event)
		require.Nil(b, err)
		if len(raw) < 10 {
			panic("length of compiled is small")
		}
	}
}

func BenchmarkJsonAvro(b *testing.B) {
	var event StatusEvent
	for n := 0; n < b.N; n++ {
		err := event.FromAvro(avroLarge)
		require.Nil(b, err)
		if event.ModelVersion == 0 {
			panic("no ModelVersion")
		}
		raw, err := event.ToAvro()
		require.Nil(b, err)
		if len(raw) < 10 {
			panic("length of compiled is small")
		}
	}
}

package events

import (
	"testing"

	testdata "github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/testdata"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
)

func TestAvroStatus(t *testing.T) {
	jsonOrig := testdata.GetBytes("events/speed/status.json")
	var event StatusEvent
	err := json.Unmarshal(jsonOrig, &event)
	require.Nil(t, err)

	avroStore, err := event.ToAvro()
	require.Nil(t, err)
	require.JSONEq(t, string(event.Entity.Input.Entity.Info), `{"thing":[666,666,777]}`)
	require.JSONEq(t, string(event.Entity.Results[0].Entity.Info), `{"items":[555,666,777]}`)

	var avroLoad StatusEvent
	err = avroLoad.FromAvro(avroStore)
	require.Nil(t, err)
	require.JSONEq(t, string(avroLoad.Entity.Input.Entity.Info), `{"thing": [666.0, 666.0, 777.0]}`)

	require.Nil(t, err)
	jsonFin, err := json.Marshal(&avroLoad)
	require.Nil(t, err)

	require.JSONEq(t, string(jsonOrig), string(jsonFin))
}

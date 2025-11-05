package msginflight

import (
	"testing"

	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/events"
	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/testdata"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
)

func TestFromEvent(t *testing.T) {
	var ok bool
	msg, err := NewMsgInFlightFromEvent(GenEventBinary(nil))
	require.Nil(t, err)
	_, ok = msg.GetBinary()
	require.True(t, ok)
	_, ok = msg.GetStatus()
	require.False(t, ok)
	// require.Equal(t, msg.Base.Model, events.ModelBinary)

	msg, err = NewMsgInFlightFromEvent(GenEventDelete(""))
	require.Nil(t, err)
	_, ok = msg.GetDelete()
	require.True(t, ok)
	_, ok = msg.GetStatus()
	require.False(t, ok)
	// require.Equal(t, msg.Base.Model, events.ModelDelete)

	msg, err = NewMsgInFlightFromEvent(GenEventInsert(""))
	require.Nil(t, err)
	_, ok = msg.GetInsert()
	require.True(t, ok)
	_, ok = msg.GetStatus()
	require.False(t, ok)
	// require.Equal(t, msg.Base.Model, events.ModelInsert)

	msg, err = NewMsgInFlightFromEvent(GenEventPlugin(""))
	require.Nil(t, err)
	_, ok = msg.GetPlugin()
	require.True(t, ok)
	_, ok = msg.GetStatus()
	require.False(t, ok)
	// require.Equal(t, msg.Base.Model, events.ModelPlugin)

	msg, err = NewMsgInFlightFromEvent(GenEventStatus(""))
	require.Nil(t, err)
	_, ok = msg.GetStatus()
	require.True(t, ok)
	_, ok = msg.GetBinary()
	require.False(t, ok)
	// require.Equal(t, msg.Base.Model, events.ModelStatus)
}

// do common processing with bulk events
func conversionTests(t *testing.T, msgs []*MsgInFlight, model events.Model) {
	// bulk
	notModel := events.ModelDelete
	if model == events.ModelDelete {
		notModel = events.ModelBinary
	}
	bulk, err := MsgInFlightsToAvroBulk(msgs, model)
	require.Nil(t, err)
	require.Greater(t, len(bulk), 50)

	require.NotNil(t, msgs[0])

	msgs2, err := AvroBulkToMsgInFlights(bulk, model)
	require.Nil(t, err)
	require.Equal(t, len(msgs2), 3)

	require.NotNil(t, msgs[0])

	_, err = MsgInFlightsToAvroBulk(msgs, notModel)
	require.NotNil(t, err)

	// avro
	ev := msgs[0]

	asavro, err := ev.Event.ToAvro()
	require.Nil(t, err)
	require.Greater(t, len(asavro), 50)

	err = ev.Event.FromAvro(asavro)
	require.Nil(t, err)

	// json
	asjson, err := json.Marshal(ev.Event)
	require.Nil(t, err)
	require.Greater(t, len(asjson), 50)

	loaded, err := NewMsgInFlightFromJson(asjson, model)
	require.Nil(t, err)

	// equal to original
	testdata.MarshalEqual(t, msgs[0].Event, loaded.Event)
}

func TestAvroBulk(t *testing.T) {
	msg, err := NewMsgInFlightFromEvent(GenEventStatus(""))
	require.Nil(t, err)
	conversionTests(t, []*MsgInFlight{msg, msg, msg}, events.ModelStatus)

	msg, err = NewMsgInFlightFromEvent(GenEventBinary(nil))
	require.Nil(t, err)
	conversionTests(t, []*MsgInFlight{msg, msg, msg}, events.ModelBinary)

	msg, err = NewMsgInFlightFromEvent(GenEventPlugin(""))
	require.Nil(t, err)
	conversionTests(t, []*MsgInFlight{msg, msg, msg}, events.ModelPlugin)

	msg, err = NewMsgInFlightFromEvent(GenEventInsert(""))
	require.Nil(t, err)
	conversionTests(t, []*MsgInFlight{msg, msg, msg}, events.ModelInsert)

	msg, err = NewMsgInFlightFromEvent(GenEventDelete(""))
	require.Nil(t, err)
	conversionTests(t, []*MsgInFlight{msg, msg, msg}, events.ModelDelete)
}

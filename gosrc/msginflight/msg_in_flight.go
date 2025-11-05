package msginflight

import (
	"fmt"
	"reflect"

	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/events"
	"github.com/goccy/go-json"
)

// MsgInFlight holds an event and common properties.
// Properties are pointers to ensure that changes are reflected in the actual event
// and that only one copy of any data is retained.
type MsgInFlight struct {
	FromClient bool                  // this message was sourced from a client
	Event      events.EventInterface // pointer to any event struct
	Base       *events.BaseEvent     // common fields to all events
	status     *events.StatusEvent
	binary     *events.BinaryEvent
	plugin     *events.PluginEvent
	insert     *events.InsertEvent
	delete     *events.DeleteEvent
	download   *events.DownloadEvent
	retrohunt  *events.RetrohuntEvent
}

// NewMsgInFlightFromEvent takes any kind of model.Event and returns a MsgInFlight from it.
func NewMsgInFlightFromEvent(event events.EventInterface) (*MsgInFlight, error) {
	msg := MsgInFlight{
		Event: event,
		Base:  event.GetBase(),
	}
	switch v := event.(type) {
	case *events.StatusEvent:
		msg.status = v
	case *events.BinaryEvent:
		msg.binary = v
	case *events.PluginEvent:
		msg.plugin = v
	case *events.InsertEvent:
		msg.insert = v
	case *events.DeleteEvent:
		msg.delete = v
	case *events.DownloadEvent:
		msg.download = v
	case *events.RetrohuntEvent:
		msg.retrohunt = v
	default:
		return nil, fmt.Errorf("unknown event to turn into msg in flight (forgot pointer?): %s", reflect.TypeOf(event))
	}
	err := msg.Event.CheckValid()
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func (mp *MsgInFlight) GetModel() events.Model {
	return mp.Base.Model
}

func (mp *MsgInFlight) GetStatus() (*events.StatusEvent, bool) {
	return mp.status, mp.status != nil
}

func (mp *MsgInFlight) GetBinary() (*events.BinaryEvent, bool) {
	return mp.binary, mp.binary != nil
}

func (mp *MsgInFlight) GetPlugin() (*events.PluginEvent, bool) {
	return mp.plugin, mp.plugin != nil
}

func (mp *MsgInFlight) GetInsert() (*events.InsertEvent, bool) {
	return mp.insert, mp.insert != nil
}

func (mp *MsgInFlight) GetDelete() (*events.DeleteEvent, bool) {
	return mp.delete, mp.delete != nil
}

func (mp *MsgInFlight) GetDownload() (*events.DownloadEvent, bool) {
	return mp.download, mp.download != nil
}

func (mp *MsgInFlight) GetRetrohunt() (*events.RetrohuntEvent, bool) {
	return mp.retrohunt, mp.retrohunt != nil
}

func (mp *MsgInFlight) MarshalJSON() ([]byte, error) {
	return json.Marshal(mp.Event)
}

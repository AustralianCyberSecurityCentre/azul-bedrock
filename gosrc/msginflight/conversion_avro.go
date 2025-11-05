package msginflight

import (
	"errors"
	"fmt"

	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/events"
)

// NewMsgInFlightFromAvro converts event bytes into MsgInFlight.
func NewMsgInFlightFromAvro(event []byte, model events.Model) (*MsgInFlight, error) {
	if len(model) == 0 {
		return nil, errors.New("no model provided to decode avro event")
	}
	var err error
	msg := MsgInFlight{}
	// golang is not great for processing similar data types.
	switch model {
	case events.ModelStatus:
		msg.status = &events.StatusEvent{}
		err = msg.status.FromAvro(event)
		if err != nil {
			break
		}
		msg.Event = msg.status
	case events.ModelBinary:
		msg.binary = &events.BinaryEvent{}
		err = msg.binary.FromAvro(event)
		if err != nil {
			break
		}
		msg.Event = msg.binary
	case events.ModelPlugin:
		msg.plugin = &events.PluginEvent{}
		err = msg.plugin.FromAvro(event)
		if err != nil {
			break
		}
		msg.Event = msg.plugin
	case events.ModelInsert:
		msg.insert = &events.InsertEvent{}
		err = msg.insert.FromAvro(event)
		if err != nil {
			break
		}
		msg.Event = msg.insert
	case events.ModelDelete:
		msg.delete = &events.DeleteEvent{}
		err = msg.delete.FromAvro(event)
		if err != nil {
			break
		}
		msg.Event = msg.delete
	case events.ModelDownload:
		msg.download = &events.DownloadEvent{}
		err = msg.download.FromAvro(event)
		if err != nil {
			break
		}
		msg.Event = msg.download
	case events.ModelRetrohunt:
		msg.retrohunt = &events.RetrohuntEvent{}
		err = msg.retrohunt.FromAvro(event)
		if err != nil {
			break
		}
		msg.Event = msg.retrohunt
	default:
		return nil, fmt.Errorf("failed to parse avro event with invalid model %s: %w", model, err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to parse avro event with len(%d): %w", len(event), err)
	}
	msg.Base = msg.Event.GetBase()
	return &msg, err
}

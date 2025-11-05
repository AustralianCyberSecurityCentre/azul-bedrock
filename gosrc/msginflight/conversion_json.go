package msginflight

import (
	"fmt"

	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/events"
	"github.com/goccy/go-json"
)

// NewMsgInFlightFromJson uses model name to decode message bytes into correct MsgInFlight properties.
func NewMsgInFlightFromJson(message []byte, model events.Model) (*MsgInFlight, error) {
	var err error
	msg := MsgInFlight{}
	// golang is not great for processing similar data types.
	switch model {
	case events.ModelStatus:
		msg.status = &events.StatusEvent{}
		err = json.Unmarshal(message, msg.status)
		if err != nil {
			break
		}
		msg.Event = msg.status
	case events.ModelBinary:
		msg.binary = &events.BinaryEvent{}
		err = json.Unmarshal(message, msg.binary)
		if err != nil {
			break
		}
		msg.Event = msg.binary
	case events.ModelPlugin:
		msg.plugin = &events.PluginEvent{}
		err = json.Unmarshal(message, msg.plugin)
		if err != nil {
			break
		}
		msg.Event = msg.plugin
	case events.ModelInsert:
		msg.insert = &events.InsertEvent{}
		err = json.Unmarshal(message, msg.insert)
		if err != nil {
			break
		}
		msg.Event = msg.insert
	case events.ModelDelete:
		msg.delete = &events.DeleteEvent{}
		err = json.Unmarshal(message, msg.delete)
		if err != nil {
			break
		}
		msg.Event = msg.delete
	case events.ModelDownload:
		msg.download = &events.DownloadEvent{}
		err = json.Unmarshal(message, msg.download)
		if err != nil {
			break
		}
		msg.Event = msg.download
	case events.ModelRetrohunt:
		msg.retrohunt = &events.RetrohuntEvent{}
		err = json.Unmarshal(message, msg.retrohunt)
		if err != nil {
			break
		}
		msg.Event = msg.retrohunt
	default:
		return nil, fmt.Errorf("failed to parse json event with invalid model %s: %w", model, err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to parse json event with len(%d): %w", len(message), err)
	}
	msg.Base = msg.Event.GetBase()
	return &msg, err
}

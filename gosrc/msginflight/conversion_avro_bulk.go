package msginflight

import (
	"errors"
	"fmt"

	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/events"
)

// MsgInFlightsToAvroBulk converts a list of msgs into a bulk event message.
func MsgInFlightsToAvroBulk(msgs []*MsgInFlight, model events.Model) ([]byte, error) {
	if len(model) == 0 {
		return nil, errors.New("no model provided to encode avro events")
	}
	var err error
	var rawBulk []byte
	// golang is not great for processing similar data types.
	switch model {
	case events.ModelStatus:
		var bulk = events.BulkStatusEvent{Events: []*events.StatusEvent{}}
		for _, msg := range msgs {
			if msg.status != nil {
				bulk.ModelVersion = msg.status.ModelVersion
			}
			bulk.Events = append(bulk.Events, msg.status)
		}
		rawBulk, err = bulk.ToAvro()
		if err != nil {
			break
		}
	case events.ModelBinary:
		var bulk = events.BulkBinaryEvent{Events: []*events.BinaryEvent{}}
		for _, msg := range msgs {
			if msg.binary != nil {
				bulk.ModelVersion = msg.binary.ModelVersion
			}
			bulk.Events = append(bulk.Events, msg.binary)
		}
		rawBulk, err = bulk.ToAvro()
		if err != nil {
			break
		}
	case events.ModelPlugin:
		var bulk = events.BulkPluginEvent{Events: []*events.PluginEvent{}}
		for _, msg := range msgs {
			if msg.plugin != nil {
				bulk.ModelVersion = msg.plugin.ModelVersion
			}
			bulk.Events = append(bulk.Events, msg.plugin)
		}
		rawBulk, err = bulk.ToAvro()
		if err != nil {
			break
		}
	case events.ModelInsert:
		var bulk = events.BulkInsertEvent{Events: []*events.InsertEvent{}}
		for _, msg := range msgs {
			if msg.insert != nil {
				bulk.ModelVersion = msg.insert.ModelVersion
			}
			bulk.Events = append(bulk.Events, msg.insert)
		}
		rawBulk, err = bulk.ToAvro()
		if err != nil {
			break
		}
	case events.ModelDelete:
		var bulk = events.BulkDeleteEvent{Events: []*events.DeleteEvent{}}
		for _, msg := range msgs {
			if msg.delete != nil {
				bulk.ModelVersion = msg.delete.ModelVersion
			}
			bulk.Events = append(bulk.Events, msg.delete)
		}
		rawBulk, err = bulk.ToAvro()
		if err != nil {
			break
		}
	case events.ModelDownload:
		var bulk = events.BulkDownloadEvent{Events: []*events.DownloadEvent{}}
		for _, msg := range msgs {
			if msg.download != nil {
				bulk.ModelVersion = msg.download.ModelVersion
			}
			bulk.Events = append(bulk.Events, msg.download)
		}
		rawBulk, err = bulk.ToAvro()
		if err != nil {
			break
		}
	case events.ModelRetrohunt:
		var bulk = events.BulkRetrohuntEvent{Events: []*events.RetrohuntEvent{}}
		for _, msg := range msgs {
			if msg.retrohunt != nil {
				bulk.ModelVersion = msg.retrohunt.ModelVersion
			}
			bulk.Events = append(bulk.Events, msg.retrohunt)
		}
		rawBulk, err = bulk.ToAvro()
		if err != nil {
			break
		}
	default:
		return nil, fmt.Errorf("failed to store messages with model %s: %w", model, err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to store messages len(%d): %w", len(msgs), err)
	}
	return rawBulk, err
}

// AvroBulkToMsgInFlights converts bulk avro message into msginflights
func AvroBulkToMsgInFlights(rawEvents []byte, model events.Model) ([]*MsgInFlight, error) {
	if len(model) == 0 {
		return nil, errors.New("no model provided to decode avro event")
	}
	var err error
	var msgs = []*MsgInFlight{}
	// golang is not great for processing similar data types.
	switch model {
	case events.ModelStatus:
		var bulk events.BulkStatusEvent
		err = bulk.FromAvro(rawEvents)
		if err != nil {
			break
		}
		// convert all into msginflight
		for _, ev := range bulk.Events {
			msg, err := NewMsgInFlightFromEvent(ev)
			if err != nil {
				break
			}
			msgs = append(msgs, msg)
		}
	case events.ModelBinary:
		var bulk events.BulkBinaryEvent
		err = bulk.FromAvro(rawEvents)
		if err != nil {
			break
		}
		// convert all into msginflight
		for _, ev := range bulk.Events {
			msg, err := NewMsgInFlightFromEvent(ev)
			if err != nil {
				break
			}
			msgs = append(msgs, msg)
		}
	case events.ModelPlugin:
		var bulk events.BulkPluginEvent
		err = bulk.FromAvro(rawEvents)
		if err != nil {
			break
		}
		// convert all into msginflight
		for _, ev := range bulk.Events {
			msg, err := NewMsgInFlightFromEvent(ev)
			if err != nil {
				break
			}
			msgs = append(msgs, msg)
		}
	case events.ModelInsert:
		var bulk events.BulkInsertEvent
		err = bulk.FromAvro(rawEvents)
		if err != nil {
			break
		}
		// convert all into msginflight
		for _, ev := range bulk.Events {
			msg, err := NewMsgInFlightFromEvent(ev)
			if err != nil {
				break
			}
			msgs = append(msgs, msg)
		}
	case events.ModelDelete:
		var bulk events.BulkDeleteEvent
		err = bulk.FromAvro(rawEvents)
		if err != nil {
			break
		}
		// convert all into msginflight
		for _, ev := range bulk.Events {
			msg, err := NewMsgInFlightFromEvent(ev)
			if err != nil {
				break
			}
			msgs = append(msgs, msg)
		}
	case events.ModelDownload:
		var bulk events.BulkDownloadEvent
		err = bulk.FromAvro(rawEvents)
		if err != nil {
			break
		}
		// convert all into msginflight
		for _, ev := range bulk.Events {
			msg, err := NewMsgInFlightFromEvent(ev)
			if err != nil {
				break
			}
			msgs = append(msgs, msg)
		}
	case events.ModelRetrohunt:
		var bulk events.BulkRetrohuntEvent
		err = bulk.FromAvro(rawEvents)
		if err != nil {
			break
		}
		// convert all into msginflight
		for _, ev := range bulk.Events {
			msg, err := NewMsgInFlightFromEvent(ev)
			if err != nil {
				break
			}
			msgs = append(msgs, msg)
		}
	default:
		return nil, fmt.Errorf("failed to parse avro bulk event with model %s: %w", model, err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to parse avro bulk event with len(%d): %w", len(rawEvents), err)
	}
	for i := range msgs {
		msgs[i].FromClient = true
	}
	return msgs, err
}

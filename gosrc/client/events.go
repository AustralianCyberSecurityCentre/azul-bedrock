/*
Package client handles registration and event handling for different authors.
*/
package client

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/goccy/go-json"

	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/client/getevents"
	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/client/postevents"
	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/events"
	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/models"
)

type FetchEventsStruct struct {
	Model                   events.Model
	AvroFormat              bool // events should be avro formatted
	Count                   int
	Deadline                int
	IsTask                  bool
	RequireExpedite         bool
	RequireLive             bool
	RequireHistoric         bool
	RequireContent          bool                  // only model=binary
	RequireSources          []string              // only model=binary,status
	RequireUnderContentSize int                   // only model=binary
	RequireOverContentSize  int                   // only model=binary
	RequireActions          []events.BinaryAction // only model=binary,status
	RequireStreams          []string              // only model=binary
	DenyActions             []events.BinaryAction // only model=binary,status
	DenySelf                bool                  // only model=binary,status
}

// Pulls a list of raw messages and related info or an error.
func (c *Client) GetEventsBytes(query *FetchEventsStruct) ([]byte, *models.EventResponseInfo, error) {
	u, err := url.Parse(c.eventsUrl + "/api/v2/event/" + string(query.Model) + "/" + getevents.EndpointPassive)
	if query.IsTask {
		u, err = url.Parse(c.eventsUrl + "/api/v2/event/" + string(query.Model) + "/" + getevents.EndpointActive)
	}
	if err != nil {
		return nil, nil, err
	}

	if !events.IsValidModel(query.Model) {
		return nil, nil, fmt.Errorf("model %s is not a registered model", query.Model)
	}

	if query.Model != events.ModelBinary && len(query.RequireActions) > 0 {
		return nil, nil, fmt.Errorf("can't set RequireActions when RequireModel is not set to ModelBinary")
	}

	q := u.Query()
	q.Set(getevents.Name, c.Author.Name)
	q.Set(getevents.Version, c.Author.Version)
	q.Set(getevents.DeploymentKey, c.deploymentKey)

	q.Set(getevents.Deadline, fmt.Sprint(query.Deadline))
	q.Set(getevents.Count, fmt.Sprint(query.Count))
	q.Set(getevents.AvroFormat, strconv.FormatBool(query.AvroFormat))

	// custom filters
	if query.RequireExpedite {
		q.Set(getevents.RequireExpedite, strconv.FormatBool(query.RequireExpedite))
	}
	if query.RequireLive {
		q.Set(getevents.RequireLive, strconv.FormatBool(query.RequireLive))
	}
	if query.RequireHistoric {
		q.Set(getevents.RequireHistoric, strconv.FormatBool(query.RequireHistoric))
	}
	if query.RequireContent {
		q.Set(getevents.RequireContent, strconv.FormatBool(query.RequireContent))
	}
	for _, src := range query.RequireSources {
		q.Add(getevents.RequireSources, src)
	}
	if query.RequireUnderContentSize > 0 {
		q.Set(getevents.RequireUnderContentSize, strconv.FormatInt(int64(query.RequireUnderContentSize), 10))
	}
	if query.RequireOverContentSize > 0 {
		q.Set(getevents.RequireOverContentSize, strconv.FormatInt(int64(query.RequireOverContentSize), 10))
	}
	actions, err := events.StringsFromActions(query.RequireActions)
	if err != nil {
		return nil, nil, err
	}
	q[getevents.RequireActions] = actions
	actions, err = events.StringsFromActions(query.DenyActions)
	if err != nil {
		return nil, nil, err
	}
	q[getevents.DenyActions] = actions
	if query.DenySelf {
		q.Set(getevents.DenySelf, strconv.FormatBool(query.DenySelf))
	}
	q[getevents.RequireStreams] = query.RequireStreams

	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		// error during connection to server
		return nil, nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		// error during connection to server
		return nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		// bad return code
		return nil, nil, statusError(resp)
	}

	mediatype, params, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil {
		return nil, nil, fmt.Errorf("%w in %s", err, resp.Header.Get("Content-Type"))
	}
	if mediatype != "multipart/form-data" {
		return nil, nil, fmt.Errorf("media type is not form data, is %s", mediatype)
	}

	var info models.EventResponseInfo
	var data []byte
	mpread := multipart.NewReader(resp.Body, params["boundary"])
	for {
		part, err_part := mpread.NextPart()
		if err_part == io.EOF {
			break
		}
		raw, err := io.ReadAll(part)
		if err != nil {
			return nil, nil, err
		}
		if part.FormName() == getevents.RespInfo {
			err = json.Unmarshal([]byte(raw), &info)
			if err != nil {
				return nil, nil, err
			}
		} else if part.FormName() == getevents.RespEvents {
			data = raw
		} else {
			return nil, nil, fmt.Errorf("unknown form part %s", part.FormName())
		}
	}
	return data, &info, nil
}

// Converts received avro events into a specific event type
// Its not possible to have a generic within a struct function so this is separate on purpose
func GetEvents[T events.BulkEventInterface](c *Client, query *FetchEventsStruct, data T) (T, *models.EventResponseInfo, error) {
	query.AvroFormat = true
	raw, info, err := c.GetEventsBytes(query)
	if err != nil {
		return data, nil, fmt.Errorf("failed to get events: %w", err)
	}
	err = data.FromAvro(raw)
	if err != nil {
		return data, nil, fmt.Errorf("failed to avro decode events: %w", err)
	}
	return data, info, err
}

// Pulls a list of binary events and related info or an error.
func (c *Client) GetBinaryEvents(query *FetchEventsStruct) (*events.BulkBinaryEvent, *models.EventResponseInfo, error) {
	query.Model = events.ModelBinary
	data := events.BulkBinaryEvent{}
	return GetEvents(c, query, &data)
}

// Pulls a list of status events and related info or an error.
func (c *Client) GetStatusEvents(query *FetchEventsStruct) (*events.BulkStatusEvent, *models.EventResponseInfo, error) {
	query.Model = events.ModelStatus
	data := events.BulkStatusEvent{}
	return GetEvents(c, query, &data)
}

// Pulls a list of plugin events and related info or an error.
func (c *Client) GetPluginEvents(query *FetchEventsStruct) (*events.BulkPluginEvent, *models.EventResponseInfo, error) {
	query.Model = events.ModelPlugin
	data := events.BulkPluginEvent{}
	return GetEvents(c, query, &data)
}

// Pulls a list of insert events and related info or an error.
func (c *Client) GetInsertEvents(query *FetchEventsStruct) (*events.BulkInsertEvent, *models.EventResponseInfo, error) {
	query.Model = events.ModelInsert
	data := events.BulkInsertEvent{}
	return GetEvents(c, query, &data)
}

// Pulls a list of delete events and related info or an error.
func (c *Client) GetDeleteEvents(query *FetchEventsStruct) (*events.BulkDeleteEvent, *models.EventResponseInfo, error) {
	query.Model = events.ModelDelete
	data := events.BulkDeleteEvent{}
	return GetEvents(c, query, &data)
}

// Pulls a list of download events and related info or an error.
func (c *Client) GetDownloadEvents(query *FetchEventsStruct) (*events.BulkDownloadEvent, *models.EventResponseInfo, error) {
	query.Model = events.ModelDownload
	data := events.BulkDownloadEvent{}
	return GetEvents(c, query, &data)
}

// Pulls a list of retrohunt events and related info or an error.
func (c *Client) GetRetrohuntEvents(query *FetchEventsStruct) (*events.BulkRetrohuntEvent, *models.EventResponseInfo, error) {
	query.Model = events.ModelRetrohunt
	data := events.BulkRetrohuntEvent{}
	return GetEvents(c, query, &data)
}

type PublishBytesOptions struct {
	Model        events.Model
	AvroFormat   bool // events should be avro formatted
	Sync         bool
	IncludeOk    bool
	PausePlugins bool
}

type PublishEventsOptions = PublishBytesOptions

// Send the requested raw messages to the system.
// Events must be avro bulk events format
func (c *Client) PostEventsBytes(evs []byte, opts *PublishBytesOptions) (*models.ResponsePostEvent, error) {
	if len(opts.Model) == 0 {
		return nil, fmt.Errorf("must supply model information")
	}
	if !events.IsValidModel(opts.Model) {
		return nil, fmt.Errorf("model %s is not a registered model", opts.Model)
	}

	u, err := url.Parse(c.eventsUrl + "/api/v2/event")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set(postevents.Model, opts.Model.Str())
	q.Set(postevents.AvroFormat, strconv.FormatBool(opts.AvroFormat))
	q.Set(postevents.Sync, strconv.FormatBool(opts.Sync))
	q.Set(postevents.IncludeOk, strconv.FormatBool(opts.IncludeOk))
	q.Set(postevents.PausePlugins, strconv.FormatBool(opts.PausePlugins))
	u.RawQuery = q.Encode()
	req, err := http.NewRequest("POST", u.String(), bytes.NewReader(evs))
	if err != nil {
		return nil, err
	}
	if opts.AvroFormat {
		// not technically a valid content type, but able to use it anyway.
		req.Header.Set("Content-Type", "application/avro")
	} else {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, statusError(resp)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response models.ResponsePostEvent
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w in '%s'", err, string(body))
	}

	return &response, nil
}

// Posts the requested events to dispatcher.
func (c *Client) PostEvents(events events.BulkEventInterface, opts *PublishEventsOptions) (*models.ResponsePostEvent, error) {
	opts.Model = events.GetModel()
	opts.AvroFormat = true
	o, err := events.ToAvro()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal: %w", err)
	}
	resp, err := c.PostEventsBytes(o, opts)
	return resp, err
}

// Registers the plugin with dispatcher.
func (c *Client) PublishPlugin() error {
	now := time.Now()
	ob := events.PluginEvent{
		ModelVersion: events.CurrentModelVersion,
		Author:       c.summary,
		Timestamp:    now,
		Entity:       c.Author,
	}
	bulk := events.BulkPluginEvent{Events: []*events.PluginEvent{&ob}}
	_, err := c.PostEvents(&bulk, &PublishEventsOptions{Model: events.ModelPlugin})
	return err
}

// Returns information about which consumers would process the provided event
func (c *Client) SimulateConsumersOnEventBytes(msg []byte) (*models.EventSimulate, error) {
	u, err := url.Parse(c.eventsUrl + "/api/v2/event/simulate")
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", u.String(), bytes.NewReader(msg))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, statusError(resp)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	ret := models.EventSimulate{}
	err = json.Unmarshal(raw, &ret)
	if err != nil {
		return nil, err
	}

	return &ret, nil
}

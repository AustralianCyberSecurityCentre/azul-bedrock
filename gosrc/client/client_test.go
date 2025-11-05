package client

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/events"
	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/models"
	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/testdata"
	"github.com/goccy/go-json"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var server *httptest.Server
var respCode int = 200
var respBody []byte = []byte{}
var respBodyInfo []byte = []byte{}
var respBodyEvents []byte = []byte{}

// last response body
var lastReqBody []byte = []byte{}
var lastReqURL string = ""

var c ClientInterface

func responser(rw http.ResponseWriter, req *http.Request) {
	lastReqURL = req.URL.String()
	var err error
	lastReqBody, err = io.ReadAll(req.Body)
	if err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte("bad body"))
		return
	}
	if respCode == 200 && req.URL.Path == "/api/v2/event/binary/passive" {
		var err error
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		var fw io.Writer
		// write info
		fw, err = w.CreateFormFile("info", "info.json")
		if err != nil {
			return
		}
		_, err = fw.Write(respBodyInfo)
		if err != nil {
			return
		}
		// write data
		fw, err = w.CreateFormFile("events", "events.json")
		if err != nil {
			return
		}
		_, err = fw.Write(respBodyEvents)
		if err != nil {
			return
		}
		w.Close()
		rw.Header().Set("Content-Type", w.FormDataContentType())
		respBody, err = io.ReadAll(&b)
		if err != nil {
			return
		}
	}
	rw.WriteHeader(respCode)
	rw.Write(respBody)
	respCode = 501
}

func TestMain(m *testing.M) {
	// Start a local HTTP server
	server = httptest.NewServer(http.HandlerFunc(responser))

	c = NewClient(server.URL, server.URL, events.PluginEntity{
		Name: "MyPlugin", Version: "1",
	}, "my-plugin-key")
	ret := m.Run()
	// Close the server when test finishes
	server.Close()
	os.Exit(ret)
}

// TestMock ensures that the mocked client stays up to date with the ClientInterface.
// See readme for how to automatically update the mock_ClientInterface.go or run `mockery`
func TestMock(t *testing.T) {
	var ci ClientInterface
	mock2 := NewMockClientInterface(t)
	ci = mock2
	mock2.EXPECT().Exists(mock.Anything, mock.Anything, mock.Anything).Return(true, nil)

	exists, err := ci.Exists("mysource", "mylabel", "myhash")
	require.Nil(t, err)
	require.Equal(t, exists, true)
}

func TestPluginStart(t *testing.T) {
	respCode = 200
	respBody = []byte(`{"total_ok": 1, "total_failures":0}`)
	err := c.PublishPlugin()
	require.Nil(t, err)
	require.Equal(t, lastReqURL, "/api/v2/event?avro-format=true&include_ok=false&model=plugin&pause_plugins=false&sync=false")

	bulk := events.BulkPluginEvent{}
	err = bulk.FromAvro(lastReqBody)
	require.Nil(t, err)
	testdata.MarshalEqual(t, &bulk.Events[0].Entity, &events.PluginEntity{Name: "MyPlugin", Version: "1"})
}

func TestPostEventsBytes(t *testing.T) {
	respCode = 200
	respBody = []byte(`{"total_ok": 1, "total_failures":0}`)
	_, err := c.PostEventsBytes([]byte{}, &PublishBytesOptions{Model: "scoob"})
	require.ErrorContains(t, err, "model scoob is not a registered model")
}

func GetEventsBytes(t *testing.T) {
	// invalid model on generic
	_, _, err := c.GetEventsBytes(&FetchEventsStruct{
		Model:           "scoob",
		Count:           10,
		Deadline:        1,
		IsTask:          false,
		RequireSources:  []string{"mysource"},
		RequireContent:  true,
		RequireHistoric: true,
	})
	require.NotNil(t, err)
	require.ErrorContains(t, err, "model scoob is not a registered model")
}

func TestGetBinaryEvents(t *testing.T) {
	// serve 2 valid events
	respCode = 200
	respBodyInfo = testdata.GetBytes("events/api/get_events/valid.info.json")
	jsonEvents := testdata.GetBytes("events/api/get_events/valid.events.json")
	var eventsTmp = events.BulkBinaryEvent{}
	err := json.Unmarshal(jsonEvents, &eventsTmp)
	require.Nil(t, err)
	respBodyEvents, err = eventsTmp.ToAvro()
	require.Nil(t, err)
	data, info, err := c.GetBinaryEvents(&FetchEventsStruct{
		Model:                   "binary",
		Count:                   10,
		Deadline:                1,
		IsTask:                  false,
		RequireExpedite:         true,
		RequireLive:             true,
		RequireUnderContentSize: 100,
		RequireActions:          []events.BinaryAction{events.ActionEnriched},
		DenyActions:             []events.BinaryAction{events.ActionExtracted},
		DenySelf:                true,
		RequireStreams:          []string{"content,windows/exe"},
	})
	require.Nil(t, err)
	require.Equal(t, info, &models.EventResponseInfo{Filtered: 200, Fetched: 2, Ready: true, ConsumersNotReady: "", Filters: map[string]int{"blah": 7}})
	require.Equal(t, len(data.Events), 2)
	require.Equal(t, lastReqURL, "/api/v2/event/binary/passive?avro-format=true&count=10&d-action=extracted&d-self=true&deadline=1&deployment_key=my-plugin-key&name=MyPlugin&r-action=enriched&r-expedite=true&r-live=true&r-streams=content%2Cwindows%2Fexe&r-under-content-size=100&version=1")

	// use source filter
	respCode = 200
	data, info, err = c.GetBinaryEvents(&FetchEventsStruct{
		Model:           "binary",
		Count:           10,
		Deadline:        1,
		IsTask:          false,
		RequireSources:  []string{"mysource"},
		RequireContent:  true,
		RequireHistoric: true,
	})
	require.Nil(t, err)
	require.Equal(t, info, &models.EventResponseInfo{Filtered: 200, Fetched: 2, Ready: true, ConsumersNotReady: "", Filters: map[string]int{"blah": 7}})
	require.Equal(t, len(data.Events), 2)
	require.Equal(t, lastReqURL, "/api/v2/event/binary/passive?avro-format=true&count=10&deadline=1&deployment_key=my-plugin-key&name=MyPlugin&r-content=true&r-historic=true&r-source=mysource&version=1")

	// return bad client data to ensure we can get status codes and body from error (though its tedious)
	// use source filter
	respCode = 422
	respBody = []byte("you didnt set some parameter that i need or something")
	_, _, err = c.GetBinaryEvents(&FetchEventsStruct{
		Model:           "binary",
		Count:           10,
		Deadline:        1,
		IsTask:          false,
		RequireSources:  []string{"mysource"},
		RequireContent:  true,
		RequireHistoric: true,
	})
	require.NotNil(t, err)
	var httpError *HttpError
	require.True(t, errors.As(err, &httpError))
	require.Equal(t, httpError.StatusCode, 422)
	require.Equal(t, httpError.Body, "you didnt set some parameter that i need or something")
}

func TestPostBinary(t *testing.T) {
	// var resp *bufio.Reader
	var data []byte
	var err error

	// success
	respCode = 200
	respBody = []byte(`{"data": {"sha256": "test"}}`)
	data = []byte("this is a file")
	reader := bytes.NewReader(data)
	resp, err := c.PostStream("source", events.DataLabelContent, reader, &PostStreamStruct{})
	require.Nil(t, err)
	require.Equal(t, resp, &events.BinaryEntityDatastream{Sha256: "test", Label: "content"})
	require.Equal(t, lastReqURL, "/api/v3/stream/source/content")

	// success
	respCode = 200
	respBody = []byte(`{"data": {"sha256": "test"}}`)
	data = []byte("this is a file")
	reader = bytes.NewReader(data)
	resp, err = c.PostStream("source", events.DataLabelContent, reader, &PostStreamStruct{SkipIdentify: true, ExpectedSha256: "legit"})
	require.Nil(t, err)
	require.Equal(t, resp, &events.BinaryEntityDatastream{Sha256: "test", Label: "content"})
	require.Equal(t, lastReqURL, "/api/v3/stream/source/content?expected-sha256=legit&skip-identify=true")

	// bad args
	respCode = 400
	respBody = []byte(`missing the sha256`)
	data = []byte("this is a file")
	reader = bytes.NewReader(data)
	_, err = c.PostStream("source", events.DataLabelContent, reader, &PostStreamStruct{SkipIdentify: true})
	require.ErrorContains(t, err, "missing the sha256")
	require.Equal(t, lastReqURL, "/api/v3/stream/source/content?skip-identify=true")

}

func TestDownloadBinary(t *testing.T) {
	var resp *bufio.Reader
	var data []byte
	var err error

	// success
	respCode = 200
	respBody = []byte("OK")
	resp, err = c.DownloadBinary("source", events.DataLabelContent, "blah")
	require.Nil(t, err)
	data, err = io.ReadAll(resp)
	require.Nil(t, err)
	require.Equal(t, data, []byte("OK"))
	require.Equal(t, lastReqURL, "/api/v3/stream/source/content/blah")

	// failure
	respCode = 400
	respBody = []byte("SAD")
	_, err = c.DownloadBinary("source", events.DataLabelContent, "blah")
	require.Error(t, err)
	require.Equal(t, lastReqURL, "/api/v3/stream/source/content/blah")

	// failure - bad return code
	respCode = 422
	respBody = []byte("SAD")
	_, err = c.DownloadBinary("source", events.DataLabelContent, "blah")
	require.Error(t, err)
	require.Equal(t, lastReqURL, "/api/v3/stream/source/content/blah")
	var httpError *HttpError
	require.True(t, errors.As(err, &httpError))
	require.Equal(t, httpError.StatusCode, 422)
	require.Equal(t, httpError.Body, "SAD")
}

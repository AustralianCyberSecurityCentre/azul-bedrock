package client

import (
	"bufio"
	"io"
	"time"

	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/events"
	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/models"
)

// client interface necessary for mocking/testing
type ClientInterface interface {
	SetAuth(username, password string)
	// events
	PublishPlugin() error
	GetEventsBytes(query *FetchEventsStruct) ([]byte, *models.EventResponseInfo, error)
	GetBinaryEvents(query *FetchEventsStruct) (*events.BulkBinaryEvent, *models.EventResponseInfo, error)
	GetStatusEvents(query *FetchEventsStruct) (*events.BulkStatusEvent, *models.EventResponseInfo, error)
	GetPluginEvents(query *FetchEventsStruct) (*events.BulkPluginEvent, *models.EventResponseInfo, error)
	GetInsertEvents(query *FetchEventsStruct) (*events.BulkInsertEvent, *models.EventResponseInfo, error)
	GetDeleteEvents(query *FetchEventsStruct) (*events.BulkDeleteEvent, *models.EventResponseInfo, error)
	PostEventsBytes(events []byte, opts *PublishBytesOptions) (*models.ResponsePostEvent, error)
	PostEvents(events events.BulkEventInterface, opts *PublishEventsOptions) (*models.ResponsePostEvent, error)
	SimulateConsumersOnEventBytes(msg []byte) (*models.EventSimulate, error)
	// streams
	PostStream(source string, label events.DatastreamLabel, reader io.Reader, query *PostStreamStruct) (*events.BinaryEntityDatastream, error)
	PostStreamContent(source string, reader io.Reader) (*events.BinaryEntity, error)
	DownloadBinaryChunk(source string, label events.DatastreamLabel, hash string, start uint64, end uint64) ([]byte, error)
	DownloadBinary(source string, label events.DatastreamLabel, hash string) (*bufio.Reader, error)
	DeleteBinary(source string, label events.DatastreamLabel, hash string, ifOlderThan *time.Time) (bool, error)
	Exists(source string, label events.DatastreamLabel, hash string) (bool, error)
	GetMetadata(source string, label events.DatastreamLabel, hash string) (events.BinaryEntityDatastream, error)
}

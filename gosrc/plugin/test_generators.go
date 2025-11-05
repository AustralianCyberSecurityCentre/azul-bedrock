package plugin

import (
	"bufio"
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/client"
	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/events"
	"github.com/stretchr/testify/mock"
)

type DummyPlugin struct {
	executeMethod func(context.Context, *Job, *PluginInputUtils) *PluginError
}

func NewDummyPlugin(executeReplacement func(context.Context, *Job, *PluginInputUtils) *PluginError) *DummyPlugin {
	return &DummyPlugin{
		executeMethod: executeReplacement,
	}
}

func (dp *DummyPlugin) GetName() string {
	return "dummy-plugin"
}
func (dp *DummyPlugin) GetVersion() string {
	return "1.0.0"
}
func (dp *DummyPlugin) GetDescription() string {
	return "Not a real plugin."
}
func (dp *DummyPlugin) GetFeatures() []events.PluginEntityFeature {
	return []events.PluginEntityFeature{
		{Name: "dummy", Type: events.FeatureString, Description: "dummy feature for dummy plugin."},
		{Name: "dummyInt", Type: events.FeatureInteger, Description: "dummy integer feature for dummy plugin."},
		{Name: "dummyFloat", Type: events.FeatureFloat, Description: "dummy float feature for dummy plugin."},
		{Name: "dummyDate", Type: events.FeatureDatetime, Description: "dummy date feature for dummy plugin."},
	}
}

func (dp *DummyPlugin) GetDefaultSettings() *PluginSettings {
	return NewDefaultPluginSettings()
}

func (dp *DummyPlugin) Execute(context context.Context, job *Job, inputUtils *PluginInputUtils) *PluginError {
	if dp.executeMethod == nil {
		return nil
	}
	return dp.executeMethod(context, job, inputUtils)
}

func createFakeEvent(fileBytes []byte, author *events.PluginEntity) *events.BinaryEvent {
	testSha256 := calculateSha256OfBytes(fileBytes)
	testFileSize := uint64(len(fileBytes))
	timestamp := time.Time{}
	fakeEntity := events.BinaryEntity{
		Sha256: testSha256,
		Size:   testFileSize,
		Datastreams: []events.BinaryEntityDatastream{{
			Sha256: testSha256,
			Size:   testFileSize,
			Label:  events.DataLabelContent,
		},
		}}
	fakeEvent := events.BinaryEvent{
		Dequeued:     "test-fake-dequeued",
		Author:       author.Summary(),
		ModelVersion: events.CurrentModelVersion,
		Timestamp:    timestamp,
		Action:       events.ActionExtracted,
		Entity:       fakeEntity,
		Source: events.EventSource{
			Name:       "source",
			References: map[string]string{},
			Path: []events.EventSourcePathNode{
				{
					Author: events.EventAuthor{
						Name:     "TestServer",
						Version:  "1",
						Category: "plugin",
					},
					Action:    events.ActionExtracted,
					Sha256:    testSha256,
					Timestamp: timestamp,
					Size:      testFileSize,
				},
			},
			Timestamp: timestamp,
		},
	}
	return &fakeEvent
}

// Mock the bare minimum amount of dispatcher for runEvent in pluginRunner to work.
func createMiniFakeDispatcher(t *testing.T, testFileBytes []byte) client.ClientInterface {
	fakeDpClient := client.NewMockClientInterface(t)
	fakeDpClient.EXPECT().DownloadBinary(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
		func(source string, label events.DatastreamLabel, hash string) (*bufio.Reader, error) {
			return bufio.NewReader(bytes.NewReader(testFileBytes)), nil
		}).Maybe()
	fakeDpClient.EXPECT().DownloadBinaryChunk(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
		func(source string, label events.DatastreamLabel, hash string, start uint64, end uint64) ([]byte, error) {
			return testFileBytes[start : end+1], nil
		}).Maybe()
	fakeDpClient.EXPECT().PostStream(mock.Anything, mock.Anything, mock.Anything, mock.Anything).RunAndReturn(mockPostStreamResponse).Maybe()
	// Unrequired because it uses PostStream internally anyway.
	//fakeDpClient.EXPECT().PostStreamContent(mock.Anything, mock.Anything).RunAndReturn()
	return fakeDpClient
}

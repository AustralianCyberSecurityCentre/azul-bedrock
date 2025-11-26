package plugin

import (
	"context"
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/client"
	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/events"
	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/models"
	"github.com/sanity-io/litter"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHeartbeat(t *testing.T) {
	pr := NewPluginRunner(NewDummyPlugin(nil))
	fakeDpClient := client.NewMockClientInterface(t)
	fakeDpClient.EXPECT().PostEvents(mock.Anything, mock.Anything).RunAndReturn(func(bei events.BulkEventInterface, pbo *client.PublishBytesOptions) (*models.ResponsePostEvent, error) {
		require.Equal(t, bei.GetModel(), events.ModelStatus)
		return &models.ResponsePostEvent{}, nil
	})
	pr.dpClient = fakeDpClient
	fakeEvent := createFakeEvent([]byte("unused"), &pr.author)
	pr.performHeartbeat(time.Time{}, fakeEvent)
}

func TestPeriodicHeartbeat(t *testing.T) {
	pr := NewPluginRunner(NewDummyPlugin(nil))
	defer pr.cancelFunc()

	fakeDpClient := client.NewMockClientInterface(t)
	fakeDpClient.EXPECT().PostEvents(mock.Anything, mock.Anything).RunAndReturn(func(bei events.BulkEventInterface, pbo *client.PublishBytesOptions) (*models.ResponsePostEvent, error) {
		require.Equal(t, bei.GetModel(), events.ModelStatus)
		return &models.ResponsePostEvent{}, nil
	})
	pr.dpClient = fakeDpClient
	pr.config.HeartbeatIntervalSeconds = 1
	// Start heartbeat in background
	go func() { pr.startHeartbeat() }()

	fakeEvent := createFakeEvent([]byte("unused"), &pr.author)
	pr.heartBeatChannel <- fakeEvent
	// Give enough time for heartbeat to occur.
	time.Sleep(time.Duration(1100) * time.Millisecond)
}

// Verify that plugin is published at startup.
func TestStartup(t *testing.T) {
	pr := NewPluginRunner(NewDummyPlugin(nil))
	defer pr.cancelFunc()

	fakeDpClient := client.NewMockClientInterface(t)
	fakeDpClient.EXPECT().PublishPlugin().RunAndReturn(func() error { return nil })
	pr.dpClient = fakeDpClient

	err := pr.startup()
	require.Nil(t, err)
}

// Test a generic run with two results and then stop the run
func TestRun(t *testing.T) {
	pr := NewPluginRunner(NewDummyPlugin(nil))

	fakeEvent := createFakeEvent([]byte("unused"), &pr.author)

	fakeDpClient := client.NewMockClientInterface(t)
	fakeDpClient.EXPECT().PublishPlugin().RunAndReturn(func() error { return nil })
	fakeDpClient.EXPECT().PostEvents(mock.Anything, mock.Anything).RunAndReturn(
		func(bei events.BulkEventInterface, pbo *client.PublishBytesOptions) (*models.ResponsePostEvent, error) {
			return &models.ResponsePostEvent{}, nil
		},
	)
	fakeDpClient.EXPECT().GetBinaryEvents(mock.Anything).RunAndReturn(
		func(fes *client.FetchEventsStruct) (*events.BulkBinaryEvent, *models.EventResponseInfo, error) {
			return &events.BulkBinaryEvent{
					Events: []*events.BinaryEvent{fakeEvent},
				}, &models.EventResponseInfo{
					Filtered:          20,
					Fetched:           1,
					Ready:             true,
					Paused:            false,
					ConsumersNotReady: "",
					Filters:           map[string]int{},
				}, nil
		},
	)
	pr.dpClient = fakeDpClient
	// Run and exit plugin collecting the reason for the exit
	reasonChan := make(chan string)
	go func() {
		reason := pr.Run()
		reasonChan <- reason
	}()
	// Give time for the plugin to run.
	time.Sleep(500 * time.Millisecond)
	pr.cancelFunc()
	// If this exits early verify that heartbeat hasn't closed before the main plugin loop causing the plugin loop to hang
	select {
	case exitReason := <-reasonChan:
		require.Equal(t, exitReason, "Exiting runner after context was cancelled.")
	case <-time.After(5 * time.Second):
		t.Fatal("Failed to confirm plugin run exited in time!")
	}
}

// Test a generic run with two results and then stop the run
func TestRunDepthLimit(t *testing.T) {
	pr := NewPluginRunner(NewDummyPlugin(nil))

	fakeEvent := createFakeEvent([]byte("unused"), &pr.author)
	for range 10 {
		fakeEvent.Source.Path = append(fakeEvent.Source.Path, events.EventSourcePathNode{
			Author: pr.author.Summary(),
			Action: events.ActionExtracted,
		})
	}

	fakeDpClient := client.NewMockClientInterface(t)
	fakeDpClient.EXPECT().PublishPlugin().RunAndReturn(func() error { return nil })
	fakeDpClient.EXPECT().PostEvents(mock.Anything, mock.Anything).RunAndReturn(
		func(bei events.BulkEventInterface, pbo *client.PublishBytesOptions) (*models.ResponsePostEvent, error) {
			bei.GetModel()
			statusEvent := bei.(*events.BulkStatusEvent)
			optOutEvent := statusEvent.Events[0]
			require.Contains(t, optOutEvent.Entity.Message, "plugin_depth_limit")
			require.Equal(t, optOutEvent.Entity.Status, events.StatusTypeOptOut)
			return &models.ResponsePostEvent{}, nil
		},
	)
	fakeDpClient.EXPECT().GetBinaryEvents(mock.Anything).RunAndReturn(
		func(fes *client.FetchEventsStruct) (*events.BulkBinaryEvent, *models.EventResponseInfo, error) {
			return &events.BulkBinaryEvent{
					Events: []*events.BinaryEvent{fakeEvent},
				}, &models.EventResponseInfo{
					Filtered:          20,
					Fetched:           1,
					Ready:             true,
					Paused:            false,
					ConsumersNotReady: "",
					Filters:           map[string]int{},
				}, nil
		},
	)
	pr.dpClient = fakeDpClient

	go func() { pr.Run() }()
	// Give time for the plugin to run.
	time.Sleep(500 * time.Millisecond)
	pr.cancelFunc()
}

// Test run enrichment
func TestRunEnrichment(t *testing.T) {
	pr := NewPluginRunner(NewDummyPlugin(func(ctx context.Context, j *Job, inputUtils *PluginInputUtils) *PluginError {
		err := j.AddFeature("dummy", "value1")
		require.Nil(t, err)
		err = j.AddFeature("dummy", "value2")
		require.Nil(t, err)
		err = j.AddFeature("dummyInt", "10")
		require.Nil(t, err)
		return nil
	}))

	fakeEvent := createFakeEvent([]byte("unused"), &pr.author)

	fakeDpClient := client.NewMockClientInterface(t)
	fakeDpClient.EXPECT().PublishPlugin().RunAndReturn(func() error { return nil })
	fakeDpClient.EXPECT().PostEvents(mock.Anything, mock.Anything).RunAndReturn(
		func(bei events.BulkEventInterface, pbo *client.PublishBytesOptions) (*models.ResponsePostEvent, error) {
			litter.Config.DisablePointerReplacement = true
			t.Logf("%s", litter.Sdump(bei))
			bei.(*events.BulkStatusEvent).Events[0].Entity.RunTime = 0
			// Zero out all the timestamps
			bei.(*events.BulkStatusEvent).Events[0].Timestamp = time.Time{}
			bei.(*events.BulkStatusEvent).Events[0].Entity.Input.Timestamp = time.Time{}
			bei.(*events.BulkStatusEvent).Events[0].Entity.Input.Source.Timestamp = time.Time{}
			bei.(*events.BulkStatusEvent).Events[0].Entity.Input.Source.Path[0].Timestamp = time.Time{}
			bei.(*events.BulkStatusEvent).Events[0].Entity.Results[0].Timestamp = time.Time{}
			bei.(*events.BulkStatusEvent).Events[0].Entity.Results[0].Source.Timestamp = time.Time{}
			bei.(*events.BulkStatusEvent).Events[0].Entity.Results[0].Source.Path[0].Timestamp = time.Time{}
			bei.(*events.BulkStatusEvent).Events[0].Entity.Results[0].Source.Path[1].Timestamp = time.Time{}

			// This test ensures that enrichment events don't have any binary streams.
			// And it ensures changes to the output can be verified.

			require.Equal(t, &events.BulkStatusEvent{
				Events: []*events.StatusEvent{
					{
						ModelVersion: 5,
						KafkaKey:     "go-runner-placeholder",
						Timestamp:    time.Time{},
						Author: events.EventAuthor{
							Name:     "dummy-plugin",
							Version:  "1.0.0",
							Category: "plugin",
							Security: "",
						},
						Entity: events.StatusEntity{
							Input: events.BinaryEvent{
								ModelVersion: 5,
								KafkaKey:     "",
								Timestamp:    time.Time{},
								Author: events.EventAuthor{
									Name:     "dummy-plugin",
									Version:  "1.0.0",
									Category: "plugin",
									Security: "",
								},
								Source: events.EventSource{
									Name:       "source",
									References: map[string]string{},
									Security:   "",
									Path: []events.EventSourcePathNode{
										{
											Author: events.EventAuthor{
												Name:     "TestServer",
												Version:  "1",
												Category: "plugin",
												Security: "",
											},
											Action:           "extracted",
											Sha256:           "febe1d741b49e5a9c31526728d8c5134a803adfc4c04c4f052673722ed85597e",
											Relationship:     map[string]string(nil), // p0
											Timestamp:        time.Time{},
											FileFormatLegacy: "",
											FileFormat:       "",
											Size:             6,
											Filename:         "",
											Language:         "",
										},
									},
									Timestamp: time.Time{},
								},
								Entity: events.BinaryEntity{
									Sha256:           "febe1d741b49e5a9c31526728d8c5134a803adfc4c04c4f052673722ed85597e",
									Sha512:           "",
									Sha1:             "",
									Md5:              "",
									Ssdeep:           "",
									Tlsh:             "",
									Size:             6,
									Mime:             "",
									Magic:            "",
									FileFormatLegacy: "",
									FileFormat:       "",
									FileExtension:    "",
									Features:         []events.BinaryEntityFeature{},
									Datastreams:      []events.BinaryEntityDatastream{},
									Info:             json.RawMessage{},
								},
								Action:   "extracted",
								Dequeued: "test-fake-dequeued",
								Retries:  0,
								Flags: events.BinaryFlags{
									BypassCache: false,
									Expedite:    false,
									Retry:       false,
								},
								TrackSourceReferences: "",
								TrackLinks:            nil,
								TrackAuthors:          nil,
							},
							Status:  "completed",
							RunTime: 0,
							Error:   "",
							Message: "",
							Results: []events.BinaryEvent{
								{
									ModelVersion: 5,
									KafkaKey:     "go-runner-placeholder",
									Timestamp:    time.Time{},
									Author: events.EventAuthor{
										Name:     "dummy-plugin",
										Version:  "1.0.0",
										Category: "plugin",
										Security: "",
									},
									Source: events.EventSource{
										Name:       "source",
										References: map[string]string(nil), // p0
										Security:   "",
										Path: []events.EventSourcePathNode{
											{
												Author: events.EventAuthor{
													Name:     "TestServer",
													Version:  "1",
													Category: "plugin",
													Security: "",
												},
												Action:           "extracted",
												Sha256:           "febe1d741b49e5a9c31526728d8c5134a803adfc4c04c4f052673722ed85597e",
												Relationship:     map[string]string(nil), // p0
												Timestamp:        time.Time{},
												FileFormatLegacy: "",
												FileFormat:       "",
												Size:             6,
												Filename:         "",
												Language:         "",
											},
											{
												Author: events.EventAuthor{
													Name:     "dummy-plugin",
													Version:  "1.0.0",
													Category: "plugin",
													Security: "",
												},
												Action:           "enriched",
												Sha256:           "febe1d741b49e5a9c31526728d8c5134a803adfc4c04c4f052673722ed85597e",
												Relationship:     map[string]string(nil), // p0
												Timestamp:        time.Time{},
												FileFormatLegacy: "",
												FileFormat:       "",
												Size:             0,
												Filename:         "",
												Language:         "",
											},
										},
										Timestamp: time.Time{},
									},
									Entity: events.BinaryEntity{
										Sha256:           "febe1d741b49e5a9c31526728d8c5134a803adfc4c04c4f052673722ed85597e",
										Sha512:           "",
										Sha1:             "",
										Md5:              "",
										Ssdeep:           "",
										Tlsh:             "",
										Size:             6,
										Mime:             "",
										Magic:            "",
										FileFormatLegacy: "",
										FileFormat:       "",
										FileExtension:    "",
										Features: []events.BinaryEntityFeature{
											{
												Name:   "dummy",
												Value:  "value1",
												Type:   "string",
												Label:  "",
												Size:   0,
												Offset: 0,
											},
											{
												Name:   "dummy",
												Value:  "value2",
												Type:   "string",
												Label:  "",
												Size:   0,
												Offset: 0,
											},
											{
												Name:   "dummyInt",
												Value:  "10",
												Type:   "integer",
												Label:  "",
												Size:   0,
												Offset: 0,
											},
										},
										Datastreams: []events.BinaryEntityDatastream{}, // p1
										Info:        json.RawMessage{},                 // p2
									},
									Action:   "enriched",
									Dequeued: "",
									Retries:  0,
									Flags: events.BinaryFlags{
										BypassCache: false,
										Expedite:    false,
										Retry:       false,
									},
									TrackSourceReferences: "",
									TrackLinks:            nil,
									TrackAuthors:          nil,
								},
							},
						},
					},
				},
			}, bei)
			return &models.ResponsePostEvent{}, nil
		},
	)
	fakeDpClient.EXPECT().GetBinaryEvents(mock.Anything).RunAndReturn(
		func(fes *client.FetchEventsStruct) (*events.BulkBinaryEvent, *models.EventResponseInfo, error) {
			return &events.BulkBinaryEvent{
					Events: []*events.BinaryEvent{fakeEvent},
				}, &models.EventResponseInfo{
					Filtered:          20,
					Fetched:           1,
					Ready:             true,
					Paused:            false,
					ConsumersNotReady: "",
					Filters:           map[string]int{},
				}, nil
		},
	)
	pr.dpClient = fakeDpClient

	go func() { pr.Run() }()
	// Give time for the plugin to run.
	time.Sleep(500 * time.Millisecond)
	pr.cancelFunc()
}

func TestFormatRequireDataTypes(t *testing.T) {
	input := map[string][]string{
		"content": {"image/"},
	}
	require.Equal(t, []string{"content,image/"}, FormatRequireDataTypes(input))

	input2 := map[string][]string{
		"content": {""},
	}
	require.Equal(t, []string{"content,"}, FormatRequireDataTypes(input2))

	input3 := map[string][]string{
		"content": {
			// Windows exe
			"executable/windows/",
			// Non windows exe
			"executable/dll32",
			"executable/pe32",
			// Linux elf
			"executable/linux/elf64",
			"executable/linux/elf32",
			"executable/mach-o",
		},
	}
	require.Equal(t, []string{"content,executable/windows/,executable/dll32,executable/pe32,executable/linux/elf64,executable/linux/elf32,executable/mach-o"}, FormatRequireDataTypes(input3))

	input4 := map[string][]string{
		"content":  {"executable/"},
		"safe_png": {"image/"},
	}
	result := FormatRequireDataTypes(input4)
	slices.Sort(result)
	require.Equal(t, []string{"content,executable/", "safe_png,image/"}, result)
}

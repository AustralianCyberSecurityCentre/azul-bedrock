package plugin

import (
	"bufio"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/cart"
	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/client"
	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/events"
	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/testutils"
	"github.com/rs/zerolog"
)

type PluginInputUtils struct {
	Logger   *zerolog.Logger
	Settings *PluginSettings
}

type Plugin interface {
	/*Getters used by the runner framework.*/
	GetName() string
	GetVersion() string
	GetDescription() string
	GetFeatures() []events.PluginEntityFeature
	GetDefaultSettings() *PluginSettings
	/*Core execute method used to execute events provided by the PluginRunner framework.
	Expected errors are opt-out, timeout and generic errors.
	*/
	Execute(context context.Context, job *Job, inputUtils *PluginInputUtils) *PluginError
}

type PluginRunner struct {
	author           events.PluginEntity
	plugin           Plugin
	heartBeatChannel chan *events.BinaryEvent
	runContext       context.Context
	cancelFunc       context.CancelFunc
	dpClient         client.ClientInterface
	config           *PluginSettings
	logger           *zerolog.Logger
}

var localFileManager *testutils.FileManager

func createOrGetFileManager() (*testutils.FileManager, error) {
	var err error
	if localFileManager == nil {
		localFileManager, err = testutils.NewFileManager()
		if err != nil {
			return nil, err
		}
	}
	return localFileManager, nil
}

/*Create a new plugin runner.*/
func NewPluginRunner(inPlugin Plugin) *PluginRunner {
	heartBeatChannel := make(chan *events.BinaryEvent)
	context, cancelFunc := context.WithCancel(context.Background())
	settings := parsePluginSettings(inPlugin.GetDefaultSettings())
	// Set default deployment key if not set.
	if settings.DeploymentKey == "" {
		settings.DeploymentKey = fmt.Sprintf("plugin-%s", inPlugin.GetName())
	}

	author := events.PluginEntity{
		Name:        inPlugin.GetName(),
		Version:     inPlugin.GetVersion(),
		Contact:     "azul@asd.gov.au",
		Category:    "plugin",
		Description: inPlugin.GetDescription(),
		Features:    inPlugin.GetFeatures(),
		Config:      settings.convertToMap(),
	}

	dpClient := client.NewClient(settings.PluginEventsUrl, settings.PluginDataUrl, author, settings.DeploymentKey)

	return &PluginRunner{
		author:           author,
		plugin:           inPlugin,
		heartBeatChannel: heartBeatChannel,
		runContext:       context,
		cancelFunc:       cancelFunc,
		dpClient:         dpClient,
		config:           settings,
		logger:           &Logger,
	}
}

func (pr *PluginRunner) performHeartbeat(startTime time.Time, event *events.BinaryEvent) {
	result, err := NewJob(pr.dpClient, event, pr.author)
	if err != nil {
		pr.logger.Warn().Err(err).Msg("unable to perform heartbeat (setup) with error")
		return
	}
	defer result.Close()
	result.startTime = startTime
	heartbeatStatus := result.generateHeartbeat(pr.config)
	_, err = pr.dpClient.PostEvents(heartbeatStatus, &client.PublishEventsOptions{Model: events.ModelStatus})
	if err != nil {
		pr.logger.Warn().Err(err).Msg("unable to perform heartbeat (dispatcher contact) with error")
	}

}

/*Runs the heartbeat process continually sending heartbeat status events to dispatcher.*/
func (pr *PluginRunner) startHeartbeat() {
	nextHeartBeat := time.Duration(pr.config.HeartbeatIntervalSeconds) * time.Second
	heartBeatTicker := time.NewTicker(nextHeartBeat)
	var startTime time.Time
	var waitingOnEvent *events.BinaryEvent
	for {
		select {
		case waitingOnEvent = <-pr.heartBeatChannel:
			heartBeatTicker.Reset(nextHeartBeat)
		case startTime = <-heartBeatTicker.C:
			// Heartbeat if the current event is not nil.
			if (*events.BinaryEvent)(nil) != waitingOnEvent {
				pr.performHeartbeat(startTime, waitingOnEvent)
			}
		case <-pr.runContext.Done():
			pr.logger.Info().Msg("Runner exiting, shutdown heartbeats.")
			return
		}
	}
}

/*Startup background processes such as heartbeat.*/
func (pr *PluginRunner) startup() error {
	/*Start the heartbeats (continually send messages to dispatcher unless time is updated)*/
	if pr.config.HeartbeatIntervalSeconds > 0 {
		go pr.startHeartbeat()
	}

	err := pr.dpClient.PublishPlugin()
	if err != nil {
		pr.logger.Err(err).Msg("error when attempting to publish plugin.")
		return err
	}
	return nil
}

// Format required data types for RestAPI usage.
func FormatRequireDataTypes(inputFilters map[string][]string) []string {
	ret := []string{}
	for k, v := range inputFilters {
		returnString := fmt.Sprintf("%s,%s", k, strings.Join(v, ","))
		ret = append(ret, returnString)
	}
	return ret
}

/*Core run loop for running against dispatcher (returns a exit reason as a string).*/
func (pr *PluginRunner) Run() string {
	err := pr.startup()
	if err != nil {
		log.Fatalf("Failed to startup the plugin! Error: %v", err)
	}
	// Cancel if the function exits during an error.
	defer pr.cancelFunc()

	// Close context when a SIGINT or SIGTERM is received
	signal.NotifyContext(pr.runContext, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Main run loop.
	for {
		select {
		case <-pr.runContext.Done():
			reason := "Exiting runner after context was cancelled."
			pr.logger.Info().Msg(reason)
			return reason
		case pr.heartBeatChannel <- nil:
			// attempt to write nil to heartbeat, if the heartbeat receiver has closed this will skip.
		default:
		}
		// Get Event(s)
		inputBulkEvents, eventResponseInfo, err := pr.dpClient.GetBinaryEvents(&client.FetchEventsStruct{
			Count:                   1,
			Deadline:                30,
			AvroFormat:              false,
			IsTask:                  true,
			RequireUnderContentSize: int(pr.config.FilterMaxContentSize),
			RequireOverContentSize:  int(pr.config.FilterMinContentSize),
			RequireActions:          pr.config.FilterAllowEventTypes,
			RequireStreams:          FormatRequireDataTypes(pr.config.FilterDataTypes),
			RequireExpedite:         pr.config.RequireExpedite,
			RequireLive:             pr.config.RequireLive,
			RequireHistoric:         pr.config.RequireHistoric,
			DenySelf:                pr.config.FilterSelf,
		})
		// Something stopped us getting events from dispatcher (either a bad event that couldn't be decoded or dispatcher is unavailable.)
		if err != nil {
			pr.logger.Fatal().Err(err).Msg("couldn't get events from dispatcher")
		}

		if eventResponseInfo.Filtered > 0 {
			pr.logger.Info().Msgf("%d  uninteresting events filtered by dispatcher", eventResponseInfo.Filtered)
			// No event found but some were filtered look for more events.
			if eventResponseInfo.Fetched == 0 {
				continue
			}
		}
		if eventResponseInfo.Fetched == 0 {
			if eventResponseInfo.Paused {
				pr.logger.Info().Msg("Plugin has been paused sleeping and then trying to get more events.")
			} else {
				pr.logger.Info().Msg("No events fetched or filtered by dispatcher; backing off for 10 seconds")
			}
			time.Sleep(time.Duration(10) * time.Second)
			continue
		}

		for _, event := range inputBulkEvents.Events {
			// Done this way to guarantee the job always closes.
			err = pr.runInner(event)
			if err != nil {
				return err.Error()
			}
		}
	}
}

// Inner runInner function returns nil to skip to the next event returns error to exit the runner.
func (pr *PluginRunner) runInner(event *events.BinaryEvent) error {
	// Write to heartbeat channel unless context is already done.
	select {
	case <-pr.runContext.Done():
	case pr.heartBeatChannel <- event:
	}

	// Ensure this get's closed to clear up all the temporary files.
	job, err := NewJob(pr.dpClient, event, pr.author)
	if err != nil {
		// This is expected to occur only if the application can't create a directory in temp.
		pr.logger.Fatal().Err(err).Msg("couldn't setup job (check your pod's permissions for temp)")
	}
	defer job.Close()
	if len(event.Source.Path) >= pr.config.DepthLimit {
		_, err = pr.dpClient.PostEvents(job.generateErrorResults(
			NewPluginError(OptOutError, "", fmt.Sprintf("%s reached configured plugin_depth_limit", pr.plugin.GetName()))),
			&client.PublishEventsOptions{Model: events.ModelStatus})
		if err != nil {
			pr.logger.Err(err).Msg("Failed to publish error result events to dispatcher.")
			return errors.New("failed to publish error result events to dispatcher")
		}
		return nil
	}
	sourceEntity := job.GetSourceEvent().Entity
	pr.logger.Info().Msgf("received plugin=%s file_format=%s size=%d sha256=%s", pr.plugin.GetName(), sourceEntity.FileFormat, sourceEntity.Size, sourceEntity.Sha256)
	pluginError := pr.runEvent(&job)
	var outputBulkEvents *events.BulkStatusEvent
	if pluginError == nil {
		outputBulkEvents = job.generateResultEvents()
	} else {
		outputBulkEvents = job.generateErrorResults(pluginError)
	}
	state := events.StatusTypeErrorException
	errorMessage := ""
	if len(outputBulkEvents.Events) > 0 {
		state = outputBulkEvents.Events[0].Entity.Status
		if events.IsStatusTypeError(state) {
			errorMessage = fmt.Sprintf("; Plugin Error Title: %s Message: %s", outputBulkEvents.Events[0].Entity.Error, outputBulkEvents.Events[0].Entity.Message)
		}
	}
	pr.logger.Info().Msgf("finish plugin=%s state=%s file_format=%s size=%d sha256=%s%s", pr.plugin.GetName(), state, sourceEntity.FileFormat, sourceEntity.Size, sourceEntity.Sha256, errorMessage)
	_, err = pr.dpClient.PostEvents(outputBulkEvents, &client.PublishEventsOptions{Model: events.ModelStatus})
	if err != nil {
		pr.logger.Err(err).Msg("Failed to publish result events to dispatcher.")
		return errors.New("failed to publish result events to dispatcher")
	}
	return nil
}

func (pr *PluginRunner) runEvent(jobRef *Job) *PluginError {
	var pluginError *PluginError
	err := pr.plugin.Execute(pr.runContext, jobRef, &PluginInputUtils{
		Logger:   pr.logger,
		Settings: pr.config,
	})
	jobRef.SetEndTime()
	if err != nil {
		// Handle the error result ensuring it is a plugin error.
		if !errors.As(err, &pluginError) {
			err = NewPluginError(err, "Unknown error", "unknown error occurred in plugin.").WithCausalError(err)
			errors.As(err, &pluginError)
		}
		return pluginError
	}

	// Flatten all the JobEvents into a single list.
	allJobEvents := jobRef.getAllJobEventsAsList()
	// Validate all the features are correct.
	for _, je := range allJobEvents {
		pluginError = jobRef.validateAndMutateFeatures(je, pr.config)
		if pluginError != nil {
			return pluginError
		}
	}
	return nil
}

type RunTestOptions struct {
	// Sha256 of the file to be tested that can be downloaded using FileManager. (not affected by disable uncarting)
	DownloadSha256 string // First preference
	// Path to the file being tested against
	PathToContentFile string // First preference
	// Raw bytes of the file to be used for testing
	ContentFileBytes []byte // Second preference

	// Disable the uncarting for the source content file.
	DisableUncartingContentFile bool

	IncludeStreamsInResult bool
}

/*Mock for posting a generic stream to dispatcher.*/
func mockPostStreamResponse(source string, label events.DatastreamLabel, reader io.Reader, query *client.PostStreamStruct) (*events.BinaryEntityDatastream, error) {
	bufferedReader := bufio.NewReader(reader)
	sha256Sum := sha256.New()
	totalBytes, err := bufferedReader.WriteTo(sha256Sum)
	if err != nil {
		return &events.BinaryEntityDatastream{}, err
	}
	return &events.BinaryEntityDatastream{
		Label:  label,
		Size:   uint64(totalBytes),
		Sha256: fmt.Sprintf("%x", sha256Sum.Sum(nil)),
	}, nil
}

/*Run the plugin in test mode which doesn't start up un-necessary things*/
func (pr *PluginRunner) RunTest(t *testing.T, runOptions *RunTestOptions, fileDescription string) *TestJobResult {
	var testFileBytes []byte
	pathToFileToUncart := ""
	if len(runOptions.DownloadSha256) > 0 {
		curFileManager, err := createOrGetFileManager()
		if err != nil {
			t.Fatalf("Failed to startup FileManager to download test files with error %s", err.Error())
		}
		fileBytes, err := curFileManager.DownloadFileBytes(runOptions.DownloadSha256)
		if err != nil {
			t.Fatalf("Failed to download provided sha256 %s with error %s", runOptions.DownloadSha256, err.Error())
		}
		if len(fileBytes) == 0 {
			t.Fatalf("Zero bytes in downloaded file %s", runOptions.DownloadSha256)
		}
		testFileBytes = fileBytes
	} else if len(runOptions.PathToContentFile) > 0 {
		if runOptions.DisableUncartingContentFile {
			f, err := os.Open(runOptions.PathToContentFile)
			if err != nil {
				t.Fatalf("Failed to open the test file '%s' with error %v", runOptions.PathToContentFile, err)
			}
			defer f.Close()
			testFileBytes, err = io.ReadAll(f)
			if err != nil {
				t.Fatalf("Failed to read test file '%s' with error %v", runOptions.PathToContentFile, err)
			}
		} else {
			pathToFileToUncart = runOptions.PathToContentFile
		}
	} else if len(runOptions.ContentFileBytes) > 0 {
		if runOptions.DisableUncartingContentFile {
			testFileBytes = runOptions.ContentFileBytes
		} else {
			fileHandle, err := os.CreateTemp("", "runner-cached-cart-")
			if err != nil {
				t.Fatalf("Failed to create a temporary file when uncarting raw bytes.")
			}
			_, err = fileHandle.Write(runOptions.ContentFileBytes)
			if err != nil {
				t.Fatalf("Failed to write raw bytes to a temporary file for uncarting.")
			}
			fileHandle.Close()
			defer os.Remove(fileHandle.Name())
			pathToFileToUncart = fileHandle.Name()
		}
	} else {
		t.Fatalf("No content provided for go-runner test and it's required.")
	}
	// Uncart the file or the bytes converted into a file. (Doesn't uncart when using a sha256 because the file is already raw)
	if len(runOptions.DownloadSha256) == 0 && !runOptions.DisableUncartingContentFile {
		reader, err := cart.Uncart(pathToFileToUncart)
		if err != nil {
			t.Fatalf("Failed to Uncart test file '%s' with error %v", pathToFileToUncart, err)
		}
		defer reader.Close()
		testFileBytes, err = io.ReadAll(reader)
		if err != nil {
			t.Fatalf("Failed to read Uncarted test file '%s' with error %v", pathToFileToUncart, err)
		}
	}
	fakeEvent := createFakeEvent(testFileBytes, &pr.author)
	fakeDpClient := createMiniFakeDispatcher(t, testFileBytes)

	job, err := NewJob(fakeDpClient, fakeEvent, pr.author)
	if err != nil {
		t.Fatalf("Failed to create mock result builder with error %v", err)
	}
	defer job.Close()

	pluginError := pr.runEvent(&job)
	return NewTestJobResult(&job, pluginError, &TestJobResultOptions{IncludeRawBytes: runOptions.IncludeStreamsInResult})
}

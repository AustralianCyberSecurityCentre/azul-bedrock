package plugin

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/client"
	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/events"
)

// Method for building up the result.
type Job struct {
	startTime             time.Time
	endTime               time.Time
	dpclient              client.ClientInterface
	sourceEvent           *events.BinaryEvent
	simpleSourceEvent     *events.BinaryEvent
	authorProcessingEvent events.PluginEntity
	temporaryDirPath      string
	rootEvent             *JobEvent
	warnings              []string
}

// Create a new result builder ready to pass into a plugin execution loop.
func NewJob(dpclient client.ClientInterface, sourceEvent *events.BinaryEvent, authorProcessingEvent events.PluginEntity) (Job, error) {
	tempDir, err := os.MkdirTemp("", "gorunner")
	if err != nil {
		return Job{}, fmt.Errorf("failed to create temporary directory for Job with error %v", err)
	}
	// Remove un-necessary information from source event that we don't want re-published.
	simpleSourceEvent := *sourceEvent
	simpleSourceEvent.Entity.Features = []events.BinaryEntityFeature{}
	simpleSourceEvent.Entity.Info = json.RawMessage{}
	simpleSourceEvent.Entity.Datastreams = []events.BinaryEntityDatastream{}

	return Job{
		startTime:             time.Now().UTC(),
		endTime:               time.Now().UTC(),
		dpclient:              dpclient,
		sourceEvent:           sourceEvent,
		simpleSourceEvent:     &simpleSourceEvent,
		authorProcessingEvent: authorProcessingEvent,
		temporaryDirPath:      tempDir,
		rootEvent:             NewJobEvent(tempDir, nil),
		warnings:              []string{},
	}, nil
}

func (jobRef *Job) SetEndTime() {
	jobRef.endTime = time.Now().UTC()
}

func (jobRef *Job) GetSourceEvent() events.BinaryEvent {
	return *jobRef.sourceEvent
}

/*Find the index of the content stream.*/
func (jobRef *Job) getContentDataStreamIndex() (int, error) {
	for i, label := range jobRef.sourceEvent.Entity.Datastreams {
		if label.Label == events.DataLabelContent {
			return i, nil
		}
	}
	return -1, fmt.Errorf("failed to find content label for Input Event with sha256 '%s'", jobRef.sourceEvent.Entity.Sha256)
}

/*Download Content Label by chunk.*/
func (jobRef *Job) GetContentChunk(startChunk uint64, endChunk uint64) ([]byte, bool, *PluginError) {
	jobId := jobRef.sourceEvent.Entity.Sha256
	// Locate the correct stream.
	dataStreamIndex, err := jobRef.getContentDataStreamIndex()
	if err != nil {
		return []byte{}, true, NewPluginError(
			ErrorRunner,
			"Failed to Download Content Chunk",
			fmt.Sprintf("Failed to find content stream id in job with id  %s.", jobId),
		).WithCausalError(err)
	}
	contentStream := jobRef.sourceEvent.Entity.Datastreams[dataStreamIndex]
	// Ensure start and end of chunk are set appropriately.
	if startChunk > contentStream.Size {
		return []byte{}, true, NewPluginError(
			ErrorRunner,
			"Failed to Download Content Chunk",
			fmt.Sprintf("Requested start chunk is after end of total size of file for job with id %s.", jobId),
		).WithCausalError(err)
	}
	// Check if the end of file has been reached and ensure the endChunk isn't overshooting.
	endOfFile := false
	if endChunk >= contentStream.Size-1 {
		endChunk = contentStream.Size - 1
		endOfFile = true
	}
	// Download the chunk of file.
	chunk, err := jobRef.dpclient.DownloadBinaryChunk(jobRef.sourceEvent.Source.Name, contentStream.Label, contentStream.Sha256, startChunk, endChunk)
	if err != nil {
		return nil, false, NewPluginError(
			ErrorRunner,
			"Failed to Download Content Chunk",
			fmt.Sprintf("Failed to download content chunk for job with id %s.", jobId),
		).WithCausalError(err)
	}
	return chunk, endOfFile, nil
}

/*Download Content Label and get file path to the file, also cleans up file after plugin completion.*/
func (jobRef *Job) GetContentPath() (string, *PluginError) {
	// Download the stream.
	jobId := jobRef.sourceEvent.Entity.Sha256
	dataStreamIndex, err := jobRef.getContentDataStreamIndex()
	if err != nil {
		return "", NewPluginError(
			ErrorRunner,
			"Failed to Download Content",
			fmt.Sprintf("Failed to find content stream id in job with id %s.", jobId),
		).WithCausalError(err)
	}
	contentStream := jobRef.sourceEvent.Entity.Datastreams[dataStreamIndex]
	bufferedReader, err := jobRef.dpclient.DownloadBinary(jobRef.sourceEvent.Source.Name, contentStream.Label, contentStream.Sha256)
	if err != nil {
		return "", NewPluginError(
			ErrorRunner,
			"Failed to Download Content",
			fmt.Sprintf("Failed to download content in job with id %s.", jobId),
		).WithCausalError(err)
	}
	tempFile, err := os.CreateTemp(jobRef.temporaryDirPath, "runner-content-")
	if err != nil {
		return "", NewPluginError(
			ErrorRunner,
			"Failed to Download Content",
			fmt.Sprintf("Failed to create a temporary file when downloading content in job with id %s.", jobId),
		).WithCausalError(err)
	}
	// Save the stream to disk.
	defer tempFile.Close()
	_, err = bufferedReader.WriteTo(tempFile)

	if err != nil {
		return "", NewPluginError(ErrorRunner,
			"Failed to Download Content",
			fmt.Sprintf("Failed to write downloaded content file to cache with job id %s.", jobId),
		).WithCausalError(err)

	}
	return tempFile.Name(), nil
}

/*Collect the tree structure of the jobs as a flat list.*/
func (jobRef *Job) getAllJobEventsAsList() []*JobEvent {
	allJobEvent := []*JobEvent{}
	allJobEvent = append(allJobEvent, jobRef.rootEvent)
	jobEventsToCheck := []*JobEvent{}
	jobEventsToCheck = append(jobEventsToCheck, jobRef.rootEvent)

	for len(jobEventsToCheck) > 0 {
		startingLength := len(jobEventsToCheck)
		for _, currentJobEvent := range jobEventsToCheck {
			for _, currentChild := range currentJobEvent.children {
				// Ignore the job if we've already seen it (avoid infinite loops).
				if slices.Contains(allJobEvent, currentChild) {
					continue
				}
				// Add the jobs to the flat list and to the future jobs to check.
				allJobEvent = append(allJobEvent, currentChild)
				jobEventsToCheck = append(jobEventsToCheck, currentChild)
			}
		}
		// only take the new jobs that need checking.
		jobEventsToCheck = jobEventsToCheck[startingLength:]
	}
	return allJobEvent
}

/*Standardised way to get the stream key from uploaded binaries.*/
func getStreamKey(rs *ResultStream) string {
	return fmt.Sprintf("%s-%s", rs.Sha256, rs.Label)
}

/*Upload the child and augmented binaries to Azul and get the metadata to be added as child events.*/
func (jobRef *Job) uploadChildAndAugmentedBinaries(allJobEvents []*JobEvent) (map[string]*events.BinaryEntityDatastream, error) {

	uploadedStreamsWithLabels := map[string]*events.BinaryEntityDatastream{}
	var result *events.BinaryEntityDatastream
	var err error

	for _, currentJobEvent := range allJobEvents {
		for _, s := range currentJobEvent.augmentedAndContentStreams {
			if len(s.path) > 0 {
				file, err := os.Open(s.path)
				if err != nil {
					return uploadedStreamsWithLabels, err
				}
				bufReader := bufio.NewReader(file)
				result, err = jobRef.dpclient.PostStream(jobRef.sourceEvent.Source.Name, s.Label, bufReader, &client.PostStreamStruct{})
				if err != nil {
					return uploadedStreamsWithLabels, err
				}
			} else if len(s.RawBytes) > 0 {
				result, err = jobRef.dpclient.PostStream(jobRef.sourceEvent.Source.Name, s.Label, bytes.NewReader(s.RawBytes), &client.PostStreamStruct{})
			} else {
				err = fmt.Errorf("provided stream with label %s and relationship %v doesn't have content", s.Label, s.Relationship)
			}
			if err != nil {
				return uploadedStreamsWithLabels, err
			}
			uploadedStreamsWithLabels[getStreamKey(s)] = result
		}
	}
	return uploadedStreamsWithLabels, err
}

/*Verify and normalise features and add them to the provided resultEntity.*/
func (jobRef *Job) validateAndMutateFeatures(jobEvent *JobEvent, settings *PluginSettings) *PluginError {
	// Get all of the features that the plugin has registered.
	allowedFeatureNames := make(map[string]events.PluginEntityFeature)
	for _, featName := range jobRef.authorProcessingEvent.Features {
		allowedFeatureNames[featName.Name] = featName
	}
	// Check all the provided features are valid and have been registered by the plugin.
	// Also set the type to whatever was registered even if it was set by the user.
	invalidFeatures := []string{}
	for idx, feat := range jobEvent.features {
		if sourceFeat, ok := allowedFeatureNames[feat.Name]; !ok {
			invalidFeatures = append(invalidFeatures, feat.Name)
		} else {
			jobEvent.features[idx].Type = sourceFeat.Type
		}
	}
	if len(invalidFeatures) > 0 {
		return NewPluginError(
			ErrorsOutput,
			"Plugin Unregistered Features",
			fmt.Sprintf("Plugin tried to set undeclared features: %v", invalidFeatures),
		)
	}

	validationOptions := events.NewValidationOptions().WithMaxValuesInOneFeatureCount(settings.MaxValuesPerFeature).WithMaxFVLength(settings.MaxValueLength)
	feats, warnings, err := events.ProcessAndValidateBinaryFeatures(jobEvent.features, validationOptions)
	if err != nil {
		return NewPluginError(
			ErrorRunner,
			"Feature Error",
			"Error occurred when attempting to process features.",
		).WithCausalError(err)
	}
	jobEvent.features = feats
	jobRef.warnings = append(jobRef.warnings, warnings...)
	// Sort warnings for consistency
	slices.Sort(jobRef.warnings)
	return nil
}

/*Create a binaryEvent from a binaryEntity.*/
func (jobRef *Job) createBinaryEvent(inEntity *events.BinaryEntity, newPathNodes []*events.EventSourcePathNode, actionType events.BinaryAction, createAtTime time.Time) (*events.BinaryEvent, error) {
	sourceCopy, err := jobRef.sourceEvent.Source.DeepCopy()
	if err != nil {
		return &events.BinaryEvent{}, err
	}

	for _, node := range newPathNodes {
		sourceCopy.Path = append(sourceCopy.Path, *node)
	}

	return &events.BinaryEvent{
		ModelVersion: events.CurrentModelVersion,
		KafkaKey:     "go-runner-placeholder",
		Author:       jobRef.authorProcessingEvent.Summary(),
		Timestamp:    createAtTime,
		Source:       sourceCopy,
		Action:       actionType,
		Entity:       *inEntity,
	}, nil
}

func (jobRef *Job) generateErrorResults(pluginError *PluginError) *events.BulkStatusEvent {
	statusResults := events.BulkStatusEvent{Events: []*events.StatusEvent{}}
	now := time.Now().UTC()
	resultStatusEvent := events.StatusEvent{
		ModelVersion: events.CurrentModelVersion,
		KafkaKey:     "go-runner-placeholder",
		Timestamp:    now,
		Author:       jobRef.authorProcessingEvent.Summary(),
		Entity: events.StatusEntity{
			Input:   *jobRef.simpleSourceEvent,
			Status:  jobRef.getEventResultStatus(pluginError),
			RunTime: jobRef.endTime.Sub(jobRef.startTime).Seconds(),
			Error:   pluginError.GetTitle(),
			Message: jobRef.getEventResultMessage(pluginError),
		},
	}
	err := resultStatusEvent.CheckValid()
	if err != nil {
		log.Fatalf("Failed to generate a status event %v", err)
	}
	statusResults.Events = append(statusResults.Events, &resultStatusEvent)

	return &statusResults
}

/*Generate a heartbeat events.*/
func (jobRef *Job) generateHeartbeat(settings *PluginSettings) *events.BulkStatusEvent {
	now := time.Now().UTC()

	resultStatusEvent := events.StatusEvent{
		ModelVersion: events.CurrentModelVersion,
		KafkaKey:     "go-runner-placeholder",
		Timestamp:    now,
		Author:       jobRef.authorProcessingEvent.Summary(),
		Entity: events.StatusEntity{
			Input:   *jobRef.simpleSourceEvent,
			Status:  events.StatusTypeHeartbeat,
			RunTime: time.Since(jobRef.startTime).Seconds(),
		},
	}
	err := resultStatusEvent.CheckValid()
	if err != nil {
		return jobRef.generateErrorResults(NewPluginError(
			ErrorRunner,
			"Error Status Event",
			"status event was invalid.",
		).WithCausalError(err))
	}
	return &events.BulkStatusEvent{Events: []*events.StatusEvent{&resultStatusEvent}}
}

/*Get the current successful result stats.*/
func (jobRef *Job) getEventResultStatus(pluginError *PluginError) string {
	if pluginError != nil {
		switch pluginError.innerError {
		case OptOutError:
			return events.StatusTypeOptOut
		case ErrorException:
			return events.StatusTypeErrorException
		case ErrorNetwork:
			return events.StatusTypeErrorNetwork
		case ErrorRunner:
			return events.StatusTypeErrorRunner
		case ErrorInput:
			return events.StatusTypeErrorInput
		case ErrorsOutput:
			return events.StatusTypeErrorOutput
		case ErrorTimeout:
			return events.StatusTypeErrorTimeout
		default:
			return events.StatusTypeErrorException
		}
	}

	if len(jobRef.rootEvent.features) == 0 && len(jobRef.rootEvent.augmentedAndContentStreams) == 0 && (len(jobRef.rootEvent.info) == 0) {
		return events.StatusTypeCompletedEmpty
	}
	if len(jobRef.warnings) > 0 {
		return events.StatusTypeCompletedWithErrors
	}
	return events.StatusTypeCompleted
}

/*Get the current result message */
func (jobRef *Job) getEventResultMessage(pluginError *PluginError) string {
	if pluginError != nil {
		return pluginError.GetMessage()
	}

	if len(jobRef.warnings) > 0 {
		return fmt.Sprintf("Partial completion occurred with the following errors: %s", strings.Join(jobRef.warnings, "\n"))
	}
	return ""
}

/*Generate all the events including the core event with features etc.*/
func (jobRef *Job) generateResultEvents() *events.BulkStatusEvent {
	now := time.Now().UTC()

	// Flatten the job structure.
	allJobEvents := jobRef.getAllJobEventsAsList()

	var err error
	uploadedStreamsWithLabels, err := jobRef.uploadChildAndAugmentedBinaries(allJobEvents)
	if err != nil {
		return jobRef.generateErrorResults(NewPluginError(
			ErrorNetwork,
			"Error uploading binaries",
			"An error occurred when trying to upload child and augmented binaries",
		).WithCausalError(err))
	}

	allBinaryEventResults := []events.BinaryEvent{}
	var resultBinaryEvent *events.BinaryEvent
	// Generate all binary events.
	for _, je := range allJobEvents {
		// Root event either enrichment or augmented
		if je.parent == nil {
			resultEntity := jobRef.sourceEvent.Entity.CopyWithDataStreams()
			resultEntity.Info = je.info
			resultEntity.Features = je.features
			action := events.ActionEnriched
			if len(je.augmentedAndContentStreams) > 0 {
				action = events.ActionAugmented
				simplifiedStreams := []events.BinaryEntityDatastream{}
				// Get rid of all parent alt streams and just keep the parent content stream.
				// Just in case input event is an augmented Event.
				for _, s := range resultEntity.Datastreams {
					if s.Label == events.DataLabelContent {
						simplifiedStreams = append(simplifiedStreams, s)
					}
				}
				resultEntity.Datastreams = simplifiedStreams
				for _, augStream := range je.augmentedAndContentStreams {
					s, ok := uploadedStreamsWithLabels[getStreamKey(augStream)]
					if !ok {
						return jobRef.generateErrorResults(NewPluginError(
							ErrorRunner,
							"Error Expected stream missing",
							fmt.Sprintf("Could not find expected stream with Label, sha256 (%s, %s) in list of uploaded streams.", augStream.Label, augStream.Sha256),
						))
					}
					resultEntity.Datastreams = append(resultEntity.Datastreams, *s)
				}
			} else {
				// Enriched events can't have any streams
				resultEntity.Datastreams = []events.BinaryEntityDatastream{}
			}

			pathNodes := []*events.EventSourcePathNode{{
				Author:    jobRef.authorProcessingEvent.Summary(),
				Action:    action,
				Sha256:    resultEntity.Sha256,
				Timestamp: now,
			}}
			resultBinaryEvent, err = jobRef.createBinaryEvent(resultEntity, pathNodes, action, now)
		} else { // Extracted event
			extractedContentStream, err := je.getContentStream()
			if err != nil {
				return jobRef.generateErrorResults(NewPluginError(
					ErrorRunner,
					"Error Child Missing Content",
					"A child event has no content which is required.",
				).WithCausalError(err))
			}
			uploadedFile, ok := uploadedStreamsWithLabels[getStreamKey(extractedContentStream)]
			if !ok {
				return jobRef.generateErrorResults(NewPluginError(
					ErrorRunner,
					"Error Expected Stream Missing",
					fmt.Sprintf("Could not find expected child content stream with Label, sha256 (%s, %s) in list of uploaded streams.", extractedContentStream.Label, extractedContentStream.Sha256),
				))
			}
			resultEntity := uploadedFile.ToInputEntity()
			resultEntity.Features = append(resultEntity.Features, je.features...)

			// Append any augmented streams associated with the child event.
			for _, s := range je.augmentedAndContentStreams {
				if s.Label != events.DataLabelContent {
					augStreamRef, ok := uploadedStreamsWithLabels[getStreamKey(s)]
					if !ok {
						return jobRef.generateErrorResults(NewPluginError(
							ErrorRunner,
							"Error Expected Stream Missing",
							fmt.Sprintf("Could not find expected augmented stream in child event with Label, sha256 (%s, %s) in list of uploaded streams.", extractedContentStream.Label, extractedContentStream.Sha256),
						))
					}
					resultEntity.Datastreams = append(resultEntity.Datastreams, *augStreamRef)
				}
			}
			// Add info if it's present.
			if je.info != nil {
				resultEntity.Info = je.info
			}

			pathNodes, err := je.recursivelyGetExtractedPathNodes(jobRef.authorProcessingEvent.Summary(), now, uploadedStreamsWithLabels)
			if err != nil {
				return jobRef.generateErrorResults(NewPluginError(
					ErrorRunner,
					"Error Generating Extracted Path",
					fmt.Sprintf("error when attempting to generate the path for the extracted event with the sha256 %s.", resultEntity.Sha256),
				).WithCausalError(err))
			}
			resultBinaryEvent, err = jobRef.createBinaryEvent(resultEntity, pathNodes, events.ActionExtracted, now)
			if err != nil {
				return jobRef.generateErrorResults(NewPluginError(
					ErrorRunner,
					"Error Generating Binary events",
					fmt.Sprintf("error when attempting to generate the binary event for the sha256 %s.", resultEntity.Sha256),
				).WithCausalError(err))
			}
		}

		if err == nil {
			err = resultBinaryEvent.CheckValid()
		}
		// If there is an error return an error status result instead.
		if err != nil {
			return jobRef.generateErrorResults(NewPluginError(
				ErrorRunner,
				"Error Binary Event",
				"binary event was invalid",
			).WithCausalError(err))
		}
		allBinaryEventResults = append(allBinaryEventResults, *resultBinaryEvent)
	}

	resultStatusEvent := events.StatusEvent{
		ModelVersion: events.CurrentModelVersion,
		KafkaKey:     "go-runner-placeholder",
		Timestamp:    now,
		Author:       jobRef.authorProcessingEvent.Summary(),
		Entity: events.StatusEntity{
			Input:   *jobRef.simpleSourceEvent,
			Status:  jobRef.getEventResultStatus(nil),
			Message: jobRef.getEventResultMessage(nil),
			RunTime: jobRef.endTime.Sub(jobRef.startTime).Seconds(),
			Results: allBinaryEventResults,
		},
	}
	err = resultStatusEvent.CheckValid()
	if err != nil {
		return jobRef.generateErrorResults(NewPluginError(
			ErrorRunner,
			"Error Status Event",
			"status event was invalid",
		).WithCausalError(err))
	}
	return &events.BulkStatusEvent{Events: []*events.StatusEvent{&resultStatusEvent}}
}

/*Delete all resources held by result builder including files cached in the temporary directory.*/
func (jobRef *Job) Close() {
	// Fatal error because if this fails we have a memory leak.
	err := os.RemoveAll(jobRef.temporaryDirPath)
	if err != nil {
		log.Fatalf("Failed to cleanup result builder's temporary directory with error: %v", err)
	}
}

// Job Event proxy methods
// Add a feature to the event, note type is not required and will be set to whatever was set in the plugin's registered feature list.
func (jobRef *Job) AddFeature(name string, value interface{}) *PluginError {
	return jobRef.rootEvent.AddFeature(name, value)
}

// Add a feature to the job event, and a label, offset and/or size as well.
func (jobRef *Job) AddFeatureWithExtra(name string, value interface{}, options *AddFeatureOptions) *PluginError {
	return jobRef.rootEvent.AddFeatureWithExtra(name, value, options)
}

/*Add info.*/
func (jobRef *Job) AddInfo(info json.RawMessage) {
	jobRef.rootEvent.AddInfo(info)
}

/*Add child file as part of results with a relationship to the current binary.*/
func (jobRef *Job) AddChildBytes(data []byte, relationship map[string]string) *JobEvent {
	return jobRef.rootEvent.AddChildBytes(data, relationship)
}

/*Add augmented file as part of results with a relationship to the current binary.*/
func (jobRef *Job) AddAugmentedBytes(data []byte, label events.DatastreamLabel) error {
	return jobRef.rootEvent.AddAugmentedBytes(data, label)
}

/*Add child file as part of results with a relationship to the current binary (Plugin is expected to delete original file).*/
func (jobRef *Job) AddChild(dataPath string, relationship map[string]string) (*JobEvent, error) {
	return jobRef.rootEvent.AddChild(dataPath, relationship)
}

/*Add augmented file as part of results with a relationship to the current binary (Plugin is expected to delete original file).*/
func (jobRef *Job) AddAugmented(dataPath string, label events.DatastreamLabel) error {
	return jobRef.rootEvent.AddAugmented(dataPath, label)
}

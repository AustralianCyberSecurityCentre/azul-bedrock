package plugin

import (
	"encoding/json"
	"io"
	"os"
	"testing"
	"time"

	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var jobTestFakeTestFileContent = []byte("fake file for job tests!")

func createFakeAuthorSummary(inPlugin Plugin) *events.PluginEntity {
	return &events.PluginEntity{
		Name:        inPlugin.GetName(),
		Version:     inPlugin.GetVersion(),
		Contact:     "azul@asd.gov.au",
		Category:    "plugin",
		Description: inPlugin.GetDescription(),
		Features:    inPlugin.GetFeatures(),
		Config:      map[string]string{},
	}
}

func createDummyJob(t *testing.T) *Job {
	testFile := jobTestFakeTestFileContent
	dummyPlugin := &DummyPlugin{}
	dpClient := createMiniFakeDispatcher(t, testFile)
	authorSummary := createFakeAuthorSummary(dummyPlugin)
	fakeEvent := createFakeEvent(testFile, authorSummary)
	job, err := NewJob(dpClient, fakeEvent, *authorSummary)
	require.Nil(t, err)
	return &job
}

func TestStartStopTimes(t *testing.T) {
	j := createDummyJob(t)
	defer j.Close()
	beforeSetTime := time.Now().UTC()
	j.SetEndTime()
	currentTime := time.Now().UTC()
	require.GreaterOrEqual(t, j.endTime, beforeSetTime)
	require.GreaterOrEqual(t, currentTime, j.endTime)
}

func TestGetContentStreamIndex(t *testing.T) {
	j := createDummyJob(t)
	defer j.Close()
	i, err := j.getContentDataStreamIndex()
	require.Nil(t, err)
	require.Equal(t, 0, i)

	// Hide content stream in middle of other streams.
	j.sourceEvent.Entity.Datastreams = []events.BinaryEntityDatastream{
		{Label: events.DataLabelText, Size: 1025, Sha256: "702e31ed1537c279459a255460f12f0f2863f973e121cd9194957f4f3e7b099a"},
		{Label: events.DataLabelTest, Size: 1025, Sha256: "702e31ed1537c279459a255460f12f0f2863f973e121cd9194957f4f3e7b099b"},
		j.sourceEvent.Entity.Datastreams[0],
		{Label: events.DataLabelDeobJs, Size: 1025, Sha256: "702e31ed1537c279459a255460f12f0f2863f973e121cd9194957f4f3e7b099c"},
		{Label: events.DataLabelAssemblyline, Size: 1025, Sha256: "702e31ed1537c279459a255460f12f0f2863f973e121cd9194957f4f3e7b099d"},
	}
	i, err = j.getContentDataStreamIndex()
	require.Nil(t, err)
	require.Equal(t, 2, i)
}

func TestGetContentChunk(t *testing.T) {
	j := createDummyJob(t)
	defer j.Close()
	data, isEnd, err := j.GetContentChunk(4, 10)
	require.Nil(t, err)
	require.False(t, isEnd)
	require.Equal(t, jobTestFakeTestFileContent[4:10+1], data)

	data, isEnd, err = j.GetContentChunk(0, 3)
	require.Nil(t, err)
	require.False(t, isEnd)
	require.Equal(t, jobTestFakeTestFileContent[0:3+1], data)

	data, isEnd, err = j.GetContentChunk(0, 100000)
	require.Nil(t, err)
	require.True(t, isEnd)
	require.Equal(t, jobTestFakeTestFileContent[:], data)

}

func TestGetContentPath(t *testing.T) {
	j := createDummyJob(t)
	defer j.Close()
	path, pluginErr := j.GetContentPath()
	require.Nil(t, pluginErr)
	f, err := os.Open(path)
	require.Nil(t, err)
	defer f.Close()

	rawBytes, err := io.ReadAll(f)
	require.Nil(t, err)
	require.Equal(t, jobTestFakeTestFileContent, rawBytes)
}

func TestGetAllJobEventsAsList(t *testing.T) {
	j := createDummyJob(t)
	defer j.Close()
	err := j.AddFeature("dummy", "value1")
	require.Nil(t, err)
	jobList := j.getAllJobEventsAsList()
	require.Equal(t, 1, len(jobList))

	j.AddChildBytes([]byte("abcdef child binary!"), map[string]string{"child": "extracted"})
	jobList = j.getAllJobEventsAsList()
	require.Equal(t, 2, len(jobList))

	j.AddAugmentedBytes([]byte("Augmented stream woo!"), events.DataLabelTest)
	jobList = j.getAllJobEventsAsList()
	require.Equal(t, 2, len(jobList))

	marshalledInfo, normalErr := json.Marshal(&map[string]string{"info": "infoValues"})
	require.Nil(t, normalErr)
	j.AddInfo(marshalledInfo)

	require.Equal(t, 1, len(j.rootEvent.features))
	require.Equal(t, json.RawMessage(marshalledInfo), j.rootEvent.info)
	require.Equal(t, 1, len(j.rootEvent.children))
	require.Equal(t, 1, len(j.rootEvent.augmentedAndContentStreams))
}

func TestUploadingBinaries(t *testing.T) {
	j := createDummyJob(t)
	defer j.Close()
	uploadedStreams, err := j.uploadChildAndAugmentedBinaries(j.getAllJobEventsAsList())
	require.Nil(t, err)
	require.Equal(t, 0, len(uploadedStreams))

	j.AddChildBytes([]byte("abcdef child binary!"), map[string]string{"child": "extracted"})
	j.AddAugmentedBytes([]byte("Augmented stream woo!"), events.DataLabelTest)

	uploadedStreams, err = j.uploadChildAndAugmentedBinaries(j.getAllJobEventsAsList())
	require.Nil(t, err)
	require.Equal(t, 2, len(uploadedStreams))
	t.Logf("%+v", uploadedStreams)
	require.Equal(t, uploadedStreams["1c24fd61e61c98f2788ea3b6f19d330079ded26eb6471b330fc58ee4aad48cd9-content"].Sha256, "1c24fd61e61c98f2788ea3b6f19d330079ded26eb6471b330fc58ee4aad48cd9")
	require.Equal(t, uploadedStreams["3da92e003ce48a79c64d2befab9289633f92e5a3b6f2ed66a3caacf60c429e8c-test"].Sha256, "3da92e003ce48a79c64d2befab9289633f92e5a3b6f2ed66a3caacf60c429e8c")
}

func TestFeatureValidation(t *testing.T) {
	// Standard Feature should just work.
	j := createDummyJob(t)
	defer j.Close()
	err := j.AddFeature("dummy", "value1")
	require.Nil(t, err)
	pluginError := j.validateAndMutateFeatures(j.rootEvent, &defaults)
	require.Nil(t, pluginError)

	// Add an unregistered feature
	err = j.AddFeature("notreal", "garbage")
	require.Nil(t, err)
	pluginError = j.validateAndMutateFeatures(j.rootEvent, &defaults)
	require.NotNil(t, pluginError)
	require.Equal(t, ErrorsOutput, pluginError.innerError)
	require.Equal(t, "Plugin Unregistered Features", pluginError.errorTitle)

	// Add a feature of each type that is valid

	invalidFeatChecks := func(t *testing.T, pluginError *PluginError) {
		require.NotNil(t, pluginError)
		require.Equal(t, ErrorRunner, pluginError.innerError)
		require.Equal(t, "Feature Error", pluginError.errorTitle)
	}
	// Add a feature with an invalid type (integer) FeatureInteger
	j = createDummyJob(t)
	defer j.Close()
	err = j.AddFeature("dummyInt", "notGoodType")
	require.Nil(t, err)
	pluginError = j.validateAndMutateFeatures(j.rootEvent, &defaults)
	invalidFeatChecks(t, pluginError)

	// Add a feature with an invalid type (float) FeatureFloat
	j = createDummyJob(t)
	defer j.Close()
	err = j.AddFeature("dummyFloat", "notGoodType")
	require.Nil(t, err)
	pluginError = j.validateAndMutateFeatures(j.rootEvent, &defaults)
	invalidFeatChecks(t, pluginError)

	// Add a feature with an invalid type (date) FeatureDatetime
	j = createDummyJob(t)
	defer j.Close()
	err = j.AddFeature("dummyDate", "notGoodType")
	require.Nil(t, err)
	pluginError = j.validateAndMutateFeatures(j.rootEvent, &defaults)
	invalidFeatChecks(t, pluginError)
}

func TestFeatureWithLabel(t *testing.T) {
	// Standard Feature should just work.
	j := createDummyJob(t)
	defer j.Close()
	err := j.AddFeatureWithExtra("dummy", "value1", &AddFeatureOptions{Label: "dummyLabel"})
	require.Nil(t, err)
	pluginError := j.validateAndMutateFeatures(j.rootEvent, &defaults)
	require.Nil(t, pluginError)
	require.Equal(t, j.rootEvent.features[0], events.BinaryEntityFeature{Name: "dummy", Value: "value1", Type: events.FeatureString, Label: "dummyLabel", Offset: 0, Size: 0})

	// Standard Feature should just work.
	j = createDummyJob(t)
	defer j.Close()
	err = j.AddFeatureWithExtra("dummy", "value2", &AddFeatureOptions{Offset: 5678})
	require.Nil(t, err)
	pluginError = j.validateAndMutateFeatures(j.rootEvent, &defaults)
	require.Nil(t, pluginError)
	require.Equal(t, j.rootEvent.features[0], events.BinaryEntityFeature{Name: "dummy", Value: "value2", Type: events.FeatureString, Label: "", Offset: 5678, Size: 0})

	// Standard Feature should just work.
	j = createDummyJob(t)
	defer j.Close()
	err = j.AddFeatureWithExtra("dummy", "value3", &AddFeatureOptions{Size: 9999})
	require.Nil(t, err)
	pluginError = j.validateAndMutateFeatures(j.rootEvent, &defaults)
	require.Nil(t, pluginError)
	require.Equal(t, j.rootEvent.features[0], events.BinaryEntityFeature{Name: "dummy", Value: "value3", Type: events.FeatureString, Label: "", Offset: 0, Size: 9999})

	// Standard Feature should just work.
	j = createDummyJob(t)
	defer j.Close()
	err = j.AddFeatureWithExtra("dummy", "value4", &AddFeatureOptions{Label: "dummyLabel", Offset: 123, Size: 9000})
	require.Nil(t, err)
	pluginError = j.validateAndMutateFeatures(j.rootEvent, &defaults)
	require.Nil(t, pluginError)
	require.Equal(t, j.rootEvent.features[0], events.BinaryEntityFeature{Name: "dummy", Value: "value4", Type: events.FeatureString, Label: "dummyLabel", Offset: 123, Size: 9000})
}

func TestGenEventResultStatusAndMessage(t *testing.T) {
	j := createDummyJob(t)
	defer j.Close()
	// Opt out case
	pluginError := NewPluginError(OptOutError, "", "opt-out reason")
	status := j.getEventResultStatus(pluginError)
	message := j.getEventResultMessage(pluginError)
	assert.Equal(t, events.StatusTypeOptOut, status)
	assert.Equal(t, "opt-out reason", message)

	// Opt out case convenience method.
	pluginError = NewPluginOptOut("opt-out reason")
	status = j.getEventResultStatus(pluginError)
	message = j.getEventResultMessage(pluginError)
	assert.Equal(t, events.StatusTypeOptOut, status)
	assert.Equal(t, "opt-out reason", message)

	// Exception Case
	pluginError = NewPluginError(ErrorException, "errortitle", "error")
	status = j.getEventResultStatus(pluginError)
	message = j.getEventResultMessage(pluginError)
	assert.Equal(t, events.StatusTypeErrorException, status)
	assert.Equal(t, "error", message)

	// Completed no features
	status = j.getEventResultStatus(nil)
	message = j.getEventResultMessage(nil)
	assert.Equal(t, events.StatusTypeCompletedEmpty, status)
	assert.Equal(t, "", message)

	// Completed with features
	err := j.AddFeature("dummy", "value1")
	require.Nil(t, err)
	status = j.getEventResultStatus(nil)
	message = j.getEventResultMessage(nil)
	assert.Equal(t, events.StatusTypeCompleted, status)
	assert.Equal(t, "", message)
}

// Add a child binary and ensure it is deleted at job close.
func TestJobClose(t *testing.T) {
	j := createDummyJob(t)
	path := utilWriteToTempFile(t, []byte("abcdef child binary!"))
	defer os.Remove(path)
	j.AddChild(path, map[string]string{"child": "extracted"})
	fileInfo, err := os.ReadDir(j.temporaryDirPath)
	require.Nil(t, err)
	require.Equal(t, 1, len(fileInfo), "Expected one child file to have been created.")
	// Close file and ensure it no longer exists.
	j.Close()
	_, err = os.Stat(j.temporaryDirPath)
	require.NotNil(t, err)
	// Ensure there are no orphaned files outside of temporaryDirPath
	for _, curChild := range j.rootEvent.children {
		for _, curChildStream := range curChild.augmentedAndContentStreams {
			_, err = os.Stat(curChildStream.path)
			require.NotNil(t, err, "Child stream wasn't deleted %v", curChildStream.Sha256)
		}

	}
}

// Add child and augmented files at multiple level and ensure they are deleted when the job closes.
func TestJobCloseComplex(t *testing.T) {
	j := createDummyJob(t)
	pathChild1 := utilWriteToTempFile(t, []byte("pathChild1 abcdef child binary!"))
	defer os.Remove(pathChild1)
	pathChild2 := utilWriteToTempFile(t, []byte("pathChild2 abcdef child binary!"))
	defer os.Remove(pathChild2)
	pathAug1 := utilWriteToTempFile(t, []byte("pathAug1 abcdef child binary!"))
	defer os.Remove(pathAug1)
	pathAug1ForChild1 := utilWriteToTempFile(t, []byte("pathAug1ForChild1 abcdef child binary!"))
	defer os.Remove(pathAug1ForChild1)

	_, err := j.AddChild(pathChild1, map[string]string{"child": "extracted"})
	require.Nil(t, err)
	err = j.AddAugmented(pathAug1, events.DataLabelTest)
	require.Nil(t, err)
	childEvent, err := j.AddChild(pathChild2, map[string]string{"child": "extracted"})
	require.Nil(t, err)
	err = childEvent.AddAugmented(pathAug1ForChild1, events.DataLabelTest)
	require.Nil(t, err)

	fileInfo, err := os.ReadDir(j.temporaryDirPath)
	require.Nil(t, err)
	require.Equal(t, 4, len(fileInfo), "Expected 2 children and 2 augmented files but not all are present.")
	// Close file and ensure it no longer exists.
	j.Close()
	_, err = os.Stat(j.temporaryDirPath)
	require.NotNil(t, err)
	// Ensure there are no orphaned files outside of temporaryDirPath
	for _, curAug := range j.rootEvent.augmentedAndContentStreams {
		_, err = os.Stat(curAug.path)
		require.NotNil(t, err, "Root Augmented stream wasn't deleted %v", curAug.Sha256)
	}

	for _, curChild := range j.rootEvent.children {
		for _, curChildStream := range curChild.augmentedAndContentStreams {
			_, err = os.Stat(curChildStream.path)
			require.NotNil(t, err, "Child or Augmented stream wasn't deleted %v", curChildStream.Sha256)
		}
	}
}

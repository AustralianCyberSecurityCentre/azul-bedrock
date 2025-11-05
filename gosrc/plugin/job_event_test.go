package plugin

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/events"
	"github.com/stretchr/testify/require"
)

/*Basic suite of tests for functionallity of plugin code.*/

func utilGetJob(t *testing.T) (*JobEvent, string) {
	tempDir, err := os.MkdirTemp("", "gorunner")
	require.Nil(t, err)
	jobEvent := NewJobEvent(tempDir, &JobEvent{})
	return jobEvent, tempDir
}

/*write bytes to a temp file and give back the file Name()*/
func utilWriteToTempFile(t *testing.T, rawBytes []byte) string {
	tempFile, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatalf("Failed test when attempting to create tempfile with error %v", err)
	}
	tempFile.Write(rawBytes)
	tempFile.Close()
	return tempFile.Name()
}

func TestSha256Calculations(t *testing.T) {
	rawBytes := []byte("jdcaojinvjiads jfcasdkfasdjfhasdkjf ahskjdfkjasdhf")
	path := utilWriteToTempFile(t, rawBytes)
	defer os.Remove(path)

	sha256A := calculateSha256OfBytes(rawBytes)
	sha256B, sizeB, errB := calculateSha256OfFile(path)
	require.Nil(t, errB)
	require.Equal(t, uint64(len(rawBytes)), sizeB)
	require.Equal(t, sha256A, sha256B)
	require.Equal(t, "9f2ec83668a30cb517c7d137c7d3036a47733a583ddd6a89d68b6fb5157f8fda", sha256A)
}

func TestAddFeature(t *testing.T) {
	job, tempDir := utilGetJob(t)
	defer os.RemoveAll(tempDir)
	err := job.AddFeature("dummy1", "dummy2")
	require.Nil(t, err)
	require.Equal(t, 1, len(job.features))

	// Should error with an unknown type.
	err = job.AddFeature("dummy1", map[string]string{"abc": "def"})
	require.NotNil(t, err)
}

func TestAddFeatureExtended(t *testing.T) {
	job, tempDir := utilGetJob(t)
	// String
	defer os.RemoveAll(tempDir)
	err := job.AddFeature("dummy", "dummy2")
	require.Equal(t, "dummy2", job.features[len(job.features)-1].Value)
	// Date
	require.Nil(t, err)
	err = job.AddFeature("dummyDate", time.Time{}.UTC())
	require.Equal(t, "0001-01-01T00:00:00Z", job.features[len(job.features)-1].Value)
	// Byte array
	require.Nil(t, err)
	err = job.AddFeature("dummyByte", []byte("dummy value for features"))
	require.Equal(t, "dummy value for features", job.features[len(job.features)-1].Value)
	require.Nil(t, err)
	// Floats
	err = job.AddFeature("dummyFloat", float32(1.02))
	require.Equal(t, "1.0199999809265137", job.features[len(job.features)-1].Value)
	require.Nil(t, err)
	err = job.AddFeature("dummyFloat", float64(1.02342342))
	require.Equal(t, "1.02342342", job.features[len(job.features)-1].Value)
	require.Nil(t, err)
	// Integers
	err = job.AddFeature("dummyInt", uint64(2))
	require.Equal(t, "2", job.features[len(job.features)-1].Value)
	require.Nil(t, err)
	err = job.AddFeature("dummyInt", int64(2))
	require.Equal(t, "2", job.features[len(job.features)-1].Value)
	require.Nil(t, err)
	err = job.AddFeature("dummyInt", int(2))
	require.Equal(t, "2", job.features[len(job.features)-1].Value)
	require.Nil(t, err)
	err = job.AddFeature("dummyInt", int8(2))
	require.Equal(t, "2", job.features[len(job.features)-1].Value)
	require.Nil(t, err)

	require.Equal(t, 9, len(job.features))
}

func TestAddInfo(t *testing.T) {
	job, tempDir := utilGetJob(t)
	defer os.RemoveAll(tempDir)
	marshalledinfo, err := json.Marshal(&map[string]string{"info": "infoValues"})
	require.Nil(t, err)
	job.AddInfo(marshalledinfo)
	require.Equal(t, json.RawMessage(marshalledinfo), job.info)
}

func TestAddAugmentedBoth(t *testing.T) {
	job, tempDir := utilGetJob(t)
	defer os.RemoveAll(tempDir)
	fileBytes := []byte("augmented file raw bytes!")
	path := utilWriteToTempFile(t, fileBytes)
	defer os.Remove(path)

	job.AddAugmented(path, events.DataLabelTest)
	require.Equal(t, 1, len(job.augmentedAndContentStreams))

	job.AddAugmentedBytes([]byte("different augmented file bytes"), events.DataLabelText)
	require.Equal(t, 2, len(job.augmentedAndContentStreams))
}

// Ensure when you add a file by path the file exists in the job after the original file is destroyed.
func TestCopyFilePath(t *testing.T) {
	job, tempDir := utilGetJob(t)
	defer os.RemoveAll(tempDir)
	// Add file as augmented
	augmentedFileBytes := []byte("augmented file raw bytes!")
	path := utilWriteToTempFile(t, augmentedFileBytes)
	job.AddAugmented(path, events.DataLabelTest)
	os.Remove(path)

	require.Equal(t, 1, len(job.augmentedAndContentStreams))
	// Just getting the first stream
	for _, augFileRef := range job.augmentedAndContentStreams {
		readFileBytes, err := os.ReadFile(augFileRef.path)
		require.Nil(t, err)
		require.Equal(t, augmentedFileBytes, readFileBytes)
	}

	// Add file for child binary as well
	childFileBytes := []byte("child file raw bytessss")
	childPath := utilWriteToTempFile(t, childFileBytes)
	je, err := job.AddChild(childPath, map[string]string{"action": "extracted"})
	// Remove original file to ensure the file in the job isn't pointing to the original file
	os.Remove(childPath)
	require.Nil(t, err)

	require.Equal(t, len(je.augmentedAndContentStreams), 1)
	for _, childBinRef := range je.augmentedAndContentStreams {
		require.Equal(t, events.DataLabelContent, childBinRef.Label)
		readFileBytes, err := os.ReadFile(childBinRef.path)
		require.Nil(t, err)
		require.Equal(t, childFileBytes, readFileBytes)
	}
}

func TestAddChildBoth(t *testing.T) {
	job, tempDir := utilGetJob(t)
	defer os.RemoveAll(tempDir)
	fileBytes := []byte("child file raw bytes!")
	path := utilWriteToTempFile(t, fileBytes)
	defer os.Remove(path)

	childJob, err := job.AddChild(path, map[string]string{"child": "extractedFile"})
	require.Nil(t, err)
	require.Equal(t, 1, len(childJob.augmentedAndContentStreams))
	require.Equal(t, 1, len(job.children))

	childJob = job.AddChildBytes(fileBytes, map[string]string{"child": "extractedBytes"})
	require.Equal(t, 1, len(childJob.augmentedAndContentStreams))
	require.Equal(t, 2, len(job.children))

	childJob.AddAugmentedBytes([]byte("augmented file attached to child"), events.DataLabelTest)
	require.Equal(t, 2, len(childJob.augmentedAndContentStreams))
}

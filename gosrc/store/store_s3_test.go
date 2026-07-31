//go:build integration

package store

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v12/gosrc/models"
	st "github.com/AustralianCyberSecurityCentre/azul-bedrock/v12/gosrc/settings"
	testdata "github.com/AustralianCyberSecurityCentre/azul-bedrock/v12/gosrc/testdata"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreS3(t *testing.T) {
	s3Store, err := NewS3Store(
		st.TestSettings.Streams.S3.Endpoint,
		st.TestSettings.Streams.S3.AccessKey,
		st.TestSettings.Streams.S3.SecretKey,
		st.TestSettings.Streams.S3.Secure,
		st.TestSettings.Streams.S3.Bucket,
		st.TestSettings.Streams.S3.Region,
		nil,
		AutomaticAgeOffSettings{EnableAutomaticAgeOff: false},
	)
	require.NoError(t, err)

	StoreImplementationBaseTests(t, s3Store)
	StoreImplementationListBaseTests(t, s3Store)
}

func TestStoreS3WithCache(t *testing.T) {
	s3Store, err := NewS3Store(
		st.TestSettings.Streams.S3.Endpoint,
		st.TestSettings.Streams.S3.AccessKey,
		st.TestSettings.Streams.S3.SecretKey,
		st.TestSettings.Streams.S3.Secure,
		st.TestSettings.Streams.S3.Bucket,
		st.TestSettings.Streams.S3.Region,
		nil,
		AutomaticAgeOffSettings{EnableAutomaticAgeOff: false},
	)
	require.NoError(t, err)
	// Ensure max file size stored is 2kb.
	cacheStore, err := NewDataCache(1, 300, 256, s3Store, StoreCacheMetricCollectors{})
	require.NoError(t, err, "Error creating LocalStore Cache")

	StoreImplementationBaseTests(t, cacheStore)
	StoreImplementationListBaseTests(t, cacheStore)
}

// Verify the automatic age off policy is created.
// Due to timings it's impossible to verify the policy itself.
func TestAutoAgeOff(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	data := string(testdata.GetBytes("sources/sources1.yaml"))
	sourceConf, err := models.ParseSourcesYaml(data)
	require.Nil(t, err)
	// There should be at lest 5 sources
	NUMBER_OF_SOURCES := len(sourceConf.Sources)
	require.GreaterOrEqual(t, NUMBER_OF_SOURCES, 5)
	// Standard no policy S3 Store (enable auto-cleanup just in case.)
	s3Store, err := NewS3Store(
		st.TestSettings.Streams.S3.Endpoint,
		st.TestSettings.Streams.S3.AccessKey,
		st.TestSettings.Streams.S3.SecretKey,
		st.TestSettings.Streams.S3.Secure,
		st.TestSettings.Streams.S3.Bucket,
		st.TestSettings.Streams.S3.Region,
		nil,
		AutomaticAgeOffSettings{EnableAutomaticAgeOff: false, EnableCleanupAutoAgeOff: true, SourceConf: &sourceConf},
	)
	minioClient := s3Store.(*StoreS3).client
	bucketLifeCycle, err := minioClient.GetBucketLifecycle(ctx, s3Store.(*StoreS3).bucket)
	// error means the policy doesn't exist so assume there are 0 rules to begin with.
	baseRuleCount := 0
	if err == nil {
		baseRuleCount = len(bucketLifeCycle.Rules)
	}

	// Setup auto-ageoff rules for S3 Store
	s3Store, err = NewS3Store(
		st.TestSettings.Streams.S3.Endpoint,
		st.TestSettings.Streams.S3.AccessKey,
		st.TestSettings.Streams.S3.SecretKey,
		st.TestSettings.Streams.S3.Secure,
		st.TestSettings.Streams.S3.Bucket,
		st.TestSettings.Streams.S3.Region,
		nil,
		AutomaticAgeOffSettings{EnableAutomaticAgeOff: true, SourceConf: &sourceConf},
	)
	require.NoError(t, err)
	minioClient = s3Store.(*StoreS3).client
	bucketLifeCycle, err = minioClient.GetBucketLifecycle(ctx, s3Store.(*StoreS3).bucket)
	require.Nil(t, err)
	ruleCountAfterAutoCreation := len(bucketLifeCycle.Rules)

	// Cleanup the auto-created rules.
	s3Store, err = NewS3Store(
		st.TestSettings.Streams.S3.Endpoint,
		st.TestSettings.Streams.S3.AccessKey,
		st.TestSettings.Streams.S3.SecretKey,
		st.TestSettings.Streams.S3.Secure,
		st.TestSettings.Streams.S3.Bucket,
		st.TestSettings.Streams.S3.Region,
		nil,
		AutomaticAgeOffSettings{EnableAutomaticAgeOff: false, EnableCleanupAutoAgeOff: true, SourceConf: &sourceConf},
	)
	require.NoError(t, err)
	minioClient = s3Store.(*StoreS3).client
	bucketLifeCycle, err = minioClient.GetBucketLifecycle(ctx, s3Store.(*StoreS3).bucket)
	var minioError minio.ErrorResponse
	ruleCountAfterCleanup := 0
	// Case where there are no rules with the bucket requires that specific error.
	if err != nil {
		errorIsMinioError := errors.As(err, &minioError)
		require.True(t, errorIsMinioError)
		require.Equal(t, "NoSuchLifecycleConfiguration", minioError.Code)
		ruleCountAfterCleanup = 0
	} else {
		// If there is no error take the length of the rules.
		ruleCountAfterCleanup = len(bucketLifeCycle.Rules)
	}
	assert.Equal(t, baseRuleCount, ruleCountAfterCleanup, "The number of rules before and after cleanup should be the same.")
	assert.Greater(t, ruleCountAfterAutoCreation, baseRuleCount, "There should be at least a rule created by auto-creation")
	assert.Equal(t, ruleCountAfterAutoCreation, baseRuleCount+NUMBER_OF_SOURCES, "The expected number of rules to be created is equal to the number of sources.")
}

func TestS3WithAesChoppyBuffer(t *testing.T) {
	/* Verifies that reading and writing from a buffer that gives content in a random order works.

	e.g the buffer may give the first 10 bytes then the next 12 etc.
	This can occur in production in buffering and streaming situations where not all bytes can be immediately provided.

	Regression test due to issue that occurred when reading from Piped Gzip content that was providing bytes in a choppy manner.
	This caused AES corruption.

	Note this is done with S3 and localstorage as their read behaviour is different.
	*/
	assert := assert.New(t)

	s3Store, err := NewS3Store(
		st.TestSettings.Streams.S3.Endpoint,
		st.TestSettings.Streams.S3.AccessKey,
		st.TestSettings.Streams.S3.SecretKey,
		st.TestSettings.Streams.S3.Secure,
		st.TestSettings.Streams.S3.Bucket,
		st.TestSettings.Streams.S3.Region,
		nil,
		AutomaticAgeOffSettings{EnableAutomaticAgeOff: false, EnableCleanupAutoAgeOff: true},
	)

	aesCtrStore := NewAESCtrStore(s3Store, aesDummyKey, true)

	var probMalware = []byte("Hello, this is malware! held in a choppy buffer that will come through at an odd pace.")
	// Convert raw bytes to reader
	reader := bytes.NewReader(probMalware)
	// Choppy reader that provides bytes to a choppy fashion into AES.
	choppyReader := CustomChoppyByteReader{
		innerReader:    reader,
		lastReadLength: 1,
	}
	readCloser := io.NopCloser(&choppyReader)

	err = aesCtrStore.Put("testsource", "testlabel", "aesctredfile", readCloser, -1)
	require.NoError(t, err, "Error writing to AES_CTR store", err)

	testData, err := aesCtrStore.Fetch("testsource", "testlabel", "aesctredfile", WithOffsetAndSize(0, -1))
	require.NoError(t, err, "Error reading from AES_CTR store", err)

	readBuffer := getDataSliceBytesInterfaceTest(t, testData)
	assert.Equal(probMalware, readBuffer, "Bad encoding has occurred and AES encryption is corrupting files.")

	// Second fetch to verify that choppy reads don't prevent the read by corrupting the file during read.
	testData, err = aesCtrStore.Fetch("testsource", "testlabel", "aesctredfile", WithOffsetAndSize(0, -1))
	require.NoError(t, err, "Error reading from AES_CTR store", err)

	byteBufferLargerThanContent := make([]byte, len(probMalware)*4)
	readBytes, err := testData.DataReader.Read(byteBufferLargerThanContent)
	require.Equal(t, err, io.EOF)
	assert.Equal(probMalware, byteBufferLargerThanContent[:readBytes])

	// confirm EOF with 0 bytes read is returned on subsequent reads.
	readBytes, err = testData.DataReader.Read(byteBufferLargerThanContent)
	require.Equal(t, err, io.EOF)
	require.Equal(t, 0, readBytes)
}

//go:build integration

package store

import (
	"context"
	"errors"
	"testing"

	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/models"
	st "github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/settings"
	testdata "github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/testdata"
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

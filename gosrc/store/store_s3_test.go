//go:build integration

package store

import (
	"testing"

	"github.com/stretchr/testify/require"

	st "github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/settings"
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
}

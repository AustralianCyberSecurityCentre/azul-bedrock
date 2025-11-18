//go:build integration_azure

package store

import (
	"testing"

	st "github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/settings"
	"github.com/stretchr/testify/require"
)

func TestAzureStore(t *testing.T) {
	azureStore, err := NewAzureStore(st.TestSettings.Streams.Azure.Endpoint, st.TestSettings.Streams.Azure.Container, st.TestSettings.Streams.Azure.StorageAccount, st.TestSettings.Streams.Azure.AccessKey, nil)
	require.NoError(t, err)

	StoreImplementationBaseTests(t, azureStore)
	StoreImplementationListBaseTests(t, azureStore)
}

func TestAzureStoreWithCache(t *testing.T) {
	azureStore, err := NewAzureStore(st.TestSettings.Streams.Azure.Endpoint, st.TestSettings.Streams.Azure.Container, st.TestSettings.Streams.Azure.StorageAccount, st.TestSettings.Streams.Azure.AccessKey, nil)
	require.NoError(t, err)

	// Ensure max file size stored is 2kb.
	cacheStore, err := NewDataCache(1, 300, 256, azureStore, StoreCacheMetricCollectors{})
	require.NoError(t, err, "Error creating LocalStore Cache")

	StoreImplementationBaseTests(t, cacheStore)
	StoreImplementationListBaseTests(t, cacheStore)
}

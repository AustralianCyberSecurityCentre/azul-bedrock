package store

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocalStore(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "test-bedrock-store")
	defer os.RemoveAll(dir)
	require.NoError(t, err, "Error creating temp dir", err)

	store, err := NewEmptyLocalStore(dir)
	require.NoError(t, err, "Error creating local store", err)

	StoreImplementationBaseTests(t, store)
	StoreImplementationListBaseTests(t, store)
}

func BenchmarkLocalReadStore(b *testing.B) {
	dir, err := os.MkdirTemp("/tmp", "test-bedrock-store")
	defer os.RemoveAll(dir)
	if err != nil {
		b.Error("Error creating temp dir", err)
	}
	store, err := NewEmptyLocalStore(dir)
	if err != nil {
		b.Error("Error creating LocalStore", err)
	}

	BaseBenchmarkReadStore(b, store)
}

func BenchmarkLocalWriteStore(b *testing.B) {
	dir, err := os.MkdirTemp("/tmp", "test-bedrock-store")
	defer os.RemoveAll(dir)
	if err != nil {
		b.Error("Error creating temp dir", err)
	}
	store, err := NewEmptyLocalStore(dir)
	if err != nil {
		b.Error("Error creating LocalStore", err)
	}

	BaseBenchmarkWriteStore(b, store)
}

package indexstorage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v13/gosrc/store"
)

func TestLocalStoreRoundTripAndCacheRepair(t *testing.T) {
	root := t.TempDir()
	storage, err := store.NewLocalStore(filepath.Join(root, "object-store"))
	if err != nil {
		t.Fatal(err)
	}
	m := Manager{Storage: storage}
	input := filepath.Join(root, "test.bgi")
	original := bytes.Repeat([]byte("abcxyz0123456789"), 1024)
	if err := os.WriteFile(input, original, 0600); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(original)
	expected := hex.EncodeToString(hash[:])
	id, err := m.Put(input)
	if err != nil {
		t.Fatal(err)
	}
	if id != expected {
		t.Fatalf("ID %s != %s", id, expected)
	}
	// Repeated publication of a content-addressed object is safe.
	if _, err := m.Put(input); err != nil {
		t.Fatal(err)
	}
	ids, err := m.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != id {
		t.Fatalf("unexpected published IDs: %v", ids)
	}
	output := filepath.Join(root, "cache", id+".bgi")
	if err := m.Fetch(id, output); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := m.Fetch(id, output); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, original) {
		t.Fatal("cache repair did not restore data")
	}
}

func TestUnpublishedIndexCannotBeFetched(t *testing.T) {
	storage, err := store.NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m := Manager{Storage: storage}
	data := []byte("incomplete")
	hash := sha256.Sum256(data)
	id := hex.EncodeToString(hash[:])
	if err := storage.Put(Source, Label, id+".bgi", ioNopCloser(data), int64(len(data))); err != nil {
		t.Fatal(err)
	}
	ids, err := m.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 0 {
		t.Fatalf("unpublished ID visible: %v", ids)
	}
	if err := m.Fetch(id, filepath.Join(t.TempDir(), "output.bgi")); err == nil {
		t.Fatal("unpublished fetch should fail")
	}
}

func TestConcurrentFetchSameDestination(t *testing.T) {
	storage, err := store.NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m := Manager{Storage: storage}
	input := filepath.Join(t.TempDir(), "input.bgi")
	if err := os.WriteFile(input, bytes.Repeat([]byte("z"), 8192), 0600); err != nil {
		t.Fatal(err)
	}
	id, err := m.Put(input)
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "index.bgi")
	var wg sync.WaitGroup
	failures := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); failures <- m.Fetch(id, output) }()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	if digest, _, err := HashFile(output); err != nil || digest != id {
		t.Fatalf("output corrupted: %s %v", digest, err)
	}
}

func TestRejectPathTraversal(t *testing.T) {
	for _, s := range []string{"../foo", "ABC", strings.Repeat("a", 63)} {
		if checkID(s) == nil {
			t.Fatalf("invalid ID accepted: %s", s)
		}
	}
}

func ioNopCloser(b []byte) *nopCloser { return &nopCloser{bytes.NewReader(b)} }

type nopCloser struct{ *bytes.Reader }

func (*nopCloser) Close() error { return nil }

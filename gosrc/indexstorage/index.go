// Package indexstorage publishes and fetches immutable BigGrep index files.
//
// BigGrep requires a local .bgi file for mmap-based searching. This package
// deliberately keeps object storage and caching outside the BigGrep engine.
package indexstorage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"

	bedSet "github.com/AustralianCyberSecurityCentre/azul-bedrock/v13/gosrc/settings"
	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v13/gosrc/store"
)

const (
	// Source and Label form the shared object namespace: retrohunt/indices/.
	Source       = "retrohunt"
	Label        = "indices"
	markerSuffix = ".ready"
)

var validID = regexp.MustCompile(`^[a-f0-9]{64}$`)

// Manager stores content-addressed indexes using Bedrock's existing FileStorage
// interface. IDs are SHA-256 digests, so publishing is safe to retry and
// readers can verify downloaded files without a separate metadata service.
type Manager struct{ Storage store.FileStorage }

func checkID(id string) error {
	if !validID.MatchString(id) {
		return fmt.Errorf("invalid index ID %q: expected 64 lowercase hexadecimal characters", id)
	}
	return nil
}

// HashFile streams the input file. It never reads the whole index into RAM.
func HashFile(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	h := sha256.New()
	n, err := io.Copy(h, file)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

// Put publishes an index in two stages: the content object first, then a tiny
// ready marker. List ignores uncommitted content objects. BigGrep can never
// discover an index until its upload has completed successfully.
//
// Index objects are named by their SHA-256 digest rather than a mutable path.
// This protects running hunts from a writer replacing an index underneath them.
func (m Manager) Put(path string) (string, error) {
	id, size, err := HashFile(path)
	if err != nil {
		return "", fmt.Errorf("hash index: %w", err)
	}
	if size == 0 {
		return "", errors.New("refusing to publish an empty index")
	}
	if !strings.HasSuffix(strings.ToLower(path), ".bgi") {
		return "", errors.New("input must have a .bgi extension")
	}
	object := id + ".bgi"
	ready, err := m.Storage.Exists(Source, Label, id+markerSuffix)
	if err != nil {
		return "", fmt.Errorf("check publication: %w", err)
	}
	if ready {
		return id, nil
	} // A completed content-addressed upload is immutable.
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	// Pass the known file size so Bedrock's S3 implementation can use its
	// bounded-memory multipart uploader for large indexes.
	err = m.Storage.Put(Source, Label, object, file, size)
	closeErr := file.Close()
	if err != nil {
		return "", fmt.Errorf("upload index: %w", err)
	}
	if closeErr != nil {
		return "", closeErr
	}
	// The marker is small and only created after the object upload succeeded.
	// Readers additionally verify its content and the complete object digest.
	marker := io.NopCloser(strings.NewReader(id + "\n"))
	if err := m.Storage.Put(Source, Label, id+markerSuffix, marker, int64(len(id)+1)); err != nil {
		return "", fmt.Errorf("index uploaded but publication marker failed (retry is safe): %w", err)
	}
	return id, nil
}

// List returns published IDs. Object listing can be streamed by providers;
// only the short identifiers are accumulated here, not index contents.
func (m Manager) List(ctx context.Context) ([]string, error) {
	var result []string
	for entry := range m.Storage.List(ctx, Source+"/"+Label+"/", "") {
		if entry.Err != nil {
			return nil, entry.Err
		}
		if !strings.HasSuffix(entry.Key, markerSuffix) {
			continue
		}
		id := strings.TrimSuffix(filepath.Base(entry.Key), markerSuffix)
		if checkID(id) == nil {
			result = append(result, id)
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	sort.Strings(result)
	return result, nil
}

// Fetch atomically ensures a verified local index at output. An advisory
// filesystem lock serializes processes downloading the same index to the same
// path, preventing redundant network transfers and partially visible files.
// The lock file must live on a filesystem supporting flock (Linux pods do).
func (m Manager) Fetch(id, output string) error {
	if err := checkID(id); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0750); err != nil {
		return err
	}
	lock, err := os.OpenFile(output+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer func() {
		if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_UN); err != nil {
			bedSet.Logger.Error().
				Err(err).
				Msgf("failed to release lock file %s", output+".lock")
		}
	}()

	// Validate the cached copy before making network requests. This costs one
	// sequential disk scan, but avoids trusting a corrupt or truncated cache.
	if digest, size, err := HashFile(output); err == nil && size > 0 && digest == id {
		return nil
	}

	// Reject unpublished objects; a writer may still be uploading their data.
	marker, err := m.Storage.Fetch(Source, Label, id+markerSuffix)
	if err != nil {
		return fmt.Errorf("index %s is not published: %w", id, err)
	}
	markerBytes, readErr := io.ReadAll(io.LimitReader(marker.DataReader, 128))
	closeErr := marker.DataReader.Close()
	if readErr != nil {
		return readErr
	}
	if closeErr != nil {
		return closeErr
	}
	if strings.TrimSpace(string(markerBytes)) != id {
		return errors.New("invalid publication marker")
	}

	data, err := m.Storage.Fetch(Source, Label, id+".bgi")
	if err != nil {
		return fmt.Errorf("fetch index: %w", err)
	}
	defer data.DataReader.Close()
	if data.Avail <= 0 {
		return errors.New("empty index object")
	}
	// Write in the destination directory so rename is atomic on the same volume.
	tmp, err := os.CreateTemp(filepath.Dir(output), ".bgi-download-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	if err := tmp.Chmod(0640); err != nil {
		return err
	}
	h := sha256.New()
	copied, err := io.Copy(io.MultiWriter(tmp, h), data.DataReader)
	if err != nil {
		return fmt.Errorf("stream index: %w", err)
	}
	if copied != data.Avail {
		return fmt.Errorf("truncated download: got %d bytes, expected %d", copied, data.Avail)
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != id {
		return fmt.Errorf("index checksum mismatch: expected %s, got %s", id, got)
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), output); err != nil {
		return err
	}
	return nil
}

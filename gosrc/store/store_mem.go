package store

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
)

/* In memory version of store for testing, storage of files into an s3 provider. */
type StoreMem struct {
	Data map[string][]byte
	mu   sync.Mutex
}

func NewStoreMem() *StoreMem {
	return &StoreMem{
		Data: map[string][]byte{},
	}
}

func (s *StoreMem) Put(source, label, id string, data io.ReadCloser, fileSize int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rawData, err := io.ReadAll(data)
	if err != nil {
		return err
	}
	s.Data[createIdPath(source, label, id)] = rawData
	return nil
}

func (s *StoreMem) Fetch(source, label, id string, opts ...FileStorageFetchOption) (DataSlice, error) {
	key := createIdPath(source, label, id)
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.Data[key]
	if !ok {
		return NewDataSlice(), fmt.Errorf("key %s not found, %w", key, &NotFoundError{})
	}

	fetchOpt := NewFileStorageFetchOptions(opts...)
	empty := NewDataSlice()

	actualSize := int64(len(data))
	// -ve or zero is read all
	if fetchOpt.Size <= 0 {
		fetchOpt.Size = actualSize
	}
	// treat -ve as relative to end
	if fetchOpt.Offset < 0 {
		fetchOpt.Offset = actualSize + fetchOpt.Offset
	}
	// still -ve (-ve offset was bigger than file)
	if fetchOpt.Offset < 0 {
		fetchOpt.Offset = 0
	}
	// offset after end of file
	if fetchOpt.Offset > 0 && fetchOpt.Offset >= actualSize {
		return empty, &OffsetAfterEnd{msg: fmt.Sprintf("offset after EOF: %d", actualSize)}
	}
	// requested more than available
	if fetchOpt.Offset+fetchOpt.Size > actualSize {
		fetchOpt.Size = actualSize - fetchOpt.Offset
	}

	reader := bytes.NewReader(data)

	_, err := reader.Seek(fetchOpt.Offset, 0)
	if err != nil {
		return empty, fmt.Errorf("%w", &ReadError{msg: fmt.Sprintf("%v", err)})
	}
	// Limit the reader so gin doesn't read beyond the selected file size.
	wrappedLimitedReader := io.NopCloser(io.LimitReader(reader, fetchOpt.Size))

	return DataSlice{DataReader: wrappedLimitedReader, Start: fetchOpt.Offset, Size: fetchOpt.Size, Avail: actualSize}, nil
}

func (s *StoreMem) Exists(source, label, id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.Data[createIdPath(source, label, id)]
	return ok, nil
}

func (s *StoreMem) Delete(source, label, id string, opts ...FileStorageDeleteOption) (bool, error) {
	key := createIdPath(source, label, id)

	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.Data[key]
	if !ok {
		return ok, fmt.Errorf("key %s not found, %w", key, &NotFoundError{})
	}
	delete(s.Data, key)
	return ok, nil
}

func (s *StoreMem) Copy(sourceOld, labelOld, idOld, sourceNew, labelNew, idNew string) error {
	data, err := s.Fetch(sourceOld, labelOld, idOld)
	if err != nil {
		return err
	}
	newKey := createIdPath(sourceNew, labelNew, idNew)
	s.mu.Lock()
	defer s.mu.Unlock()
	rawData, err := io.ReadAll(data.DataReader)
	if err != nil {
		return err
	}
	s.Data[newKey] = rawData
	return nil
}

/*List the contents of the current S3 Bucket.*/
func (s *StoreMem) List(ctx context.Context, prefix string, startAfter string) <-chan FileStorageObjectListInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	// implement minimum needed for testing
	out := make(chan FileStorageObjectListInfo, 1000)

	// sort alphabetically
	keys := []string{}
	for k := range s.Data {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	hitStartAfter := false

	for _, key := range keys {
		if len(startAfter) > 0 && !hitStartAfter {
			if key == startAfter {
				hitStartAfter = true
			}
			continue
		}
		if len(prefix) > 0 && !strings.HasPrefix(key, prefix) {
			continue
		}
		source, label, id := splitIdPath(key)
		out <- FileStorageObjectListInfo{Key: key, Source: source, Label: label, Id: id}
	}

	close(out)
	return out
}

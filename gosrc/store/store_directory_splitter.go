package store

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
)

const NumberOfChars = 3
const Separator = "/"

/*
DirectorySplitterStore modifies storage path for files to add an extra directory into the directory structure.
This is done to allow for better storage in directory based storage repositories.
The change in the storage looks something like this: <source>/<label>/<sha256>
to this: <source>/<label>/<ab>/<sha256>
where ab is the first two characters of the sha256.
*/
type DirectorySplitterStore struct {
	inner FileStorage
}

// NewFolderPrefixStore wraps inner so that every operation is folded under folder.
// If folder is empty the inner store is returned unchanged.
func NewDirectorySplitterStore(inner FileStorage) FileStorage {
	return &DirectorySplitterStore{inner: inner}
}

// The key is "<source>/<label>/<id>", so prepending s.prefix to source prefixes
// the whole key with the folder.
func (s *DirectorySplitterStore) Put(source, label, id string, data io.ReadCloser, fileSize int64) error {
	newId, err := formatId(id)
	if err != nil {
		return err
	}
	return s.inner.Put(source, label, newId, data, fileSize)
}

func (s *DirectorySplitterStore) Fetch(source, label, id string, opts ...FileStorageFetchOption) (DataSlice, error) {
	newId, err := formatId(id)
	if err != nil {
		return DataSlice{}, err
	}
	data, err := s.inner.Fetch(source, label, newId, opts...)
	if err == nil {
		return data, err
	}
	var notFoundError *NotFoundError
	if err == nil {
		return data, err
	} else if errors.As(err, &notFoundError) {
		// FUTURE: expect to remove in Azul 15 this allows migration from the old format to the new format.
		return s.inner.Fetch(source, label, id)
	}
	return data, err

}

func (s *DirectorySplitterStore) Exists(source, label, id string) (bool, error) {
	newId, err := formatId(id)
	if err != nil {
		return false, err
	}
	isExists, err := s.inner.Exists(source, label, newId)
	// FUTURE: expect to remove in Azul 15 this allows migration from the old format to the new format.
	if err == nil && isExists {
		return isExists, err
	}
	return s.inner.Exists(source, label, id)
}

func (s *DirectorySplitterStore) Delete(source, label, id string, opts ...FileStorageDeleteOption) (bool, error) {
	newId, err := formatId(id)
	if err != nil {
		return false, err
	}
	deleted, err := s.inner.Delete(source, label, newId, opts...)
	// FUTURE: expect to remove in Azul 15 this allows migration from the old format to the new format.
	if err != nil || !deleted {
		return s.inner.Delete(source, label, id, opts...)
	}
	return deleted, err
}

func (s *DirectorySplitterStore) Copy(sourceOld, labelOld, idOld, sourceNew, labelNew, idNew string) error {
	newOldId, err := formatId(idOld)
	if err != nil {
		return err
	}
	newNewId, err := formatId(idNew)
	if err != nil {
		return err
	}
	err = s.inner.Copy(sourceOld, labelOld, newOldId, sourceNew, labelNew, newNewId)
	// FUTURE: expect to remove in Azul 15 this allows migration from the old format to the new format.
	if err != nil {
		return s.inner.Copy(sourceOld, labelOld, idOld, sourceNew, labelNew, newNewId)
	}
	return err
}

func (s *DirectorySplitterStore) List(ctx context.Context, prefix string, startAfter string) <-chan FileStorageObjectListInfo {
	// Ensure when listing the startAfter is modified and the resulting listing comes back in the expected format
	// Note - the format is the <source>/<label>/<id> with the extra directory stripped.
	innerStartAfter := startAfter
	if innerStartAfter != "" {
		source, label, id := splitLastThree(innerStartAfter)
		newId, err := formatId(id)
		if err != nil {
			panic(fmt.Sprintf("Failed to list directory because the id couldn't be formatted with error %v", err))
		}
		innerStartAfter = fmt.Sprintf("%s/%s/%s", source, label, newId)
	}

	out := make(chan FileStorageObjectListInfo)
	go func() {
		defer close(out)
		for obj := range s.inner.List(ctx, prefix, innerStartAfter) {
			// Strip out the directory if it's present.
			directory := getDirectory(obj.Key)
			if len(directory) > 0 {
				index := strings.LastIndex(obj.Key, directory)
				obj.Key = obj.Key[:index] + obj.Key[index+len(directory):]
				// Correctly set the source, label and id now that the directory has been removed.
				obj.Source, obj.Label, obj.Id = splitLastThree(obj.Key)
			}
			select {
			case <-ctx.Done():
				return
			case out <- obj:
			}
		}
	}()
	return out
}

// Format the ID ready for storage by adding a directory to it.
func formatId(id string) (string, error) {
	if len(id) < NumberOfChars {
		return id, fmt.Errorf("the id '%s' if not longer than the minimum number of characters for directory splitter %d", id, NumberOfChars)
	}
	return fmt.Sprintf("%s%s%s", id[:NumberOfChars], Separator, id), nil
}

// splitLastThree mirrors the store package's key splitting: the trailing three
// "/"-separated segments of a key are its source, label and id.
func splitLastThree(key string) (source, label, id string) {
	parts := strings.Split(key, Separator)
	n := len(parts)
	if n >= 1 {
		id = parts[n-1]
	}
	if n >= 2 {
		label = parts[n-2]
	}
	if n >= 3 {
		source = parts[n-3]
	}
	return source, label, id
}

func getDirectory(key string) string {
	parts := strings.Split(key, Separator)
	n := len(parts)
	directory := ""
	if n >= 2 {
		directory = parts[n-2]
		sha256 := parts[n-1]
		// Verify the second element is a directory based on the start of the sha256 and not something else.
		if len(directory) == NumberOfChars && strings.HasPrefix(sha256, directory) {
			return directory + Separator
		}
	}
	return ""

}

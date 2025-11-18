package store

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	st "github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/settings"
)

/* Store file on local filesystem. */
type StoreFilesystem struct {
	root string
}

// NewEmptyLocalStore returns a FileStorage wil no data.
// Intended for testing, as aborted tests may otherwise leave files on disk.
func NewEmptyLocalStore(root string) (FileStorage, error) {
	err := os.RemoveAll(root)
	if err != nil {
		return nil, err
	}
	return NewLocalStore(root)
}

func NewLocalStore(root string) (FileStorage, error) {
	err := os.MkdirAll(root, 0755)
	return &StoreFilesystem{root}, err
}

func (s *StoreFilesystem) GetRootPath() string {
	return s.root
}

func (s *StoreFilesystem) Put(source, label, id string, data io.ReadCloser, fileSize int64) error {
	dirname := filepath.Join(s.root, source, label, id[0:1], id[1:2])
	err := os.MkdirAll(dirname, 0755)
	if err != nil {
		// log error, if this is a critical error it will be caught and returned below
		log.Println("Error creating dir")
		log.Println(err)
	}
	path := filepath.Join(dirname, id)
	if _, err := os.Stat(path); err == nil {
		// file was already on disk with same sha256 so abort the put
		return nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		// failed to get file info for some reason but it exists
		// Log the error; if it's serious, we'll return error from WriteFile below
		log.Println("Error calling stat(" + path + ")")
		log.Println(err)
	}
	// trunc will rewrite the file if it exists
	destFile, err := os.OpenFile(path, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0640)
	if err != nil {
		return fmt.Errorf("could not save local file as local file couldn't be opened: %w", err)
	}
	defer destFile.Close()
	_, err = io.Copy(destFile, data)
	return err
}

func (s *StoreFilesystem) Fetch(source, label, id string, opts ...FileStorageFetchOption) (DataSlice, error) {
	empty := NewDataSlice()
	path := filepath.Join(s.root, source, label, id[0:1], id[1:2], id)
	f, err := os.Open(path)
	if err != nil {
		e := fmt.Errorf("%w", &AccessError{msg: fmt.Sprintf("%v", err)})
		if os.IsNotExist(err) {
			e = fmt.Errorf("%w", &NotFoundError{})
		}
		return empty, e
	}

	// find out length of object
	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return empty, fmt.Errorf("%w", &AccessError{msg: fmt.Sprintf("%v", err)})
	}

	fetchOpt := NewFileStorageFetchOptions(opts...)

	// -ve or zero is read all
	if fetchOpt.Size <= 0 {
		fetchOpt.Size = stat.Size()
	}
	// treat -ve as relative to end
	if fetchOpt.Offset < 0 {
		fetchOpt.Offset = stat.Size() + fetchOpt.Offset
	}
	// still -ve (-ve offset was bigger than file)
	if fetchOpt.Offset < 0 {
		fetchOpt.Offset = 0
	}
	// offset after end of file
	if fetchOpt.Offset > 0 && fetchOpt.Offset >= stat.Size() {
		f.Close()
		return empty, fmt.Errorf("%w", &OffsetAfterEnd{msg: fmt.Sprintf("offset after EOF: %d", stat.Size())})
	}
	// requested more than available
	// should we error or be lenient?
	if fetchOpt.Offset+fetchOpt.Size > stat.Size() {
		fetchOpt.Size = stat.Size() - fetchOpt.Offset
	}
	_, err = f.Seek(fetchOpt.Offset, 0)
	if err != nil && err != io.EOF {
		f.Close()
		return empty, fmt.Errorf("%w", &ReadError{msg: fmt.Sprintf("%v", err)})
	}
	limitedReader := NewCloseWrapper(io.LimitReader(f, fetchOpt.Size), f)
	return DataSlice{limitedReader, fetchOpt.Offset, fetchOpt.Size, stat.Size()}, nil
}

func (s *StoreFilesystem) Exists(source, label, id string) (bool, error) {
	path := filepath.Join(s.root, source, label, id[0:1], id[1:2], id)
	if _, err := os.Stat(path); err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, fmt.Errorf("%w", &AccessError{msg: fmt.Sprintf("%v", err)})
	}
}

func (s *StoreFilesystem) Copy(sourceOld, labelOld, idOld, sourceNew, labelNew, idNew string) error {
	// default srcFilePath to just the object name/hash
	srcFilePath := filepath.Join(s.root, sourceOld, labelOld, idOld[0:1], idOld[1:2], idOld)
	existsUnderSource, err := s.Exists(sourceOld, labelOld, idOld)
	if err != nil {
		return fmt.Errorf("error locating source object %s", srcFilePath)
	}

	// check if the src object is under src/label/
	if !existsUnderSource {
		st.Logger.Warn().Msgf("Object %s not found for copy operation", srcFilePath)
		return nil
	}

	// Get file size for consistency with other APIs.
	file, err := os.Open(srcFilePath)
	if err != nil {
		return err
	}
	defer file.Close()
	file_info, err := file.Stat()
	if err != nil {
		return err
	}

	// Copy using existing function
	err = s.Put(sourceNew, labelNew, idNew, file, file_info.Size())
	if err != nil {
		return err
	}

	return nil
}

func (s *StoreFilesystem) Delete(source, label, id string, opts ...FileStorageDeleteOption) (bool, error) {
	deleteOpt := NewFileStorageDeleteOptions(opts...)
	if deleteOpt.IfOlderThan > 0 {
		return false, errors.New("local filesystem does not support ifOlderThan deletion")
	}
	path := filepath.Join(s.root, source, label, id[0:1], id[1:2], id)
	err := os.Remove(path)
	if err != nil {
		e := fmt.Errorf("%w", &AccessError{msg: fmt.Sprintf("%v", err)})
		if os.IsNotExist(err) {
			e = fmt.Errorf("%w", &NotFoundError{})
		}
		return false, e
	}
	return true, nil
}

// Split a file path into it's three parts.
func splitPathToSourceLabelId(path string) (string, string, string) {
	path, id := filepath.Split(path)
	if path == "" {
		return "", "", id
	}
	path, label := filepath.Split(path)
	if path == "" {
		return "", label, id
	}
	_, source := filepath.Split(path)
	return source, label, id
}

func (s *StoreFilesystem) List(ctx context.Context, prefix string, startAfter string) <-chan FileStorageObjectListInfo {
	fileStorageObjects := make(chan FileStorageObjectListInfo)
	go func() {
		var hasPassedStartAfter bool
		if startAfter == "" {
			hasPassedStartAfter = true
		}
		err := filepath.WalkDir(filepath.Join(s.root, prefix), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err // Handle errors during traversal
			}
			if !d.IsDir() { // Only process files, not directories
				// Check if startAfter has been processed, if it hasn't keep skipping elements in the directory until it has been.
				if !hasPassedStartAfter {
					if startAfter == path {
						hasPassedStartAfter = true
					}
					return nil
				}
				source, label, id := splitPathToSourceLabelId(path)
				select {
				case <-ctx.Done():
					return nil
				case fileStorageObjects <- FileStorageObjectListInfo{
					Key:    path,
					Source: source,
					Label:  label,
					Id:     id,
				}:
				}
			}
			return nil
		})
		if err != nil {
			st.Logger.Error().Err(err).Msg("listing directory failed.")
		}
		close(fileStorageObjects) // Close the channel when all files are processed
	}()
	return fileStorageObjects
}

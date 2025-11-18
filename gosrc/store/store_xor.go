package store

import (
	"context"
	"fmt"
	"io"
	"os"
)

/* Wraps Azul files in a multibyte XOR to avoid AV detections */

/* Azul's XOR key (16 bytes): "iloveazul1234567" */
var STORAGE_KEY = [16]byte{0x69, 0x6c, 0x6f, 0x76, 0x65, 0x61, 0x7a, 0x75, 0x6c, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37}

const XOR_FILE_EXT = ".xor"
const BUFFER_SIZE int64 = 1024 * 8

/* Wrapper used to XOR encode as data passes through */
type XORWrapper struct {
	backend io.ReadCloser
	offset  uint64
}

func (w *XORWrapper) Read(buffer []byte) (int, error) {
	count, err := w.backend.Read(buffer)
	if err != nil {
		return count, err
	}

	for i := range count {
		buffer[i] = buffer[i] ^ STORAGE_KEY[(w.offset%uint64(len(STORAGE_KEY)))]
		w.offset += 1
	}

	return count, nil
}

func (w XORWrapper) Close() error {
	return w.backend.Close()
}

/* Store files with a XOR cipher. */
type StoreXOR struct {
	Backend FileStorage
	enabled bool
}

/* Creates a new XOR wrapper. */
func NewXORStore(Backend FileStorage, enabled bool) FileStorage {
	return &StoreXOR{
		Backend,
		enabled,
	}
}

func (s *StoreXOR) Put(source, label, id string, data io.ReadCloser, fileSize int64) error {
	if !s.enabled {
		// If the XOR setting is disabled, files must be stored without the cipher.
		return s.Backend.Put(source, label, id, data, fileSize)
	}

	// We need to buffer the XOR'd data to pass it to our wrapped FileStorage provider
	tmpFile, err := os.CreateTemp("", "azul-xor")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %s", err)
	}

	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Store a buffer so we aren't committing single bytes to the writer
	var buffer [BUFFER_SIZE]byte

	var totalConsumed int64 = 0

	// Loop over the source data and XOR
	cipher := XORWrapper{
		backend: data,
		offset:  0,
	}
	defer cipher.Close()

	// Read until end of source file is reached.
	for {
		// Encode data through the cipher
		available, errMaybeEOF := cipher.Read(buffer[:])
		if errMaybeEOF != nil && errMaybeEOF != io.EOF {
			return fmt.Errorf("failed to read from source file: %s", errMaybeEOF)
		}

		// Flush the data we have written
		_, err := tmpFile.Write(buffer[0:available])
		if err != nil {
			return fmt.Errorf("failed to write to temp file: %s", err)
		}

		totalConsumed += int64(available)
		if errMaybeEOF == io.EOF {
			fileSize = totalConsumed
			break
		}
	}
	_, err = tmpFile.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("failed to seek tempfile in xor back to zero with error: %s", err)
	}

	return s.Backend.Put(source, label, id+XOR_FILE_EXT, tmpFile, fileSize)
}

func (s *StoreXOR) Fetch(source, label, id string, opts ...FileStorageFetchOption) (DataSlice, error) {
	empty := NewDataSlice()

	if s.enabled {
		// Test for .xor first
		xorExists, err := s.Backend.Exists(source, label, id+XOR_FILE_EXT)
		if err != nil {
			return empty, err
		}

		if !xorExists {
			// No XOR'd copy of this file, pass directly to the underlying reader
			return s.Backend.Fetch(source, label, id, opts...)
		}
	} else {
		// Check for a non-XOR'd file first as the user has disabled XOR'ing
		rawExists, err := s.Backend.Exists(source, label, id)
		if err != nil {
			return empty, err
		}

		if rawExists {
			// Use the raw as we have spotted that
			return s.Backend.Fetch(source, label, id, opts...)
		}
	}

	// Fetch the XOR'd stream
	backingStream, err := s.Backend.Fetch(source, label, id+XOR_FILE_EXT, opts...)

	if err != nil {
		return empty, err
	}

	// Wrap it
	wrapper := XORWrapper{
		backend: backingStream.DataReader,
		offset:  uint64(backingStream.Start),
	}

	backingStream.DataReader = &wrapper

	return backingStream, nil
}

func (s *StoreXOR) Exists(source, label, id string) (bool, error) {
	var firstQuery string
	var secondQuery string
	if s.enabled {
		// Test for a XOR'd file first
		firstQuery = id + XOR_FILE_EXT
		// Fall back to the raw copy if available
		secondQuery = id
	} else {
		// Test for a raw file first (as the user has disabled XOR'ing)
		firstQuery = id
		// Fall back to finding a pre-existing XOR'd file
		secondQuery = id + XOR_FILE_EXT
	}

	resp, err := s.Backend.Exists(source, label, firstQuery)
	if err != nil || resp {
		return resp, err
	}

	return s.Backend.Exists(source, label, secondQuery)
}

func (s *StoreXOR) Copy(sourceOld, labelOld, idOld, sourceNew, labelNew, idNew string) error {
	// Test if the source is XOR'd
	resp, err := s.Backend.Exists(sourceOld, labelOld, idOld+XOR_FILE_EXT)
	if err != nil {
		return err
	}

	if resp {
		// Copy from XOR to XOR
		return s.Backend.Copy(sourceOld, labelOld, idOld+XOR_FILE_EXT, sourceNew, labelNew, idNew+XOR_FILE_EXT)
	} else {
		// Copy from non-XOR to non-XOR
		// FUTURE: Could opportunistically XOR here; would require a read & put
		return s.Backend.Copy(sourceOld, labelOld, idOld, sourceNew, labelNew, idNew)
	}
}

func (s *StoreXOR) Delete(source, label, id string, opts ...FileStorageDeleteOption) (bool, error) {
	// Delete both the XOR'd and raw versions if they exist
	rawExists, err := s.Backend.Exists(source, label, id+XOR_FILE_EXT)
	if err != nil {
		return false, err
	}

	xorExists, err := s.Backend.Exists(source, label, id)
	if err != nil {
		return false, err
	}

	if !rawExists && !xorExists {
		return false, &NotFoundError{}
	}

	didDelete := false

	if rawExists {
		resp, err := s.Backend.Delete(source, label, id+XOR_FILE_EXT, opts...)
		didDelete = resp
		if err != nil {
			return didDelete, err
		}
	}

	if xorExists {
		resp, err := s.Backend.Delete(source, label, id, opts...)
		if err != nil {
			return didDelete, err
		}
		didDelete = didDelete || resp
	}

	return didDelete, nil
}

func (s *StoreXOR) List(ctx context.Context, prefix string, startAfter string) <-chan FileStorageObjectListInfo {
	return s.Backend.List(ctx, prefix, startAfter)
}

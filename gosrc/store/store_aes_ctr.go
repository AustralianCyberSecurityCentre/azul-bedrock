package store

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"os"
)

// Extension (including version) of the aes encoding.
const HEADER_VERSION = "1"

// Version is attached to the extension to save space rather than adding it to header.
const AES_CTR_FILE_EXT = ".aesc" + HEADER_VERSION
const AES_STORAGE_VERSION = 1
const KEY_LENGTH = 24
const KEY_SALT_LENGTH = 8

// Expected to be 24 bytes
const HEADER_BYTE_LENGTH = KEY_SALT_LENGTH + aes.BlockSize

type AESCtrEncoder struct {
	backend      io.ReadCloser
	aesKey       []byte
	iv           []byte
	salt         []byte
	cipherStream cipher.Stream
}

type AESCtrDecoder struct {
	contentBackend DataSlice
	headerBackend  DataSlice
	aesKey         []byte
	iv             []byte
	salt           []byte
	cipherStream   cipher.Stream
}

// Create a new AES Encoder generating the key, iv and salt and streamCipher.
func NewAESCtrEncoder(backend io.ReadCloser, key []byte) (*AESCtrEncoder, error) {
	if len(key) != KEY_LENGTH {
		panic(fmt.Sprintf("The provided AES key is not exactly %d bytes long it was %d bytes long instead and cannot be used.", KEY_LENGTH, len(key)))
	}
	iv := make([]byte, aes.BlockSize)
	salt := make([]byte, KEY_SALT_LENGTH)
	_, err := rand.Read(iv)
	if err != nil {
		return &AESCtrEncoder{}, fmt.Errorf("failed to generate random bytes for AES iv vector. Error: %v", err)
	}
	_, err = rand.Read(salt)
	if err != nil {
		return &AESCtrEncoder{}, fmt.Errorf("failed to generate random bytes for AES salt. Error: %v", err)
	}

	// 32 bytes for AES-256
	saltedKey := append(key, salt...)
	if len(saltedKey) != KEY_LENGTH+KEY_SALT_LENGTH {
		return &AESCtrEncoder{}, fmt.Errorf("adding the key and salt together resulted in a key length (%d) not the expected length (%d)", len(saltedKey), KEY_LENGTH+KEY_SALT_LENGTH)
	}
	cipherBlock, err := aes.NewCipher(saltedKey)
	if err != nil {
		return &AESCtrEncoder{}, err
	}
	ctr := cipher.NewCTR(cipherBlock, iv)

	return &AESCtrEncoder{
		backend:      backend,
		aesKey:       key,
		iv:           iv,
		salt:         salt,
		cipherStream: ctr,
	}, nil
}

func (w *AESCtrEncoder) getHeaderBytes() []byte {
	// NOTE order of salt then IV is important and must match the decoder
	headerBytes := append(w.salt, w.iv...)
	return headerBytes
}

// Encode the backend's content into the provided buffer.
func (w *AESCtrEncoder) Encode(buffer []byte) (int, error) {
	count, err := w.backend.Read(buffer)
	if err != nil {
		return count, err
	}
	w.cipherStream.XORKeyStream(buffer, buffer)
	return count, nil
}

func (w *AESCtrEncoder) Close() error {
	return w.backend.Close()
}

// Shifts an IV to a specific block.
func shiftIvToBlock(counter []byte, n uint64) {
	for i := len(counter) - 1; i >= 0 && n > 0; i-- {
		sum := uint64(counter[i]) + (n & 0xff)
		counter[i] = byte(sum)
		n = (n >> 8) + (sum >> 8)
	}
}

// Two separate readers once for the content and the other for the header, these could be the same reader.
// But are allowed to be different in the case where there is a file offset.
func NewAESCtrDecoder(contentBackend DataSlice, headerBackend DataSlice, key []byte, offset uint64) (*AESCtrDecoder, error) {
	if len(key) != KEY_LENGTH {
		panic(fmt.Sprintf("The provided AES key is not exactly %d bytes long it was %d bytes long instead and cannot be used.", KEY_LENGTH, len(key)))
	}
	// NOTE order of salt then IV is important and must match the encoder
	salt := make([]byte, KEY_SALT_LENGTH)
	_, err := headerBackend.DataReader.Read(salt)
	if err != nil {
		return &AESCtrDecoder{}, fmt.Errorf("failed to read Salt when decoding file with error: %v", err)
	}
	iv := make([]byte, aes.BlockSize)
	_, err = headerBackend.DataReader.Read(iv)
	if err != nil {
		return &AESCtrDecoder{}, fmt.Errorf("failed to read IV when decoding file with error: %v", err)
	}

	// 32 bytes for AES-256
	saltedKey := append(key, salt...)
	if len(saltedKey) != KEY_LENGTH+KEY_SALT_LENGTH {
		return &AESCtrDecoder{}, fmt.Errorf("adding the key and salt together resulted in a key length (%d) not the expected length (%d)", len(saltedKey), KEY_LENGTH+KEY_SALT_LENGTH)
	}
	block, err := aes.NewCipher(saltedKey)
	if err != nil {
		return &AESCtrDecoder{}, fmt.Errorf("failed to initialise the aes block cipher with error %v", err)
	}

	// Shift IV to offset
	blockNum := uint64(offset / aes.BlockSize)
	blockOffset := int(offset % aes.BlockSize)
	shiftedIv := iv
	shiftIvToBlock(shiftedIv, blockNum)

	// Counter cipher object
	ctr := cipher.NewCTR(block, shiftedIv)

	// Seek to the correct section of the IV now that it's at the correct block
	if blockOffset > 0 {
		dummy := make([]byte, blockOffset)
		ctr.XORKeyStream(dummy, dummy)
	}

	return &AESCtrDecoder{
		contentBackend: contentBackend,
		headerBackend:  headerBackend,
		aesKey:         key,
		iv:             iv,
		salt:           salt,
		cipherStream:   ctr,
	}, nil
}

func (w *AESCtrDecoder) Read(buffer []byte) (int, error) {
	count, err := w.contentBackend.DataReader.Read(buffer)
	if err != nil {
		return count, err
	}
	w.cipherStream.XORKeyStream(buffer, buffer)
	return count, nil
}

func (w *AESCtrDecoder) Close() error {
	contentErr := w.contentBackend.DataReader.Close()
	headerErr := w.headerBackend.DataReader.Close()
	if contentErr != nil {
		return contentErr
	}
	return headerErr
}

type StoreAESCtr struct {
	Backend FileStorage
	key     []byte
	enabled bool
}

func NewAESCtrStore(backend FileStorage, aesKey string, enabled bool) FileStorage {
	if len(aesKey) != KEY_LENGTH {
		panic(fmt.Sprintf("The provided AES key is not 0 bytes and is not exactly %d bytes long it was %d bytes long instead and cannot be used.", KEY_LENGTH, len(aesKey)))
	}
	return &StoreAESCtr{
		Backend: backend,
		key:     []byte(aesKey),
		enabled: enabled,
	}
}

// Readcloser with the open file or stream and fileSize if known otherwise provide -1 for file size.
func (s *StoreAESCtr) Put(source, label, id string, data io.ReadCloser, fileSize int64) error {
	if !s.enabled {
		// If the AES_CTR setting is disabled, files must be stored without the cipher.
		return s.Backend.Put(source, label, id, data, fileSize)
	}

	// We need to buffer the AES_CTR'd data to pass it to our wrapped FileStorage provider
	tmpFile, err := os.CreateTemp("", "azul-aes")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %s", err)
	}

	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Store a buffer so we aren't committing single bytes to the writer
	var buffer [BUFFER_SIZE]byte
	var totalConsumed int64 = 0

	// Loop over the source data and AES_CTR
	cipher, err := NewAESCtrEncoder(data, s.key)
	if err != nil {
		return fmt.Errorf("failed to create AES encoder when encoding file %s", id)
	}
	defer cipher.Close()

	// Flush the data we have written
	headerBytes := cipher.getHeaderBytes()
	_, err = tmpFile.Write(headerBytes)
	if err != nil {
		return fmt.Errorf("failed to write header to temp file: %s", err)
	}

	totalConsumed += int64(len(headerBytes))

	// Read until end of source file is reached.
	for {
		// Encode data through the cipher
		available, errMaybeEOF := cipher.Encode(buffer[:])
		// available, errMaybeEOF := cipher.backend.Read(buffer[:])
		if errMaybeEOF != nil && errMaybeEOF != io.EOF {
			return fmt.Errorf("failed to read from source file: %s", errMaybeEOF)
		}

		// Flush the data we have written
		_, err = tmpFile.Write(buffer[0:available])
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
		return fmt.Errorf("failed to seek tempfile in aes_ctr back to zero with error: %s", err)
	}

	return s.Backend.Put(source, label, id+AES_CTR_FILE_EXT, tmpFile, fileSize)
}

// Fetch file from offset to size, if offset is 0 fetch from start, if size is -1 fetch to the end of the file.
func (s *StoreAESCtr) Fetch(source, label, id string, opts ...FileStorageFetchOption) (DataSlice, error) {
	empty := NewDataSlice()
	var err error
	if !s.enabled {
		// Check for a non-AES_CTR'd file first as the user has disabled AES_CTR'ing
		rawExists, err := s.Backend.Exists(source, label, id)
		if err != nil {
			return empty, err
		}

		if rawExists {
			// Use the raw as we have spotted that
			return s.Backend.Fetch(source, label, id, opts...)
		}
	}
	// Test for .aesc first
	aesCtrExists, err := s.Backend.Exists(source, label, id+AES_CTR_FILE_EXT)
	if err != nil {
		return empty, err
	}

	if !aesCtrExists {
		// No AES_CTR'd copy of this file, pass directly to the underlying reader
		return s.Backend.Fetch(source, label, id, opts...)
	}

	fetchOptions := NewFileStorageFetchOptions(opts...)
	originalOffset := fetchOptions.Offset

	var decoder *AESCtrDecoder
	var contentBackingStream DataSlice

	var correctedSize int64
	var correctedStart int64

	/*
		cases (SIZE)
		-ve or 0 size (read all content)
		+ve size + offset greater than length of the file, then read just to end of file.

		cases (OFFSETS)
		-ve - (read the end of the file)
		-ve that is greater than whole file (read whole file).
		+ve and larger than the whole file size (error out)

	*/

	// trivial case where the offset is at the start of the file
	if originalOffset == 0 {
		newSize := fetchOptions.Size
		// If the whole file isn't being read need to add the HEADER length to the read request so no bytes are dropped.
		if newSize > 0 {
			newSize += HEADER_BYTE_LENGTH
		}
		// Fetch the AES_CTR'd stream
		contentBackingStream, err = s.Backend.Fetch(source, label, id+AES_CTR_FILE_EXT, append(opts, WithOffsetAndSize(0, newSize))...)
		if err != nil {
			return empty, err
		}
		correctedStart = 0
		correctedSize = contentBackingStream.Size - HEADER_BYTE_LENGTH
		decoder, err = NewAESCtrDecoder(contentBackingStream, contentBackingStream, s.key, uint64(fetchOptions.Offset))
		if err != nil {
			contentBackingStream.DataReader.Close()
			return empty, err
		}
		// offset is negative
	} else {
		// Fetch the header to allow the collection of the IV and salt
		headerBackingStream, err := s.Backend.Fetch(source, label, id+AES_CTR_FILE_EXT, WithOffsetAndSize(0, HEADER_BYTE_LENGTH))
		if err != nil {
			contentBackingStream.DataReader.Close()
			return empty, err
		}
		// Calculate the actual offset using the size of the file (if offset is negative) and factor in the Header length
		var rawFileOffset int64
		var positiveOriginalOffset int64
		// Need to adjust offset to ensure it skips the header in the underlying file.
		if originalOffset < 0 {
			rawFileOffset = headerBackingStream.Avail + originalOffset
			positiveOriginalOffset = rawFileOffset - HEADER_BYTE_LENGTH
			if positiveOriginalOffset < 0 {
				rawFileOffset = HEADER_BYTE_LENGTH
				positiveOriginalOffset = 0
			}
		} else {
			rawFileOffset = originalOffset + HEADER_BYTE_LENGTH
			positiveOriginalOffset = int64(originalOffset)
		}

		// Regardless or positive or negative original offset the offset is now positive and both behave the same.
		contentBackingStream, err = s.Backend.Fetch(source, label, id+AES_CTR_FILE_EXT, append(opts, WithOffsetAndSize(rawFileOffset, fetchOptions.Size))...)
		if err != nil {
			return empty, err
		}

		if rawFileOffset >= contentBackingStream.Avail {
			contentBackingStream.DataReader.Close()
			return empty, fmt.Errorf("%w", &OffsetAfterEnd{msg: fmt.Sprintf("offset after EOF: %d", contentBackingStream.Avail)})
		}

		decoder, err = NewAESCtrDecoder(contentBackingStream, headerBackingStream, s.key, uint64(positiveOriginalOffset))
		if err != nil {
			contentBackingStream.DataReader.Close()
			headerBackingStream.DataReader.Close()
			return empty, fmt.Errorf("failed to create AES decoder for file %s with error: %v", id, err)
		}
		// Corrected values
		correctedSize = contentBackingStream.Size
		correctedStart = contentBackingStream.Start - HEADER_BYTE_LENGTH
	}
	outputStream := DataSlice{
		// Total available should exclude the AES header.
		Avail:      contentBackingStream.Avail - HEADER_BYTE_LENGTH,
		Start:      correctedStart,
		Size:       correctedSize,
		DataReader: decoder,
	}
	return outputStream, nil
}

// Check a file exists in the filestore.
func (s *StoreAESCtr) Exists(source, label, id string) (bool, error) {
	var firstQuery string
	var secondQuery string
	if s.enabled {
		// Test for a AES_CTR'd file first
		firstQuery = id + AES_CTR_FILE_EXT
		// Fall back to the raw copy if available
		secondQuery = id
	} else {
		// Test for a raw file first (as the user has disabled AES_CTR'ing)
		firstQuery = id
		// Fall back to finding a pre-existing AES_CTR'd file
		secondQuery = id + AES_CTR_FILE_EXT
	}

	resp, err := s.Backend.Exists(source, label, firstQuery)
	if err != nil || resp {
		return resp, err
	}

	return s.Backend.Exists(source, label, secondQuery)
}

// Delete deletes the specified key if older than supplied unix timestamp in seconds.
func (s *StoreAESCtr) Delete(source, label, id string, opts ...FileStorageDeleteOption) (bool, error) {
	// Delete both the AES_CTR'd and raw versions if they exist
	aesExists, err := s.Backend.Exists(source, label, id+AES_CTR_FILE_EXT)
	if err != nil {
		return false, err
	}

	rawExists, err := s.Backend.Exists(source, label, id)
	if err != nil {
		return false, err
	}

	if !rawExists && !aesExists {
		return false, &NotFoundError{}
	}

	didDelete := false

	if aesExists {
		resp, err := s.Backend.Delete(source, label, id+AES_CTR_FILE_EXT, opts...)
		didDelete = resp
		if err != nil {
			return didDelete, err
		}
	}

	if rawExists {
		resp, err := s.Backend.Delete(source, label, id, opts...)
		if err != nil {
			return didDelete, err
		}
		didDelete = didDelete || resp
	}

	return didDelete, nil
}

// Copy within the S3 store from old to new location
func (s *StoreAESCtr) Copy(sourceOld, labelOld, idOld, sourceNew, labelNew, idNew string) error {
	// Test if the source is AES_CTR'd
	aesSourceExists, err := s.Backend.Exists(sourceOld, labelOld, idOld+AES_CTR_FILE_EXT)
	if err != nil {
		return err
	}

	if aesSourceExists {
		// Copy from AES_CTR_FILE_EXT to AES_CTR_FILE_EXT
		return s.Backend.Copy(sourceOld, labelOld, idOld+AES_CTR_FILE_EXT, sourceNew, labelNew, idNew+AES_CTR_FILE_EXT)
	} else {
		// Copy from non-AES_CTR to non-AES_CTR
		// FUTURE: Could opportunistically AES_CTR here; would require a read & put
		return s.Backend.Copy(sourceOld, labelOld, idOld, sourceNew, labelNew, idNew)
	}
}

// List all objects in the S3 store, the provided context must be cancelled when list is no longer needed.
func (s *StoreAESCtr) List(ctx context.Context, prefix string, startAfter string) <-chan FileStorageObjectListInfo {
	return s.Backend.List(ctx, prefix, startAfter)
}

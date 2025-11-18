package store

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Size of the buffered readers buffer 1MB
const MAX_BUFFERED_READER_BYTES = 1024 * 1024

// Perform a concurrent upload at 50MiB
const MAX_FILE_BYTES_BEFORE_CONCURRENT_UPLOAD = 50 * 1024 * 1024

// Concurrent upload threads
const NUM_CONCURRENT_UPLOAD_THREADS = 10

// Buffer_Sizes (must be 5MiB+ for AWS minimum chunk size)
const CONCURRENT_BUFFER_SIZE_BYTES = 6 * 1024 * 1024

type DataSlice struct {
	DataReader         io.ReadCloser
	Start, Size, Avail int64
}

/* Create an empty Dataslice with a reader with no bytes.*/
func NewDataSlice() DataSlice {
	return DataSlice{
		DataReader: io.NopCloser(bytes.NewReader([]byte{})),
		Start:      0,
		Size:       0,
		Avail:      0,
	}
}

type FileStorageObjectListInfo struct {
	Key string
	// Expected form of the key is: 'Source/Label/Id' (if it isn't use the key directly.)
	// Source section of the object
	Source string
	// Label section of the object
	Label string
	// Is section of the object.
	Id string
}

// -------------------------- Fetch options --------------------------
type FileStorageFetchOptions struct {
	Offset int64
	Size   int64
}
type FileStorageFetchOption func(*FileStorageFetchOptions)

// Default constructor for file fetch
func NewFileStorageFetchOptions(opts ...FileStorageFetchOption) *FileStorageFetchOptions {
	deleteOptions := &FileStorageFetchOptions{
		Offset: 0,
		Size:   -1,
	}
	for _, opt := range opts {
		opt(deleteOptions)
	}
	return deleteOptions
}

// Set the offset and size for a fetch operation
func WithOffsetAndSize(offset int64, size int64) FileStorageFetchOption {
	return func(opts *FileStorageFetchOptions) {
		opts.Offset = offset
		opts.Size = size
	}
}

// -------------------------- Delete options --------------------------
type FileStorageDeleteOptions struct {
	IfOlderThan int64
}

type FileStorageDeleteOption func(*FileStorageDeleteOptions)

// Default constructor for file delete options
func NewFileStorageDeleteOptions(opts ...FileStorageDeleteOption) *FileStorageDeleteOptions {
	deleteOptions := &FileStorageDeleteOptions{
		IfOlderThan: 0,
	}
	for _, opt := range opts {
		opt(deleteOptions)
	}
	return deleteOptions
}

// Set the option tto only delete files if they are older than a certain date time (expressed as seconds since epoch)
func WithDeleteIfOlderThan(ifOlderThan int64) FileStorageDeleteOption {
	return func(opts *FileStorageDeleteOptions) {
		opts.IfOlderThan = ifOlderThan
	}
}

type FileStorage interface {
	// Readcloser with the open file or stream and fileSize if known otherwise provide -1 for file size.
	Put(source, label, id string, data io.ReadCloser, fileSize int64) error
	// Fetch file from offset to size, if offset is 0 fetch from start, if size is -1 fetch to the end of the file.
	Fetch(source, label, id string, opts ...FileStorageFetchOption) (DataSlice, error)
	// Check a file exists in the filestore.
	Exists(source, label, id string) (bool, error)
	// Delete deletes the specified key if older than supplied unix timestamp in seconds.
	Delete(source, label, id string, opts ...FileStorageDeleteOption) (bool, error)
	// Copy within the S3 store from old to new location
	Copy(sourceOld, labelOld, idOld, sourceNew, labelNew, idNew string) error
	// List all objects in the S3 store, the provided context must be cancelled when list is no longer needed.
	List(ctx context.Context, prefix string, startAfter string) <-chan FileStorageObjectListInfo
}

type OffsetAfterEnd struct {
	msg string
}

func (r *OffsetAfterEnd) Error() string {
	return r.msg
}

type NotFoundError struct{}

func (e *NotFoundError) Error() string {
	return "not found"
}

type AccessError struct {
	msg string
}

func (e *AccessError) Error() string {
	return fmt.Sprintf("no access: %v", e.msg)
}

type ReadError struct {
	msg string
}

func (e *ReadError) Error() string {
	return fmt.Sprintf("read error: %v", e.msg)
}

// updateIdPath prefixes source and label path to id if source and label are non-empty
func createIdPath(source, label, id string) string {
	return strings.Join([]string{source, label, id}, "/")
}

// Split an S3 path into it's original components source, label, id
func splitIdPath(key string) (string, string, string) {
	splitString := strings.Split(key, "/")
	slices.Reverse(splitString)
	source, label, id := "", "", ""
	for i := range splitString {
		if id == "" {
			id = splitString[i]
		} else if label == "" {
			label = splitString[i]
		} else if source == "" {
			source = splitString[i]
			return source, label, id
		}
	}
	return source, label, id
}

// reportStreamsOpMetric report a streams method duration for prometheus
func reportStreamsOpMetric(promStreamsOperationDuration *prometheus.HistogramVec, startTime int64, operationName string, err error) {
	// Don't bother reporting if prometheus histogram isn't provided.
	if promStreamsOperationDuration == nil {
		return
	}
	result := "ok"
	if err != nil {
		result = "error"
	}
	durationSeconds := float64(time.Now().UnixNano()-startTime) / 1e9
	promStreamsOperationDuration.WithLabelValues(operationName, result).Observe(durationSeconds)
}

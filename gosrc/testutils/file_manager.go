package testutils

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/cart"
)

type DownloadSource string

const VirusTotal DownloadSource = "Virus Total"
const AzureBlobStorage DownloadSource = "Azure Blob Storage"
const LocalCache DownloadSource = "Local file system Cache"

type NotFoundError struct {
	Sha256 string
	Source []DownloadSource
}

func NewNotFoundError(sha256 string, source ...DownloadSource) *NotFoundError {
	return &NotFoundError{
		Sha256: sha256,
		Source: source,
	}
}

// Implement the `Error` method to satisfy the `error` interface
func (e *NotFoundError) Error() string {
	stringSources := []string{}
	for _, val := range e.Source {
		stringSources = append(stringSources, string(val))
	}
	return fmt.Sprintf("Could not find the sha256 '%s' in %s", e.Sha256, strings.Join(stringSources, ", "))
}

type FileManager struct {
	settings      *FileManagerSettings
	storageClient BaseStorage
}

func NewFileManager() (*FileManager, error) {
	settings := ParseFileManagerSettings()
	var client BaseStorage
	var err error

	if settings.AzureBlobCacheEnabled {
		client, err = NewAzureStorageClient(settings)
		if err != nil {
			return nil, err
		}
	}
	if settings.FileCachingEnabled {
		if _, err := os.Stat(settings.FileCacheDir); errors.Is(err, os.ErrNotExist) {
			err = os.MkdirAll(settings.FileCacheDir, os.ModePerm)
			if err != nil {
				return nil, err
			}
		}
	}

	return &FileManager{
		settings:      settings,
		storageClient: client,
	}, nil
}

/*Download the file from file from virustotal and write the raw bytes to the output stream.*/
func (fm *FileManager) downloadFromVt(sha256 string, outStream io.Writer) error {
	downloadUrl := fmt.Sprintf("%s/files/%s/download", fm.settings.VirustotalApiUrl, sha256)
	vtRequest, err := http.NewRequest(http.MethodGet, downloadUrl, nil)
	if err != nil {
		return err
	}
	client := http.Client{Timeout: time.Duration(fm.settings.RequestTimeout) * time.Second}
	vtRequest.Header.Add("x-Apikey", fm.settings.VirustotalApiKey)
	resp, err := client.Do(vtRequest)

	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return NewNotFoundError(sha256, VirusTotal)
	} else if resp.StatusCode > 300 || resp.StatusCode < 100 {
		return fmt.Errorf("failed to download file from VT with status code %d and error %s", resp.StatusCode, resp.Status)
	}

	numberOfBytes, err := io.Copy(outStream, resp.Body)
	if numberOfBytes == 0 {
		return NewNotFoundError(sha256, VirusTotal)
	}
	return err
}

/*Load the local cart file, uncart it and write the bytes to the output stream.*/
func (fm *FileManager) loadFromLocal(sha256 string, outStream io.Writer) error {
	localPath := filepath.Join(fm.settings.FileCacheDir, fmt.Sprintf("%s%s", sha256, CART_FILE_SUFFIX))
	if _, err := os.Stat(localPath); errors.Is(err, os.ErrNotExist) {
		return NewNotFoundError(sha256, LocalCache)
	}
	foundBytes, err := cart.UncartBytes(localPath)
	if err != nil {
		return err
	}
	_, err = outStream.Write(foundBytes)
	return err
}

func (fm *FileManager) saveToLocal(sha256 string, uncartedReader io.Reader) error {
	localPath := filepath.Join(fm.settings.FileCacheDir, fmt.Sprintf("%s%s", sha256, CART_FILE_SUFFIX))
	fileHandle, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to save to local file could not create file %s with error %s", localPath, err.Error())
	}
	return cart.PackCart(uncartedReader, fileHandle)
}

/*Save to local storage cache and optionally to blob storage.*/
func (fm *FileManager) saveToActiveCaches(sha256 string, rawFileBytes []byte, isSaveToBlob bool) error {
	var err error
	// Save to local.
	if fm.settings.FileCachingEnabled {
		err = fm.saveToLocal(sha256, bufio.NewReader(bytes.NewBuffer(rawFileBytes)))
		if err != nil {
			return fmt.Errorf("warning could not save %s to %s with error %s", sha256, LocalCache, err.Error())
		}
	}

	// Save to Blob storage
	if fm.settings.AzureBlobCacheEnabled {
		err = fm.storageClient.SaveFile(sha256, bufio.NewReader(bytes.NewBuffer(rawFileBytes)))
		if err != nil {
			return fmt.Errorf("warning could not save %s to %s with error %s", sha256, AzureBlobStorage, err.Error())
		}
	}
	return nil
}

func (fm *FileManager) DownloadFileBytes(sha256 string) ([]byte, error) {
	var err error
	var notFoundError *NotFoundError
	var b bytes.Buffer
	failedToFindFileIn := []DownloadSource{}

	writer := bufio.NewWriter(&b)
	// Load from local file system cache.
	if fm.settings.FileCachingEnabled {
		err = fm.loadFromLocal(sha256, writer)
		if err == nil {
			writer.Flush()
			if b.Len() == 0 {
				return []byte{}, fmt.Errorf("empty file provided by %s for sha256 %s", LocalCache, sha256)
			}
			return b.Bytes(), nil
		} else if errors.As(err, &notFoundError) {
			failedToFindFileIn = append(failedToFindFileIn, LocalCache)
		} else {
			return []byte{}, fmt.Errorf("warning unable to load from %s with error: %v", LocalCache, err)
		}
	}
	// Load from Blob Storage.
	if fm.settings.AzureBlobCacheEnabled {
		b.Reset()
		writer.Reset(&b)
		err = fm.storageClient.DownloadFile(sha256, writer)
		if err == nil {
			writer.Flush()
			if b.Len() == 0 {
				return []byte{}, fmt.Errorf("empty file provided by %s for sha256 %s", AzureBlobStorage, sha256)
			}
			rawBytes := b.Bytes()
			err = fm.saveToActiveCaches(sha256, rawBytes, false)
			if err != nil {
				return rawBytes, err
			}
			return b.Bytes(), nil
		} else if errors.As(err, &notFoundError) {
			failedToFindFileIn = append(failedToFindFileIn, AzureBlobStorage)
		} else {
			return []byte{}, fmt.Errorf("warning unable to load from %s with error: %v", AzureBlobStorage, err)
		}
	}

	// Load from VT.
	if fm.settings.VirustotalEnabled {
		b.Reset()
		writer.Reset(&b)
		err = fm.downloadFromVt(sha256, writer)
		if err == nil {
			writer.Flush()
			if b.Len() == 0 {
				return []byte{}, fmt.Errorf("empty file provided by %s for sha256 %s", VirusTotal, sha256)
			}
			fileBytes := b.Bytes()
			err = fm.saveToActiveCaches(sha256, fileBytes, true)
			if err != nil {
				return []byte{}, err
			}
			return fileBytes, nil
		} else if errors.As(err, &notFoundError) {
			failedToFindFileIn = append(failedToFindFileIn, VirusTotal)
		} else {
			return []byte{}, fmt.Errorf("warning unable to load from %s with error: %v", VirusTotal, err)
		}
	}

	notFoundError = NewNotFoundError(sha256, failedToFindFileIn...)
	log.Printf("Warning: %s", notFoundError.Error())
	return []byte{}, notFoundError
}

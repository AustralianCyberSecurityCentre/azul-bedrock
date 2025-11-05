package testutils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/cart"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
)

type BaseStorage interface {
	DownloadFile(sha256 string, outputStream io.Writer) error
	SaveFile(sha256 string, uncartedContents io.Reader) error
}

type AzureStorageClient struct {
	client     *azblob.Client
	settings   *FileManagerSettings
	fileSuffix string
}

func NewAzureStorageClient(settings *FileManagerSettings) (BaseStorage, error) {

	var client *azblob.Client
	var err error

	// Parent context for this instance of the filestore
	ctx := context.Background()

	storageUri, err := url.Parse(settings.AzureStorageAccountAddress)
	if err != nil {
		log.Fatal(err)
	}
	// cloud storage is in format: https://<storage-account-name>.blob.core.windows.net/
	// Azurite local storage emulator will be in format http://<ip>:<port>/<storage-account-name>/
	// therefore storageAccount must be set manually for Azurite support
	storeName := strings.Split(storageUri.Hostname(), ".")[0]

	cred, err := azblob.NewSharedKeyCredential(storeName, settings.AzureStorageAccessKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get credential using Storage Access Key with error: %v", err)
	}
	client, err = azblob.NewClientWithSharedKeyCredential(settings.AzureStorageAccountAddress, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain blobstore: %v", err)
	}
	_, err = client.CreateContainer(ctx, settings.AzureContainerName, nil)
	if err == nil {
		log.Printf("Created container %s\n", settings.AzureContainerName)
	} else if !bloberror.HasCode(err, bloberror.ResourceAlreadyExists, bloberror.ContainerAlreadyExists) {
		return nil, err
	}

	// Return created client.
	return &AzureStorageClient{
		client:     client,
		settings:   settings,
		fileSuffix: CART_FILE_SUFFIX,
	}, nil
}

/*Download the file from file from Azure Blob storage and write the raw bytes to the output stream.*/
func (asc *AzureStorageClient) DownloadFile(sha256 string, outputStream io.Writer) error {
	ctx := context.Background()
	tempCartedFile, err := os.CreateTemp("", "")
	if err != nil {
		return fmt.Errorf("failed to create temporary file when dowloding with error: %s", err.Error())
	}
	defer tempCartedFile.Close()
	defer os.Remove(tempCartedFile.Name())
	_, err = asc.client.DownloadFile(ctx, asc.settings.AzureContainerName, fmt.Sprintf("%s%s", sha256, asc.fileSuffix), tempCartedFile, &blob.DownloadFileOptions{})
	// Check if not found was the reason for the error.
	var azureError *azcore.ResponseError
	if errors.As(err, &azureError) && azureError.StatusCode == http.StatusNotFound {
		return NewNotFoundError(sha256, AzureBlobStorage)
	}
	if err != nil {
		return fmt.Errorf("failed to create temporary file when downloading from azure blob storage with error %s", err)
	}
	err = tempCartedFile.Close()
	if err != nil {
		return fmt.Errorf("failed to close temporary file when downloading from azure blob storage with error %s", err)
	}
	uncartedFile, err := cart.Uncart(tempCartedFile.Name())
	if err != nil {
		return fmt.Errorf("failed when uncarting file from blob storage: %v", err)
	}
	_, err = io.Copy(outputStream, uncartedFile)
	return err
}

/*Upload the provided file to Azure blob storage.*/
func (asc *AzureStorageClient) SaveFile(sha256 string, uncartedContents io.Reader) error {
	file, err := os.CreateTemp("", "")
	if err != nil {
		return fmt.Errorf("failed to create temporary file when uploading to azure blob storage with error %s", err.Error())
	}
	defer file.Close()
	defer os.Remove(file.Name())
	err = cart.PackCart(uncartedContents, file)
	if err != nil {
		return fmt.Errorf("failed to pack cart with error: %v", err.Error())
	}
	_, err = file.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("failed seeking when uploading cart: %v", err.Error())
	}
	ctx := context.Background()
	_, err = asc.client.UploadFile(ctx, asc.settings.AzureContainerName, fmt.Sprintf("%s%s", sha256, asc.fileSuffix), file, &blockblob.UploadFileOptions{})
	return err
}

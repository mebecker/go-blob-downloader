package main

import (
	// "bufio"
	// "bytes"
	"context"
	"fmt"
	"log"
	// "os"
	"net/http"
	"net/url"
	"io"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

// Azure Storage Quickstart Sample - Demonstrate how to upload, list, download, and delete blobs.
//
// Documentation References:
// - What is a Storage Account - https://docs.microsoft.com/azure/storage/common/storage-create-storage-account
// - Blob Service Concepts - https://docs.microsoft.com/rest/api/storageservices/Blob-Service-Concepts
// - Blob Service Go SDK API - https://godoc.org/github.com/Azure/azure-storage-blob-go
// - Blob Service REST API - https://docs.microsoft.com/rest/api/storageservices/Blob-Service-REST-API
// - Scalability and performance targets - https://docs.microsoft.com/azure/storage/common/storage-scalability-targets
// - Azure Storage Performance and Scalability checklist https://docs.microsoft.com/azure/storage/common/storage-performance-checklist
// - Storage Emulator - https://docs.microsoft.com/azure/storage/common/storage-use-emulator

func handleError(err error) {
	if err != nil {
		log.Fatal(err.Error())
	}
}

func extractUrlParts(blobURL string) (string, string, string, error) {
    // Parse the URL
    u, err := url.Parse(blobURL)
    if err != nil {
        return "", "", "", err
    }

	// Extract the account URL
    accountURL := fmt.Sprintf("%s://%s", u.Scheme, u.Host)

    // Split the path to get the container name and blob path
    pathParts := strings.SplitN(strings.TrimPrefix(u.Path, "/"), "/", 2)
    if len(pathParts) < 2 {
        return "", "", "", fmt.Errorf("invalid blob URL format")
    }

    containerName := pathParts[0]
    blobPath := pathParts[1]

	fmt.Printf("Account Url: %v, Container name: %v, Blob path: %v\n", accountURL, containerName, blobPath)

    return accountURL, containerName, blobPath, nil
}

func downloadBlobHandler(w http.ResponseWriter, r *http.Request) {

	bloburl := r.URL.Query().Get("bloburl")
    if bloburl == "" {
        // If the parameter is not present, return an error message
        http.Error(w, "Missing query string parameter: bloburl", http.StatusBadRequest)
        return
    }

	fmt.Printf("Handling request url: %s\n", r.URL.String())

	u, err := url.Parse(bloburl)
	if err != nil {
		fmt.Printf("Failed to parse URL: %v\n", err)
		return
	}

	accountURL, containerName, blobPath, err := extractUrlParts(u.String())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Create a DefaultAzureCredential
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		fmt.Printf("Failed to create DefaultAzureCredential: %v\n", err)
		return
	}

	blobClient, err := azblob.NewClient(accountURL, cred, nil)
	if err != nil {
		fmt.Printf("Failed to create BlobClient: %v\n", err)
		return
	}

	downloadResponse, err := blobClient.DownloadStream(context.TODO(), containerName, blobPath, nil)

	if err != nil {
		fmt.Printf("Failed to download blob: %v\n", err)
		http.Error(w, "Server Error", http.StatusInternalServerError)
		return
	}
	
	retryReader := downloadResponse.NewRetryReader(context.TODO(), &azblob.RetryReaderOptions{})

	// Copy the downloaded blob to the HTTP response
	_, err = io.Copy(w, retryReader)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to write blob to response: %v", err), http.StatusInternalServerError)
		return
	}

	err = retryReader.Close()
	handleError(err)
}

func main() {
    http.HandleFunc("/", downloadBlobHandler)
    fmt.Println("Server started at :8080")
    http.ListenAndServe(":8080", nil)
}
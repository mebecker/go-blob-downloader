package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"io"
	"strings"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

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

	accountURL, containerName, blobPath, err := extractUrlParts(bloburl)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		http.Error(w, "Server Error", http.StatusInternalServerError)
		return
	}

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		fmt.Printf("Failed to create DefaultAzureCredential: %v\n", err)
		http.Error(w, "Server Error", http.StatusInternalServerError)
		return
	}

	blobClient, err := azblob.NewClient(accountURL, cred, nil)
	if err != nil {
		fmt.Printf("Failed to create BlobClient: %v\n", err)
		http.Error(w, "Server Error", http.StatusInternalServerError)
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

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func main() {
    http.HandleFunc("/", downloadBlobHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/readiness", readinessHandler)
    fmt.Println("Server started at :8080")
    http.ListenAndServe(":8080", nil)
}
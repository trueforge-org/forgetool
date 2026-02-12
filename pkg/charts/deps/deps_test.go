package deps

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestSmokeDeps(t *testing.T) {}

func TestDownloadFile(t *testing.T) {
	tests := []struct {
		name         string
		setupFunc    func(t *testing.T) (url, destination string, cleanup func())
		wantErr      bool
		checkContent func(t *testing.T, destination string)
	}{
		{
			name: "Download file successfully",
			setupFunc: func(t *testing.T) (string, string, func()) {
				// Create a test HTTP server
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("test file content"))
				}))

				tmpDir := t.TempDir()
				destination := filepath.Join(tmpDir, "testfile.txt")

				return server.URL, destination, server.Close
			},
			wantErr: false,
			checkContent: func(t *testing.T, destination string) {
				content, err := os.ReadFile(destination)
				if err != nil {
					t.Fatalf("Failed to read downloaded file: %v", err)
				}
				if string(content) != "test file content" {
					t.Errorf("File content = %q, want %q", string(content), "test file content")
				}
			},
		},
		{
			name: "Download empty file",
			setupFunc: func(t *testing.T) (string, string, func()) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(""))
				}))

				tmpDir := t.TempDir()
				destination := filepath.Join(tmpDir, "empty.txt")

				return server.URL, destination, server.Close
			},
			wantErr: false,
			checkContent: func(t *testing.T, destination string) {
				content, err := os.ReadFile(destination)
				if err != nil {
					t.Fatalf("Failed to read downloaded file: %v", err)
				}
				if len(content) != 0 {
					t.Errorf("Expected empty file, got %d bytes", len(content))
				}
			},
		},
		{
			name: "Handle HTTP error status",
			setupFunc: func(t *testing.T) (string, string, func()) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				}))

				tmpDir := t.TempDir()
				destination := filepath.Join(tmpDir, "notfound.txt")

				return server.URL, destination, server.Close
			},
			wantErr: false, // downloadFile doesn't check status codes, just downloads
			checkContent: func(t *testing.T, destination string) {
				// File should still be created even with 404 status
				if _, err := os.Stat(destination); os.IsNotExist(err) {
					t.Error("File should exist even with 404 status")
				}
			},
		},
		{
			name: "Invalid URL",
			setupFunc: func(t *testing.T) (string, string, func()) {
				tmpDir := t.TempDir()
				destination := filepath.Join(tmpDir, "invalid.txt")

				return "http://invalid-url-that-does-not-exist-12345.com", destination, func() {}
			},
			wantErr: true,
			checkContent: func(t *testing.T, destination string) {
				// No check needed for error case
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, destination, cleanup := tt.setupFunc(t)
			defer cleanup()

			err := downloadFile(url, destination)
			if (err != nil) != tt.wantErr {
				t.Errorf("downloadFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				tt.checkContent(t, destination)
			}
		})
	}
}

func TestDownloadFile_InvalidDestination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	}))
	defer server.Close()

	// Try to write to an invalid path (directory that doesn't exist)
	err := downloadFile(server.URL, "/nonexistent/path/file.txt")
	if err == nil {
		t.Error("Expected error when writing to invalid path, got nil")
	}
}

// Note: LoadGPGKey, fetchIndexFile, fetchDependency, copyDependency, and DownloadDeps
// are integration functions that interact with the file system, network, and other packages.
// Testing them would require:
// 1. Mocking HTTP calls for network operations
// 2. Setting up complex directory structures
// 3. Mocking dependencies on other packages (chartFile, fluxhandler, helper)
// These are better suited for integration tests rather than unit tests.
// The core downloadFile function is tested above as it's the most reusable utility.


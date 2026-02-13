package clustertemplate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

func TestResolveVersion_UsesExplicitEnv(t *testing.T) {
	t.Setenv(VersionEnv, "v9.9.9")

	version, err := resolveVersion()
	if err != nil {
		t.Fatalf("resolveVersion returned error: %v", err)
	}
	if version != "v9.9.9" {
		t.Fatalf("resolveVersion returned %q, want %q", version, "v9.9.9")
	}
}

func TestResolveVersion_FetchesLatestRelease(t *testing.T) {
	t.Setenv(VersionEnv, "")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v1.2.3"}`))
	}))
	defer server.Close()

	oldLatestURL := latestReleaseURL
	oldClient := httpClient
	latestReleaseURL = server.URL
	httpClient = server.Client()
	t.Cleanup(func() {
		latestReleaseURL = oldLatestURL
		httpClient = oldClient
	})

	version, err := resolveVersion()
	if err != nil {
		t.Fatalf("resolveVersion returned error: %v", err)
	}
	if version != "v1.2.3" {
		t.Fatalf("resolveVersion returned %q, want %q", version, "v1.2.3")
	}
}

func TestToCache_DownloadsAndExtractsRelease(t *testing.T) {
	t.Setenv(VersionEnv, "v1.0.0")

	archiveBytes := buildTarGz(t, map[string]string{
		"cluster-template-v1.0.0/base/example.yaml": "base",
		"cluster-template-v1.0.0/root/.env":         "root",
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(archiveBytes)
	}))
	defer server.Close()

	oldArchiveURL := releaseArchiveURL
	oldClient := httpClient
	oldCacheDir := helper.CacheDir
	releaseArchiveURL = server.URL + "/%s"
	httpClient = server.Client()
	helper.CacheDir = t.TempDir()
	t.Cleanup(func() {
		releaseArchiveURL = oldArchiveURL
		httpClient = oldClient
		helper.CacheDir = oldCacheDir
	})

	if err := ToCache(); err != nil {
		t.Fatalf("ToCache returned error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(helper.CacheDir, "base", "example.yaml"))
	if err != nil {
		t.Fatalf("expected extracted base/example.yaml: %v", err)
	}
	if string(content) != "base" {
		t.Fatalf("unexpected file content: %q", content)
	}
}

func buildTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)

	for name, content := range files {
		data := []byte(content)
		if err := tarWriter.WriteHeader(&tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(data)),
		}); err != nil {
			t.Fatalf("WriteHeader failed: %v", err)
		}
		if _, err := io.Copy(tarWriter, bytes.NewReader(data)); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}

	if err := tarWriter.Close(); err != nil {
		t.Fatalf("tar close failed: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("gzip close failed: %v", err)
	}

	return buffer.Bytes()
}

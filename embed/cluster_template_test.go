package embed

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

func TestResolveClusterTemplateVersion_UsesExplicitEnv(t *testing.T) {
	t.Setenv(clusterTemplateVersionEnv, "v9.9.9")

	version, err := resolveClusterTemplateVersion()
	if err != nil {
		t.Fatalf("resolveClusterTemplateVersion returned error: %v", err)
	}
	if version != "v9.9.9" {
		t.Fatalf("resolveClusterTemplateVersion returned %q, want %q", version, "v9.9.9")
	}
}

func TestResolveClusterTemplateVersion_FetchesLatestRelease(t *testing.T) {
	t.Setenv(clusterTemplateVersionEnv, "")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v1.2.3"}`))
	}))
	defer server.Close()

	oldLatestURL := clusterTemplateLatestReleaseURL
	oldClient := clusterTemplateHTTPClient
	clusterTemplateLatestReleaseURL = server.URL
	clusterTemplateHTTPClient = server.Client()
	t.Cleanup(func() {
		clusterTemplateLatestReleaseURL = oldLatestURL
		clusterTemplateHTTPClient = oldClient
	})

	version, err := resolveClusterTemplateVersion()
	if err != nil {
		t.Fatalf("resolveClusterTemplateVersion returned error: %v", err)
	}
	if version != "v1.2.3" {
		t.Fatalf("resolveClusterTemplateVersion returned %q, want %q", version, "v1.2.3")
	}
}

func TestClusterTemplateToCache_DownloadsAndExtractsRelease(t *testing.T) {
	t.Setenv(clusterTemplateVersionEnv, "v1.0.0")

	archiveBytes := buildTarGz(t, map[string]string{
		"cluster-template-v1.0.0/base/example.yaml": "base",
		"cluster-template-v1.0.0/root/.env":         "root",
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(archiveBytes)
	}))
	defer server.Close()

	oldArchiveURL := clusterTemplateReleaseArchiveURL
	oldClient := clusterTemplateHTTPClient
	oldCacheDir := helper.CacheDir
	clusterTemplateReleaseArchiveURL = server.URL + "/%s"
	clusterTemplateHTTPClient = server.Client()
	helper.CacheDir = t.TempDir()
	t.Cleanup(func() {
		clusterTemplateReleaseArchiveURL = oldArchiveURL
		clusterTemplateHTTPClient = oldClient
		helper.CacheDir = oldCacheDir
	})

	clusterTemplateToCache()

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

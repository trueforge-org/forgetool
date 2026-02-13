package clustertemplate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

type roundTripFn func(*http.Request) (*http.Response, error)

func (f roundTripFn) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func withClusterTemplateHTTP(t *testing.T, client *http.Client, latestURL, archiveURL string) {
	t.Helper()
	oldClient := httpClient
	oldLatest := latestReleaseURL
	oldArchive := releaseArchiveURL
	httpClient = client
	latestReleaseURL = latestURL
	releaseArchiveURL = archiveURL
	t.Cleanup(func() {
		httpClient = oldClient
		latestReleaseURL = oldLatest
		releaseArchiveURL = oldArchive
	})
}

func resetClusterTemplateHooks() {
	mkdirAllHook = os.MkdirAll
	openFileHook = func(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
		return os.OpenFile(name, flag, perm)
	}
	copyHook = io.Copy
	closeHook = func(c io.Closer) error { return c.Close() }
	absPathHook = filepath.Abs
	isWithinCacheHook = isWithinCache
}

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

func TestToCache_HookErrors(t *testing.T) {
	oldResolve := resolveVersionHook
	oldDownload := downloadReleaseHook
	resolveVersionHook = func() (string, error) { return "", fmt.Errorf("resolve failed") }
	downloadReleaseHook = func(string) error { return nil }
	t.Cleanup(func() {
		resolveVersionHook = oldResolve
		downloadReleaseHook = oldDownload
	})

	if err := ToCache(); err == nil || !strings.Contains(err.Error(), "resolve failed") {
		t.Fatalf("expected resolve hook error, got %v", err)
	}

	resolveVersionHook = func() (string, error) { return "v1.2.3", nil }
	downloadReleaseHook = func(string) error { return fmt.Errorf("download failed") }
	if err := ToCache(); err == nil || !strings.Contains(err.Error(), "download failed") {
		t.Fatalf("expected download hook error, got %v", err)
	}
}

func TestResolveVersion_ErrorBranches(t *testing.T) {
	t.Setenv(VersionEnv, "not-a-tag")
	if _, err := resolveVersion(); err == nil {
		t.Fatal("expected invalid version env error")
	}

	t.Setenv(VersionEnv, "v1.2.3-..x")
	if _, err := resolveVersion(); err == nil {
		t.Fatal("expected traversal version env error")
	}

	t.Setenv(VersionEnv, "")
	withClusterTemplateHTTP(t, &http.Client{}, "http://[::1", releaseArchiveURL)
	if _, err := resolveVersion(); err == nil {
		t.Fatal("expected new request error")
	}

	clientErr := &http.Client{Transport: roundTripFn(func(*http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("network")
	})}
	withClusterTemplateHTTP(t, clientErr, "https://example.com/latest", releaseArchiveURL)
	if _, err := resolveVersion(); err == nil || !strings.Contains(err.Error(), "network") {
		t.Fatalf("expected transport error, got %v", err)
	}

	statusServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer statusServer.Close()
	withClusterTemplateHTTP(t, statusServer.Client(), statusServer.URL, releaseArchiveURL)
	if _, err := resolveVersion(); err == nil {
		t.Fatal("expected non-200 status error")
	}

	invalidJSONServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{"))
	}))
	defer invalidJSONServer.Close()
	withClusterTemplateHTTP(t, invalidJSONServer.Client(), invalidJSONServer.URL, releaseArchiveURL)
	if _, err := resolveVersion(); err == nil {
		t.Fatal("expected decode error")
	}

	emptyTagServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":""}`))
	}))
	defer emptyTagServer.Close()
	withClusterTemplateHTTP(t, emptyTagServer.Client(), emptyTagServer.URL, releaseArchiveURL)
	if _, err := resolveVersion(); err == nil {
		t.Fatal("expected empty tag_name error")
	}
}

func TestDownloadRelease_ErrorBranches(t *testing.T) {
	oldURL := releaseArchiveURL
	oldClient := httpClient
	releaseArchiveURL = "://bad/%s"
	t.Cleanup(func() {
		releaseArchiveURL = oldURL
		httpClient = oldClient
	})
	if err := downloadRelease("v1.2.3"); err == nil {
		t.Fatal("expected bad archive URL error")
	}

	statusServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer statusServer.Close()
	releaseArchiveURL = statusServer.URL + "/%s"
	httpClient = statusServer.Client()
	if err := downloadRelease("v1.2.3"); err == nil {
		t.Fatal("expected non-200 archive status error")
	}

	badBodyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-gzip"))
	}))
	defer badBodyServer.Close()
	releaseArchiveURL = badBodyServer.URL + "/%s"
	httpClient = badBodyServer.Client()
	if err := downloadRelease("v1.2.3"); err == nil {
		t.Fatal("expected extract archive error")
	}
}

func TestExtractArchive_ErrorAndSafetyBranches(t *testing.T) {
	resetClusterTemplateHooks()
	t.Cleanup(resetClusterTemplateHooks)

	if err := extractArchive(bytes.NewReader([]byte("bad-gzip"))); err == nil {
		t.Fatal("expected gzip reader error")
	}

	oldCache := helper.CacheDir
	helper.CacheDir = "\x00"
	t.Cleanup(func() { helper.CacheDir = oldCache })
	absPathHook = func(string) (string, error) { return "", fmt.Errorf("abs failed") }
	if err := extractArchive(bytes.NewReader(buildTarGz(t, map[string]string{"x/y": "z"}))); err == nil || !strings.Contains(err.Error(), "abs failed") {
		t.Fatalf("expected abs path error, got %v", err)
	}
	absPathHook = filepath.Abs

	helper.CacheDir = t.TempDir()
	if err := extractArchive(bytes.NewReader(buildGzipRaw(t, []byte("not-a-tar")))); err == nil {
		t.Fatal("expected tar next error")
	}

	if err := extractArchive(bytes.NewReader(buildMalformedTarGzSingleFile(t, "root/file.txt", 20, []byte("x")))); err == nil {
		t.Fatal("expected malformed tar read/copy error")
	}

	if err := extractArchive(bytes.NewReader(buildTarGzRawHeaders(t,
		tar.Header{Name: "rootonly", Typeflag: tar.TypeDir, Mode: 0o755},
		tar.Header{Name: "root/", Typeflag: tar.TypeDir, Mode: 0o755},
	))); err != nil {
		t.Fatalf("expected root-only entries to be skipped cleanly: %v", err)
	}

	if err := extractArchive(bytes.NewReader(buildTarGzRawHeaders(t,
		tar.Header{Name: "root/../evil", Typeflag: tar.TypeReg, Mode: 0o644, Size: 1},
	))); err == nil {
		t.Fatal("expected '..' archive path rejection")
	}

	isWithinCacheHook = func(string, string, string) bool { return false }
	if err := extractArchive(bytes.NewReader(buildTarGzRawHeaders(t,
		tar.Header{Name: "root/inside.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: 1},
	))); err == nil || !strings.Contains(err.Error(), "invalid archive path") {
		t.Fatalf("expected invalid archive path rejection, got %v", err)
	}
	isWithinCacheHook = isWithinCache

	conflictFile := filepath.Join(helper.CacheDir, "conflict")
	if err := os.WriteFile(conflictFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("seed conflict file failed: %v", err)
	}
	if err := extractArchive(bytes.NewReader(buildTarGzRawHeaders(t,
		tar.Header{Name: "root/conflict", Typeflag: tar.TypeDir, Mode: 0o755},
	))); err == nil {
		t.Fatal("expected directory create error over existing file")
	}

	if err := extractArchive(bytes.NewReader(buildTarGzRawHeaders(t,
		tar.Header{Name: "root/conflict/file.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len("x"))},
	))); err == nil {
		t.Fatal("expected parent mkdir error for file entry")
	}

	if err := os.MkdirAll(filepath.Join(helper.CacheDir, "adir"), 0o755); err != nil {
		t.Fatalf("mkdir adir failed: %v", err)
	}
	if err := extractArchive(bytes.NewReader(buildTarGzRawHeaders(t,
		tar.Header{Name: "root/adir", Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len("x"))},
	))); err == nil {
		t.Fatal("expected open file error when target is existing directory")
	}

	closeHook = func(io.Closer) error { return fmt.Errorf("close failed") }
	if err := extractArchive(bytes.NewReader(buildTarGzRawHeaders(t,
		tar.Header{Name: "root/closeerr.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len("x"))},
	))); err == nil || !strings.Contains(err.Error(), "close failed") {
		t.Fatalf("expected close hook error, got %v", err)
	}

	copyHook = func(io.Writer, io.Reader) (int64, error) { return 0, fmt.Errorf("copy failed") }
	if err := extractArchive(bytes.NewReader(buildTarGzRawHeaders(t,
		tar.Header{Name: "root/copyerr.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len("x"))},
	))); err == nil || !strings.Contains(err.Error(), "copy failed") {
		t.Fatalf("expected copy hook error, got %v", err)
	}

	openFileHook = func(string, int, os.FileMode) (io.WriteCloser, error) { return nil, fmt.Errorf("open failed") }
	if err := extractArchive(bytes.NewReader(buildTarGzRawHeaders(t,
		tar.Header{Name: "root/openerr.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len("x"))},
	))); err == nil || !strings.Contains(err.Error(), "open failed") {
		t.Fatalf("expected open hook error, got %v", err)
	}

	mkdirAllHook = func(path string, _ os.FileMode) error {
		if strings.Contains(path, "mkdirerr") {
			return fmt.Errorf("mkdir failed")
		}
		return os.MkdirAll(path, os.ModePerm)
	}
	if err := extractArchive(bytes.NewReader(buildTarGzRawHeaders(t,
		tar.Header{Name: "root/mkdirerr", Typeflag: tar.TypeDir, Mode: 0o755},
	))); err == nil || !strings.Contains(err.Error(), "mkdir failed") {
		t.Fatalf("expected mkdir hook dir error, got %v", err)
	}
}

func TestIsWithinCache(t *testing.T) {
	if !isWithinCache("/tmp/cache", "/tmp/cache/", "/tmp/cache") {
		t.Fatal("expected cache root path to be allowed")
	}
	if !isWithinCache("/tmp/cache", "/tmp/cache/", "/tmp/cache/a") {
		t.Fatal("expected cache subpath to be allowed")
	}
	if isWithinCache("/tmp/cache", "/tmp/cache/", "/tmp/other/a") {
		t.Fatal("expected outside cache path to be rejected")
	}
}

func buildTarGzRawHeaders(t *testing.T, headers ...tar.Header) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, hdr := range headers {
		header := hdr
		if err := tarWriter.WriteHeader(&header); err != nil {
			t.Fatalf("WriteHeader failed: %v", err)
		}
		if header.Typeflag == tar.TypeReg && header.Size > 0 {
			payload := bytes.Repeat([]byte("x"), int(header.Size))
			if _, err := tarWriter.Write(payload); err != nil {
				t.Fatalf("Write payload failed: %v", err)
			}
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

func buildMalformedTarGzSingleFile(t *testing.T, name string, declaredSize int64, actual []byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	if err := tarWriter.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: declaredSize, Typeflag: tar.TypeReg}); err != nil {
		t.Fatalf("WriteHeader failed: %v", err)
	}
	if _, err := tarWriter.Write(actual); err != nil {
		t.Fatalf("Write malformed payload failed: %v", err)
	}
	// Intentionally do not close tarWriter to keep archive malformed.
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("gzip close failed: %v", err)
	}
	return buffer.Bytes()
}

func buildGzipRaw(t *testing.T, data []byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	if _, err := gzipWriter.Write(data); err != nil {
		t.Fatalf("gzip write failed: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("gzip close failed: %v", err)
	}
	return buffer.Bytes()
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

package deps

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestDownloadFile(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("index-data"))
	}))
	defer ts.Close()

	dest := filepath.Join(t.TempDir(), "index.yaml")
	if err := downloadFile(ts.URL, dest); err != nil {
		t.Fatalf("downloadFile failed: %v", err)
	}
	b, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read downloaded file failed: %v", err)
	}
	if string(b) != "index-data" {
		t.Fatalf("unexpected file content: %q", string(b))
	}
}

func TestFetchIndexFileBranches(t *testing.T) {
	oldIndexCache := helper.IndexCache
	helper.IndexCache = t.TempDir()
	t.Cleanup(func() {
		helper.IndexCache = oldIndexCache
	})

	if err := fetchIndexFile("oci-repo", "oci-repo", "oci://example.com/charts"); err != nil {
		t.Fatalf("OCI URL should be skipped, got error: %v", err)
	}

	repoDir := "example.com/charts"
	cachedPath := filepath.Join(helper.IndexCache, repoDir, "index.yaml")
	if err := os.MkdirAll(filepath.Dir(cachedPath), os.ModePerm); err != nil {
		t.Fatalf("mkdir cache dir failed: %v", err)
	}
	if err := os.WriteFile(cachedPath, []byte("cached"), 0644); err != nil {
		t.Fatalf("write cached index failed: %v", err)
	}
	if err := fetchIndexFile("repo", repoDir, "https://example.com/index.yaml"); err != nil {
		t.Fatalf("fetchIndexFile should use cache, got error: %v", err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("fresh"))
	}))
	defer ts.Close()

	repoDir2 := "example.com/newrepo"
	if err := fetchIndexFile("repo2", repoDir2, ts.URL); err != nil {
		t.Fatalf("fetchIndexFile download failed: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(helper.IndexCache, repoDir2, "index.yaml"))
	if err != nil {
		t.Fatalf("read downloaded index failed: %v", err)
	}
	if string(b) != "fresh" {
		t.Fatalf("unexpected downloaded index content: %q", string(b))
	}
}

func TestFetchDependencyCachedAndCopyDependency(t *testing.T) {
	oldHelmCache := helper.HelmCache
	helper.HelmCache = t.TempDir()
	t.Cleanup(func() {
		helper.HelmCache = oldHelmCache
	})

	repoDir := "repo"
	name := "dep"
	version := "1.2.3"
	cachedDep := filepath.Join(helper.HelmCache, repoDir, name+"-"+version+".tgz")
	if err := os.MkdirAll(filepath.Dir(cachedDep), os.ModePerm); err != nil {
		t.Fatalf("mkdir cached dep dir failed: %v", err)
	}
	if err := os.WriteFile(cachedDep, []byte("tgz-bytes"), 0644); err != nil {
		t.Fatalf("write cached dependency failed: %v", err)
	}

	if err := fetchDependency("repo", repoDir, name, version, "https://example.com/index.yaml"); err != nil {
		t.Fatalf("fetchDependency cached path failed: %v", err)
	}

	chartFolder := t.TempDir()
	if err := copyDependency(chartFolder, "repo", repoDir, name, version); err != nil {
		t.Fatalf("copyDependency failed: %v", err)
	}
	dest := filepath.Join(chartFolder, "charts", name+"-"+version+".tgz")
	b, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read copied dependency failed: %v", err)
	}
	if string(b) != "tgz-bytes" {
		t.Fatalf("unexpected copied content: %q", string(b))
	}
}

func TestDownloadDepsNoDependencies(t *testing.T) {
	chartDir := t.TempDir()
	chartPath := filepath.Join(chartDir, "Chart.yaml")
	chartYAML := "apiVersion: v2\nname: app\nversion: 0.1.0\n"
	if err := os.WriteFile(chartPath, []byte(chartYAML), 0644); err != nil {
		t.Fatalf("write chart file failed: %v", err)
	}

	if err := DownloadDeps(chartPath, ""); err != nil {
		t.Fatalf("DownloadDeps should succeed with no dependencies: %v", err)
	}
	if _, err := os.Stat(filepath.Join(chartDir, "charts")); err != nil {
		t.Fatalf("expected charts directory to be created: %v", err)
	}
}

func TestLoadGPGKeyWithMockTransport(t *testing.T) {
	oldGpgDir := helper.GpgDir
	helper.GpgDir = t.TempDir()
	t.Cleanup(func() {
		helper.GpgDir = oldGpgDir
	})

	oldTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := io.NopCloser(bytes.NewBufferString("gpg-data"))
		return &http.Response{StatusCode: http.StatusOK, Body: body, Header: make(http.Header)}, nil
	})
	t.Cleanup(func() {
		http.DefaultTransport = oldTransport
	})

	if err := LoadGPGKey(); err != nil {
		t.Fatalf("LoadGPGKey failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(helper.GpgDir, "pubring.gpg")); err != nil {
		t.Fatalf("expected pubring.gpg to be created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(helper.GpgDir, "certman.gpg")); err != nil {
		t.Fatalf("expected certman.gpg to be created: %v", err)
	}
}

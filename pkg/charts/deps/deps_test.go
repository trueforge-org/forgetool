package deps

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trueforge-org/forgetool/v4/pkg/helper"
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

func TestLoadGPGKey_DownloadFailures(t *testing.T) {
	oldGpgDir := helper.GpgDir
	helper.GpgDir = t.TempDir()
	t.Cleanup(func() { helper.GpgDir = oldGpgDir })

	origGet := httpGet
	t.Cleanup(func() { httpGet = origGet })

	call := 0
	httpGet = func(_ string) (*http.Response, error) {
		call++
		if call == 1 {
			return nil, errors.New("first")
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString("ok")), Header: make(http.Header)}, nil
	}
	if err := LoadGPGKey(); err == nil {
		t.Fatalf("expected first download failure")
	}

	call = 0
	httpGet = func(_ string) (*http.Response, error) {
		call++
		if call == 2 {
			return nil, errors.New("second")
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString("ok")), Header: make(http.Header)}, nil
	}
	if err := LoadGPGKey(); err == nil {
		t.Fatalf("expected second download failure")
	}
}

func TestDownloadFile_ErrorBranches(t *testing.T) {
	origGet := httpGet
	t.Cleanup(func() { httpGet = origGet })

	httpGet = func(_ string) (*http.Response, error) { return nil, errors.New("boom") }
	if err := downloadFile("https://example.com", filepath.Join(t.TempDir(), "x")); err == nil {
		t.Fatalf("expected get error")
	}

	httpGet = func(_ string) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: errReadCloser{}, Header: make(http.Header)}, nil
	}
	if err := downloadFile("https://example.com", filepath.Join(t.TempDir(), "x")); err == nil {
		t.Fatalf("expected body read error")
	}

	httpGet = func(_ string) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString("x")), Header: make(http.Header)}, nil
	}
	d := filepath.Join(t.TempDir(), "dest-dir")
	if err := os.MkdirAll(d, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := downloadFile("https://example.com", d); err == nil {
		t.Fatalf("expected write error")
	}
}

type errReadCloser struct{}

func (errReadCloser) Read(_ []byte) (int, error) { return 0, errors.New("read") }
func (errReadCloser) Close() error               { return nil }

func TestFetchDependency_NonCachedBranches(t *testing.T) {
	origHelmCache := helper.HelmCache
	helper.HelmCache = t.TempDir()
	t.Cleanup(func() { helper.HelmCache = origHelmCache })

	origPull := helmPull
	t.Cleanup(func() { helmPull = origPull })

	helper.HelmCache = filepath.Join(t.TempDir(), "cache-file")
	if err := os.WriteFile(helper.HelmCache, []byte("x"), 0644); err != nil {
		t.Fatalf("write cache-file failed: %v", err)
	}
	if err := fetchDependency("repo", "repo/dir", "dep", "1.0.0", "url"); err == nil {
		t.Fatalf("expected mkdir error")
	}

	helper.HelmCache = t.TempDir()
	helmPull = func(_, _, _, _ string, _ bool) error { return errors.New("pull") }
	if err := fetchDependency("repo", "repo/dir", "dep", "1.0.0", "url"); err == nil {
		t.Fatalf("expected pull error")
	}

	helmPull = func(_, _, _, _ string, _ bool) error { return nil }
	if err := fetchDependency("repo", "repo/dir", "dep", "1.0.0", "url"); err == nil {
		t.Fatalf("expected missing file error")
	}

	helmPull = func(_, name, version, repoCacheDir string, _ bool) error {
		return os.WriteFile(filepath.Join(repoCacheDir, name+"-"+version+".tgz"), []byte("tgz"), 0644)
	}
	if err := fetchDependency("repo", "repo/dir", "dep", "1.0.0", "url"); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestProcessChartDependencyAndDownloadDepsBranches(t *testing.T) {
	origIndexCache := helper.IndexCache
	origHelmCache := helper.HelmCache
	origGet := httpGet
	origPull := helmPull
	t.Cleanup(func() {
		helper.IndexCache = origIndexCache
		helper.HelmCache = origHelmCache
		httpGet = origGet
		helmPull = origPull
	})

	helper.IndexCache = t.TempDir()
	helper.HelmCache = t.TempDir()
	httpGet = func(_ string) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("index")), Header: make(http.Header)}, nil
	}
	helmPull = func(_, name, version, repoCacheDir string, _ bool) error {
		return os.WriteFile(filepath.Join(repoCacheDir, name+"-"+version+".tgz"), []byte("tgz"), 0644)
	}

	if err := processChartDependency(t.TempDir(), "dep", "1.0.0", "https://example.com/charts"); err != nil {
		t.Fatalf("expected process success, got %v", err)
	}

	helper.IndexCache = filepath.Join(t.TempDir(), "index-file")
	if err := os.WriteFile(helper.IndexCache, []byte("x"), 0644); err != nil {
		t.Fatalf("write index-file failed: %v", err)
	}
	if err := processChartDependency(t.TempDir(), "dep", "1.0.0", "https://example.com/charts"); err == nil {
		t.Fatalf("expected fetch index wrapper error")
	}

	helper.IndexCache = t.TempDir()
	helper.HelmCache = filepath.Join(t.TempDir(), "helm-file")
	if err := os.WriteFile(helper.HelmCache, []byte("x"), 0644); err != nil {
		t.Fatalf("write helm-file failed: %v", err)
	}
	if err := processChartDependency(t.TempDir(), "dep", "1.0.0", "https://example.com/charts"); err == nil {
		t.Fatalf("expected fetch dependency wrapper error")
	}

	chartDir := t.TempDir()
	chartPath := filepath.Join(chartDir, "Chart.yaml")
	chartYAML := "apiVersion: v2\nname: app\nversion: 0.1.0\ndependencies:\n  - name: dep\n    version: 1.0.0\n    repository: https://example.com/charts\n"
	if err := os.WriteFile(chartPath, []byte(chartYAML), 0644); err != nil {
		t.Fatalf("write chart failed: %v", err)
	}

	helper.IndexCache = filepath.Join(t.TempDir(), "index-file2")
	if err := os.WriteFile(helper.IndexCache, []byte("x"), 0644); err != nil {
		t.Fatalf("write index-file2 failed: %v", err)
	}
	if err := DownloadDeps(chartPath, ""); err == nil {
		t.Fatalf("expected dependency processing error")
	}

	blockingChartsPath := filepath.Join(chartDir, "charts")
	if err := os.RemoveAll(blockingChartsPath); err != nil {
		t.Fatalf("remove charts path failed: %v", err)
	}
	if err := os.WriteFile(blockingChartsPath, []byte("x"), 0644); err != nil {
		t.Fatalf("write blocking charts file failed: %v", err)
	}
	if err := DownloadDeps(chartPath, ""); err == nil {
		t.Fatalf("expected charts mkdir error")
	}
}

func TestDeps_FinalErrorBranches(t *testing.T) {
	oldGpgDir := helper.GpgDir
	oldIndexCache := helper.IndexCache
	oldHelmCache := helper.HelmCache
	origGet := httpGet
	origPull := helmPull
	t.Cleanup(func() {
		helper.GpgDir = oldGpgDir
		helper.IndexCache = oldIndexCache
		helper.HelmCache = oldHelmCache
		httpGet = origGet
		helmPull = origPull
	})

	root := t.TempDir()
	blocking := filepath.Join(root, "block")
	if err := os.WriteFile(blocking, []byte("x"), 0644); err != nil {
		t.Fatalf("write blocking file failed: %v", err)
	}
	helper.GpgDir = filepath.Join(blocking, "dir")
	if err := LoadGPGKey(); err == nil {
		t.Fatalf("expected gpg mkdir error")
	}

	helper.IndexCache = t.TempDir()
	httpGet = func(_ string) (*http.Response, error) { return nil, errors.New("download fail") }
	if err := fetchIndexFile("repo", "repoDir", "https://example.com/index.yaml"); err == nil {
		t.Fatalf("expected fetch index download wrapper error")
	}

	helper.HelmCache = t.TempDir()
	chartFolder := t.TempDir()
	chartsPath := filepath.Join(chartFolder, "charts")
	if err := os.WriteFile(chartsPath, []byte("x"), 0644); err != nil {
		t.Fatalf("write blocking charts failed: %v", err)
	}
	if err := copyDependency(chartFolder, "repo", "repoDir", "dep", "1.0.0"); err == nil {
		t.Fatalf("expected copyDependency mkdir error")
	}

	helper.IndexCache = t.TempDir()
	helper.HelmCache = t.TempDir()
	httpGet = func(_ string) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("index")), Header: make(http.Header)}, nil
	}
	helmPull = func(_, name, version, repoCacheDir string, _ bool) error {
		return os.WriteFile(filepath.Join(repoCacheDir, name+"-"+version+".tgz"), []byte("tgz"), 0644)
	}
	if err := processChartDependency(chartFolder, "dep", "1.0.0", "https://example.com/charts"); err == nil {
		t.Fatalf("expected wrapped copyDependency error")
	}

	if err := DownloadDeps(filepath.Join(t.TempDir(), "missing-chart.yaml"), ""); err == nil {
		t.Fatalf("expected wrapped load chart error")
	}
}

package fluxhandler

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"helm.sh/helm/v3/pkg/cli"
)

func TestRepoURLAndClean(t *testing.T) {
	cases := []struct{ in, repoOut, cleanOut string }{
		{"https://charts.trueforge.org", "https:", "trueforge"},
		{"https://library-charts.trueforge.org/some/path", "https:", "library-charts.trueforge"},
		{"http://helm.example.com/foo", "http:", "example"},
		{"example.org", "example.org", "example"},
		{"charts.foo.bar/baz", "charts.foo.bar", "foo"},
	}

	for _, c := range cases {
		r := repoURL(c.in)
		if r != c.repoOut {
			t.Fatalf("repoURL(%q) = %q, want %q", c.in, r, c.repoOut)
		}
		cr := cleanRepoURL(c.in)
		if cr != c.cleanOut {
			t.Fatalf("cleanRepoURL(%q) = %q, want %q", c.in, cr, c.cleanOut)
		}
	}
}

func TestCreateValuesYAMLAndRemove(t *testing.T) {
	td := t.TempDir()
	file := filepath.Join(td, "vals.yaml")

	// createValuesYAML with empty map should create a file
	if err := createValuesYAML(map[string]interface{}{"a": "b"}, file); err != nil {
		t.Fatalf("createValuesYAML failed: %v", err)
	}
	if _, err := os.Stat(file); err != nil {
		t.Fatalf("expected file created: %v", err)
	}

	// removeFileIfExists should delete it
	if err := removeFileIfExists(file); err != nil {
		t.Fatalf("removeFileIfExists failed: %v", err)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatalf("expected file removed, stat returned: %v", err)
	}

	// removing non-existing file should not error
	if err := removeFileIfExists(filepath.Join(td, "nope")); err != nil {
		t.Fatalf("removeFileIfExists on missing file returned error: %v", err)
	}

	// ensure createValuesYAML writes YAML content
	file2 := filepath.Join(td, "vals2.yaml")
	if err := createValuesYAML(map[string]interface{}{"k": map[string]interface{}{"n": 1}}, file2); err != nil {
		t.Fatalf("createValuesYAML failed: %v", err)
	}
	b, err := ioutil.ReadFile(file2)
	if err != nil {
		t.Fatalf("read created file: %v", err)
	}
	if len(b) == 0 {
		t.Fatalf("created file empty")
	}
}

func TestNewDefaultRegistryClientAndNoOpLog(t *testing.T) {
	settings := cli.New()
	rc, err := newDefaultRegistryClient(false, settings)
	if err != nil {
		t.Fatalf("newDefaultRegistryClient(false) failed: %v", err)
	}
	if rc == nil {
		t.Fatalf("expected registry client")
	}

	rc2, err := newDefaultRegistryClient(true, settings)
	if err != nil {
		t.Fatalf("newDefaultRegistryClient(true) failed: %v", err)
	}
	if rc2 == nil {
		t.Fatalf("expected registry client for plainHTTP")
	}

	noOpLog("ignored %s", "message")
}

func TestUpdateHelmRepo(t *testing.T) {
	td := t.TempDir()
	t.Setenv("HELM_REPOSITORY_CACHE", filepath.Join(td, "cache"))
	t.Setenv("HELM_REPOSITORY_CONFIG", filepath.Join(td, "repositories.yaml"))

	index := `apiVersion: v1
entries:
  app:
    - version: 0.1.0
      urls:
        - https://example.invalid/app-0.1.0.tgz
`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/index.yaml" {
			_, _ = w.Write([]byte(index))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	if err := updateHelmRepo("myrepo", ts.URL, true); err != nil {
		t.Fatalf("updateHelmRepo failed: %v", err)
	}

	b, err := os.ReadFile(filepath.Join(td, "repositories.yaml"))
	if err != nil {
		t.Fatalf("failed to read repositories config: %v", err)
	}
	if len(b) == 0 {
		t.Fatalf("repositories file is empty")
	}
}

func TestHelmInstallDryRunAndRemoveFileError(t *testing.T) {
	if err := HelmInstall("", "", "", "", "", "", true, false, true); err != nil {
		t.Fatalf("HelmInstall dryRun should return nil, got: %v", err)
	}

	td := t.TempDir()
	dir := filepath.Join(td, "nonempty")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("write file in dir failed: %v", err)
	}
	if err := removeFileIfExists(dir); err == nil {
		t.Fatalf("expected removeFileIfExists to fail on non-empty directory")
	} else if err != nil && fmt.Sprint(err) == "" {
		t.Fatalf("expected non-empty error")
	}
}

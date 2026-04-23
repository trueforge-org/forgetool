package helmhandler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCleanRepoURL(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"https prefix", "https://example.com/path", "example"},
		{"http prefix", "http://example.com/path", "example"},
		{"charts prefix", "https://charts.example.com", "example"},
		{"helm prefix", "https://helm.example.com", "example"},
		{"charts and helm both stripped", "https://charts.helm.example.com", "example"},
		{"no protocol", "example.com", "example"},
		{"no path no dot", "myrepo", "myrepo"},
		{"trailing path segments", "https://repo.example.org/charts/stable", "repo.example"},
		{"http charts prefix", "http://charts.myrepo.io/index", "myrepo"},
		{"deep path", "https://example.com/a/b/c/d", "example"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := cleanRepoURL(c.in)
			if got != c.want {
				t.Errorf("cleanRepoURL(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestRepoURL(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"with path", "example.com/charts/stable", "example.com"},
		{"no path", "example.com", "example.com"},
		{"single slash", "host/path", "host"},
		{"empty string", "", ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := repoURL(c.in)
			if got != c.want {
				t.Errorf("repoURL(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestNoOpLogNoPanic(t *testing.T) {
	noOpLog("test %s %d", "arg", 42)
	noOpLog("")
	noOpLog("no args")
}

func TestCreateValuesYAMLContent(t *testing.T) {
	td := t.TempDir()
	file := filepath.Join(td, "values.yaml")

	vals := map[string]interface{}{
		"key": "value",
		"nested": map[string]interface{}{
			"inner": "data",
		},
	}

	if err := createValuesYAML(vals, file); err != nil {
		t.Fatalf("createValuesYAML failed: %v", err)
	}

	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "key") || !strings.Contains(content, "value") {
		t.Errorf("expected key/value in YAML output, got: %s", content)
	}
	if !strings.Contains(content, "nested") || !strings.Contains(content, "inner") {
		t.Errorf("expected nested/inner in YAML output, got: %s", content)
	}
}

func TestCreateValuesYAMLNilValues(t *testing.T) {
	td := t.TempDir()
	file := filepath.Join(td, "nil.yaml")

	if err := createValuesYAML(nil, file); err != nil {
		t.Fatalf("createValuesYAML(nil) failed: %v", err)
	}

	if _, err := os.Stat(file); err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}
}

func TestCreateValuesYAMLEmptyMap(t *testing.T) {
	td := t.TempDir()
	file := filepath.Join(td, "empty.yaml")

	if err := createValuesYAML(map[string]interface{}{}, file); err != nil {
		t.Fatalf("createValuesYAML(empty) failed: %v", err)
	}

	if _, err := os.Stat(file); err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}
}

func TestCreateValuesYAMLOverwrite(t *testing.T) {
	td := t.TempDir()
	file := filepath.Join(td, "overwrite.yaml")

	original := map[string]interface{}{"first": "original"}
	if err := createValuesYAML(original, file); err != nil {
		t.Fatalf("first createValuesYAML failed: %v", err)
	}

	replacement := map[string]interface{}{"second": "replaced"}
	if err := createValuesYAML(replacement, file); err != nil {
		t.Fatalf("second createValuesYAML failed: %v", err)
	}

	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	content := string(data)
	if strings.Contains(content, "first") || strings.Contains(content, "original") {
		t.Errorf("expected original content to be overwritten, got: %s", content)
	}
	if !strings.Contains(content, "second") || !strings.Contains(content, "replaced") {
		t.Errorf("expected replacement content, got: %s", content)
	}
}

func TestRemoveFileIfExistsFilePresent(t *testing.T) {
	td := t.TempDir()
	file := filepath.Join(td, "removeme.txt")

	if err := os.WriteFile(file, []byte("data"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	if err := removeFileIfExists(file); err != nil {
		t.Fatalf("removeFileIfExists failed: %v", err)
	}

	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatalf("expected file to be removed")
	}
}

func TestRemoveFileIfExistsFileMissing(t *testing.T) {
	td := t.TempDir()
	file := filepath.Join(td, "nonexistent.txt")

	if err := removeFileIfExists(file); err != nil {
		t.Fatalf("removeFileIfExists on missing file should not error, got: %v", err)
	}
}

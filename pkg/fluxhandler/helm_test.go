package fluxhandler

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
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

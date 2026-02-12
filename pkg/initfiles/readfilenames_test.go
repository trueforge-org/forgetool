package initfiles

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadFilenamesInDir(t *testing.T) {
	td := t.TempDir()
	files := []string{"a.txt", "b.yaml", "c"}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(td, f), []byte("x"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	got, err := readFilenamesInDir(td)
	if err != nil {
		t.Fatalf("readFilenamesInDir error: %v", err)
	}
	if len(got) != len(files) {
		t.Fatalf("unexpected file count: got %d want %d", len(got), len(files))
	}
}

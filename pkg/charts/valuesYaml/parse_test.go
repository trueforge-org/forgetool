package valuesYaml

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndSaveValuesFile(t *testing.T) {
	td := t.TempDir()
	f := filepath.Join(td, "values.yaml")
	content := "global:\n  stopAll: true\n"
	if err := os.WriteFile(f, []byte(content), 0644); err != nil {
		t.Fatalf("write sample: %v", err)
	}

	vf := NewValuesFile()
	if err := vf.LoadFromFile(f); err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}
	if !vf.Values.Global.StopAll {
		t.Fatalf("expected StopAll true")
	}

	// Save to another file
	out := filepath.Join(td, "out.yaml")
	if err := vf.SaveToFile(out); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("expected out file to exist: %v", err)
	}
}

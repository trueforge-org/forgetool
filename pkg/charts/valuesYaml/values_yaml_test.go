package valuesYaml

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUpdatevaluesFile(t *testing.T) {
	chartDir := t.TempDir()
	valuesPath := filepath.Join(chartDir, "values.yaml")
	if err := os.WriteFile(valuesPath, []byte("global:\n  stopAll: false\n"), 0644); err != nil {
		t.Fatalf("write values.yaml failed: %v", err)
	}

	if err := UpdatevaluesFile(valuesPath, ""); err != nil {
		t.Fatalf("UpdatevaluesFile(file path) failed: %v", err)
	}
	if err := UpdatevaluesFile(chartDir, ""); err != nil {
		t.Fatalf("UpdatevaluesFile(dir path) failed: %v", err)
	}

	if _, err := os.Stat(valuesPath); err != nil {
		t.Fatalf("values.yaml should still exist: %v", err)
	}
}

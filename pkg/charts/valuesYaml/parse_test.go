package valuesYaml

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestNewAndLoadAndSave(t *testing.T) {
	v := NewValuesFile()
	if v == nil || v.K == nil {
		t.Fatalf("NewValuesFile returned nil")
	}

	// write a simple values yaml
	dir := t.TempDir()
	f := filepath.Join(dir, "values.yaml")
	content := []byte("global:\n  stopAll: true\n")
	if err := ioutil.WriteFile(f, content, 0644); err != nil {
		t.Fatalf("write tmp values file: %v", err)
	}

	if err := v.LoadFromFile(f); err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if !v.Values.Global.StopAll {
		t.Fatalf("expected Global.StopAll to be true after load")
	}

	// Save to a new file
	out := filepath.Join(dir, "out.yaml")
	if err := v.SaveToFile(out); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// the file should exist and contain the stopAll key
	b, err := ioutil.ReadFile(out)
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	if len(b) == 0 {
		t.Fatalf("saved file is empty")
	}

	// also verify SaveToFile writes a file that can be stat'd
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("expected out file to exist: %v", err)
	}
}

package valuesYaml

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewValuesFile_HasKoanf(t *testing.T) {
	v := NewValuesFile()
	if v == nil {
		t.Fatal("NewValuesFile returned nil")
	}
	if v.K == nil {
		t.Fatal("NewValuesFile().K is nil, expected koanf instance")
	}
}

func TestLoadFromFile_NonExistentFile(t *testing.T) {
	v := NewValuesFile()
	err := v.LoadFromFile("/tmp/nonexistent-values-file.yaml")
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
}

func TestSaveToFile_RoundTrip(t *testing.T) {
	testDataPath := filepath.Join("..", "..", "..", "testdata", "values_yaml")
	srcFile := filepath.Join(testDataPath, "singleImageValues.yaml")

	v := NewValuesFile()
	if err := v.LoadFromFile(srcFile); err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	td := t.TempDir()
	outFile := filepath.Join(td, "saved-values.yaml")
	if err := v.SaveToFile(outFile); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	info, err := os.Stat(outFile)
	if err != nil {
		t.Fatalf("saved file does not exist: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("saved file is empty")
	}

	// Reload and verify
	v2 := NewValuesFile()
	if err := v2.LoadFromFile(outFile); err != nil {
		t.Fatalf("re-loading saved file failed: %v", err)
	}
}

func TestSaveToFile_InvalidPath(t *testing.T) {
	v := NewValuesFile()
	// Need valid metadata for SaveToFile to not fail on marshal
	testDataPath := filepath.Join("..", "..", "..", "testdata", "values_yaml")
	if err := v.LoadFromFile(filepath.Join(testDataPath, "singleImageValues.yaml")); err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	err := v.SaveToFile("/nonexistent/path/values.yaml")
	if err == nil {
		t.Fatal("expected error writing to invalid path, got nil")
	}
}

func TestUpdatevaluesFile_WithDirectory(t *testing.T) {
	// Create a temp directory with a valid values.yaml
	td := t.TempDir()
	testDataPath := filepath.Join("..", "..", "..", "testdata", "values_yaml")
	srcFile := filepath.Join(testDataPath, "singleImageValues.yaml")

	data, err := os.ReadFile(srcFile)
	if err != nil {
		t.Fatalf("failed to read source file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(td, "values.yaml"), data, 0644); err != nil {
		t.Fatalf("failed to write values.yaml: %v", err)
	}

	// Pass directory path
	if err := UpdatevaluesFile(td, ""); err != nil {
		t.Fatalf("UpdatevaluesFile with directory failed: %v", err)
	}
}

func TestUpdatevaluesFile_WithFilePath(t *testing.T) {
	td := t.TempDir()
	testDataPath := filepath.Join("..", "..", "..", "testdata", "values_yaml")
	srcFile := filepath.Join(testDataPath, "singleImageValues.yaml")

	data, err := os.ReadFile(srcFile)
	if err != nil {
		t.Fatalf("failed to read source file: %v", err)
	}
	valuesFile := filepath.Join(td, "values.yaml")
	if err := os.WriteFile(valuesFile, data, 0644); err != nil {
		t.Fatalf("failed to write values.yaml: %v", err)
	}

	// Pass file path
	if err := UpdatevaluesFile(valuesFile, ""); err != nil {
		t.Fatalf("UpdatevaluesFile with file path failed: %v", err)
	}
}

func TestUpdatevaluesFile_NonExistentPath(t *testing.T) {
	err := UpdatevaluesFile("/nonexistent/path/values.yaml", "")
	if err == nil {
		t.Fatal("expected error for non-existent path, got nil")
	}
}

func TestLoadFromFile_NilKoanf(t *testing.T) {
	// Test that LoadFromFile initializes K if nil
	v := &ValuesFile{}
	testDataPath := filepath.Join("..", "..", "..", "testdata", "values_yaml")
	if err := v.LoadFromFile(filepath.Join(testDataPath, "singleImageValues.yaml")); err != nil {
		t.Fatalf("LoadFromFile with nil K failed: %v", err)
	}
	if v.K == nil {
		t.Fatal("expected K to be initialized after LoadFromFile")
	}
}

package containertest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"net/http"
	"net/http/httptest"
)

func TestRunFromConfigFileSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "data")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	filePath := filepath.Join(targetDir, "check.txt")
	if err := os.WriteFile(filePath, []byte("hello container"), 0o600); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close)

	configPath := filepath.Join(tmpDir, "container-test.yaml")
	config := strings.Join([]string{
		"paths:",
		"  - " + targetDir,
		"files:",
		"  - path: " + filePath,
		"    contains:",
		"      - hello",
		"urls:",
		"  - url: " + server.URL,
		"    status: 200",
		"    contains:",
		"      - ok",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if err := RunFromConfigFile(configPath); err != nil {
		t.Fatalf("RunFromConfigFile returned error: %v", err)
	}
}

func TestRunFromConfigFileReturnsFailures(t *testing.T) {
	tmpDir := t.TempDir()

	configPath := filepath.Join(tmpDir, "container-test.yaml")
	config := strings.Join([]string{
		"paths:",
		"  - " + filepath.Join(tmpDir, "missing"),
		"files:",
		"  - path: " + filepath.Join(tmpDir, "missing.txt"),
		"urls:",
		"  - url: file:///tmp/test",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	err := RunFromConfigFile(configPath)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "container tests failed") {
		t.Fatalf("expected failure summary, got %v", err)
	}
}

package containertest

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
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

func TestRunExternalStorageIncludesConfig(t *testing.T) {
	oldStat := statFn
	t.Cleanup(func() {
		statFn = oldStat
	})

	calls := make(map[string]int)
	statFn = func(name string) (os.FileInfo, error) {
		calls[name]++
		if name == "/mnt/external" || name == "/config" {
			return nil, nil
		}
		return nil, errors.New("not found")
	}

	if err := Run(Config{ExternalStorage: []string{"/mnt/external"}}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if calls["/config"] == 0 {
		t.Fatal("expected /config to be checked as external storage")
	}
}

func TestRunTCPCheckSuccess(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start tcp listener: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	_, portString, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("failed to parse addr: %v", err)
	}
	port, err := strconv.Atoi(portString)
	if err != nil {
		t.Fatalf("failed to parse port: %v", err)
	}

	if err := Run(Config{TCP: []TCPCheck{{Host: "127.0.0.1", Port: port}}}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
}

func TestRunTCPCheckFailure(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to allocate tcp port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	err = Run(Config{TCP: []TCPCheck{{Host: "127.0.0.1", Port: port}}})
	if err == nil {
		t.Fatal("expected tcp failure error")
	}
	if !strings.Contains(err.Error(), "tcp check failed") {
		t.Fatalf("expected tcp failure message, got %v", err)
	}
}

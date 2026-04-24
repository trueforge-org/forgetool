package website

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseBakeVariables_ScannerError(t *testing.T) {
	root := t.TempDir()
	f := filepath.Join(root, "docker-bake.hcl")
	// Single line that exceeds bufio.Scanner's default token size (64 KiB)
	// so scanner.Err() returns bufio.ErrTooLong.
	long := strings.Repeat("a", 64*1024+16)
	if err := os.WriteFile(f, []byte(long), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := parseBakeVariables(f); err == nil {
		t.Fatal("expected scanner error")
	}
}

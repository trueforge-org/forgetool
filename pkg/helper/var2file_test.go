package helper

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVarToFile_CreatesNewFile(t *testing.T) {
	f := filepath.Join(t.TempDir(), "newfile.txt")
	if err := VarToFile(f, "hello world"); err != nil {
		t.Fatalf("VarToFile failed: %v", err)
	}
	b, err := os.ReadFile(f)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(b) != "hello world" {
		t.Fatalf("expected %q, got %q", "hello world", string(b))
	}
}

func TestVarToFile_DoesNotOverwriteExisting(t *testing.T) {
	f := filepath.Join(t.TempDir(), "existing.txt")
	if err := os.WriteFile(f, []byte("original"), 0644); err != nil {
		t.Fatalf("setup WriteFile failed: %v", err)
	}
	if err := VarToFile(f, "replacement"); err != nil {
		t.Fatalf("VarToFile should not error on existing file: %v", err)
	}
	b, err := os.ReadFile(f)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(b) != "original" {
		t.Fatalf("file should not be overwritten, got %q", string(b))
	}
}

func TestVarToFile_ErrorWhenDirNotExist(t *testing.T) {
	f := filepath.Join(t.TempDir(), "nodir", "subdir", "file.txt")
	err := VarToFile(f, "content")
	if err == nil {
		t.Fatalf("expected error when parent directory does not exist")
	}
}

func TestVarToFile_EmptyContent(t *testing.T) {
	f := filepath.Join(t.TempDir(), "empty.txt")
	if err := VarToFile(f, ""); err != nil {
		t.Fatalf("VarToFile with empty content failed: %v", err)
	}
	b, err := os.ReadFile(f)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if len(b) != 0 {
		t.Fatalf("expected empty file, got %q", string(b))
	}
}

func TestVarToFile_EmptyFilename(t *testing.T) {
	err := VarToFile("", "content")
	if err == nil {
		t.Fatalf("expected error for empty filename")
	}
}

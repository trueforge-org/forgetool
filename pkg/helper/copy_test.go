package helper

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReplaceDotInFilename(t *testing.T) {
	in := "some/DOTREPLACEfile"
	out := ReplaceDotInFilename(in)
	if out != "some/.file" {
		t.Fatalf("unexpected replace result: %s", out)
	}
}

func TestCopyFileAndDir(t *testing.T) {
	td := t.TempDir()
	srcDir := filepath.Join(td, "src")
	dstDir := filepath.Join(td, "dst")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	// create a file with DOTREPLACE in the name
	srcFile := filepath.Join(srcDir, "myDOTREPLACEfile.txt")
	if err := os.WriteFile(srcFile, []byte("hello"), 0644); err != nil {
		t.Fatalf("write src file: %v", err)
	}

	// test CopyDir
	if err := CopyDir(srcDir, dstDir, false); err != nil {
		t.Fatalf("CopyDir failed: %v", err)
	}
	// verify file copied with dot replaced
	dstFile := filepath.Join(dstDir, "my.file.txt")
	if _, err := os.Stat(dstFile); err != nil {
		t.Fatalf("expected copied file at %s: %v", dstFile, err)
	}

	// test CopyFile skip when exists and replaceExisting=false
	if err := CopyFile(srcFile, dstFile, false); err != nil {
		t.Fatalf("CopyFile should not error when skipping: %v", err)
	}

	// test CopyFile overwrite when replaceExisting=true
	if err := CopyFile(srcFile, dstFile, true); err != nil {
		t.Fatalf("CopyFile overwrite failed: %v", err)
	}
}

func TestCopyDirFiltered(t *testing.T) {
	td := t.TempDir()
	srcDir := filepath.Join(td, "srcf")
	dstDir := filepath.Join(td, "dstf")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("mkdir srcf: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "a.txt"), []byte("a"), 0644); err != nil {
		t.Fatalf("write a: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "b.log"), []byte("b"), 0644); err != nil {
		t.Fatalf("write b: %v", err)
	}

	if err := CopyDirFiltered(srcDir, dstDir, true, `.*`); err != nil {
		t.Fatalf("CopyDirFiltered failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dstDir, "a.txt")); err != nil {
		t.Fatalf("expected a.txt copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dstDir, "b.log")); err != nil {
		t.Fatalf("expected b.log copied: %v", err)
	}
}

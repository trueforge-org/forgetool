package helper

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyDirFiltered_RegexFilterCoverage(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Create source files
	os.WriteFile(filepath.Join(src, "include.yaml"), []byte("yaml"), 0644)
	os.WriteFile(filepath.Join(src, "exclude.txt"), []byte("text"), 0644)
	os.MkdirAll(filepath.Join(src, "subdir"), 0755)
	os.WriteFile(filepath.Join(src, "subdir", "nested.yaml"), []byte("nested"), 0644)

	// Filter to yaml files and directories matching the subdir
	err := CopyDirFiltered(src, dst, false, `(\.yaml$|^\.?$|^subdir$)`)
	if err != nil {
		t.Fatalf("CopyDirFiltered failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dst, "include.yaml")); os.IsNotExist(err) {
		t.Fatal("expected include.yaml to be copied")
	}
	if _, err := os.Stat(filepath.Join(dst, "exclude.txt")); !os.IsNotExist(err) {
		t.Fatal("expected exclude.txt to NOT be copied")
	}
}

func TestCopyDirFiltered_InvalidRegexCoverage(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	err := CopyDirFiltered(src, dst, false, "[invalid")
	if err == nil {
		t.Fatal("expected error for invalid regex filter")
	}
}

func TestCopyDir_WithDotReplace(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Create a file with DOTREPLACE in the name
	os.WriteFile(filepath.Join(src, "DOTREPLACEgitignore"), []byte("content"), 0644)

	err := CopyDir(src, dst, false)
	if err != nil {
		t.Fatalf("CopyDir failed: %v", err)
	}

	// Check that DOTREPLACE was replaced with "."
	if _, err := os.Stat(filepath.Join(dst, ".gitignore")); os.IsNotExist(err) {
		t.Fatal("expected .gitignore (from DOTREPLACE) to exist")
	}
}

func TestCopyDir_PreservesDirectoryStructure(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	os.MkdirAll(filepath.Join(src, "a", "b"), 0755)
	os.WriteFile(filepath.Join(src, "a", "b", "deep.txt"), []byte("deep"), 0644)

	err := CopyDir(src, dst, false)
	if err != nil {
		t.Fatalf("CopyDir failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dst, "a", "b", "deep.txt"))
	if err != nil {
		t.Fatalf("expected deep.txt to be copied: %v", err)
	}
	if string(data) != "deep" {
		t.Fatalf("unexpected content: %q", string(data))
	}
}

func TestCopyDir_ReplaceExistingCoverage(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	os.WriteFile(filepath.Join(src, "file.txt"), []byte("new content"), 0644)
	os.WriteFile(filepath.Join(dst, "file.txt"), []byte("old content"), 0644)

	// replaceExisting=false should keep old content
	err := CopyDir(src, dst, false)
	if err != nil {
		t.Fatalf("CopyDir failed: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dst, "file.txt"))
	if string(data) != "old content" {
		t.Fatalf("expected old content preserved, got %q", string(data))
	}

	// replaceExisting=true should overwrite
	err = CopyDir(src, dst, true)
	if err != nil {
		t.Fatalf("CopyDir with replace failed: %v", err)
	}
	data, _ = os.ReadFile(filepath.Join(dst, "file.txt"))
	if string(data) != "new content" {
		t.Fatalf("expected new content after replace, got %q", string(data))
	}
}

func TestCopyFile_SourceNotExists(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "out.txt")
	err := CopyFile("/tmp/nonexistent-copy-test.txt", dst, true)
	if err == nil {
		t.Fatal("expected error for non-existent source")
	}
}

func TestCopyDir_SourceNotExists(t *testing.T) {
	err := CopyDir("/tmp/nonexistent-dir-test", t.TempDir(), false)
	if err == nil {
		t.Fatal("expected error for non-existent source directory")
	}
}

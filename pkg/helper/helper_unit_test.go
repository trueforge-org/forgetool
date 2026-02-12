package helper

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUniqueNonEmptyElementsOfEmpty(t *testing.T) {
	result := UniqueNonEmptyElementsOf([]string{})
	if len(result) != 0 {
		t.Fatalf("expected empty slice, got %v", result)
	}
}

func TestUniqueNonEmptyElementsOfWithEmpties(t *testing.T) {
	result := UniqueNonEmptyElementsOf([]string{"", "", ""})
	if len(result) != 0 {
		t.Fatalf("expected empty slice, got %v", result)
	}
}

func TestUniqueNonEmptyElementsOfWithDuplicates(t *testing.T) {
	result := UniqueNonEmptyElementsOf([]string{"a", "b", "a", "c", "b"})
	if len(result) != 3 {
		t.Fatalf("expected 3 unique elements, got %d: %v", len(result), result)
	}
	seen := make(map[string]bool)
	for _, v := range result {
		if seen[v] {
			t.Fatalf("duplicate element found: %s", v)
		}
		seen[v] = true
	}
}

func TestUniqueNonEmptyElementsOfMixed(t *testing.T) {
	result := UniqueNonEmptyElementsOf([]string{"a", "", "b", "", "a"})
	if len(result) != 2 {
		t.Fatalf("expected 2 elements, got %d: %v", len(result), result)
	}
}

func TestReplaceDotInFilenameNoMatch(t *testing.T) {
	result := ReplaceDotInFilename("normalfile.txt")
	if result != "normalfile.txt" {
		t.Fatalf("expected unchanged filename, got %s", result)
	}
}

func TestReplaceDotInFilenameWithMatch(t *testing.T) {
	result := ReplaceDotInFilename("DOTREPLACEgitignore")
	if result != ".gitignore" {
		t.Fatalf("expected .gitignore, got %s", result)
	}
}

func TestReplaceDotInFilenameMultiple(t *testing.T) {
	result := ReplaceDotInFilename("DOTREPLACEenv/DOTREPLACElocal")
	if result != ".env/.local" {
		t.Fatalf("expected .env/.local, got %s", result)
	}
}

func TestCopyFileReplaceExisting(t *testing.T) {
	td := t.TempDir()
	src := filepath.Join(td, "src.txt")
	dst := filepath.Join(td, "dst.txt")

	os.WriteFile(src, []byte("source content"), 0644)
	os.WriteFile(dst, []byte("old content"), 0644)

	if err := CopyFile(src, dst, true); err != nil {
		t.Fatalf("CopyFile(replaceExisting=true) failed: %v", err)
	}

	data, _ := os.ReadFile(dst)
	if string(data) != "source content" {
		t.Fatalf("expected 'source content', got %q", string(data))
	}
}

func TestCopyFileNoReplaceExisting(t *testing.T) {
	td := t.TempDir()
	src := filepath.Join(td, "src.txt")
	dst := filepath.Join(td, "dst.txt")

	os.WriteFile(src, []byte("source content"), 0644)
	os.WriteFile(dst, []byte("old content"), 0644)

	if err := CopyFile(src, dst, false); err != nil {
		t.Fatalf("CopyFile(replaceExisting=false) failed: %v", err)
	}

	data, _ := os.ReadFile(dst)
	if string(data) != "old content" {
		t.Fatalf("expected 'old content' (unchanged), got %q", string(data))
	}
}

func TestCopyDirFilteredWithPattern(t *testing.T) {
	td := t.TempDir()
	src := filepath.Join(td, "src")
	dst := filepath.Join(td, "dst")

	os.MkdirAll(filepath.Join(src, "sub"), os.ModePerm)
	os.WriteFile(filepath.Join(src, "file.yaml"), []byte("yaml"), 0644)
	os.WriteFile(filepath.Join(src, "file.txt"), []byte("txt"), 0644)
	os.WriteFile(filepath.Join(src, "sub", "nested.yaml"), []byte("nested"), 0644)

	// The regex filter matches against relative paths (including directories).
	// Use a pattern that matches yaml files and also matches directory entries.
	if err := CopyDirFiltered(src, dst, false, `(\.yaml$|^\.?$|^sub$)`); err != nil {
		t.Fatalf("CopyDirFiltered failed: %v", err)
	}

	// yaml file should exist
	if _, err := os.Stat(filepath.Join(dst, "file.yaml")); os.IsNotExist(err) {
		t.Fatalf("expected file.yaml to be copied")
	}
	// txt file should not exist (filtered out)
	if _, err := os.Stat(filepath.Join(dst, "file.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected file.txt to not be copied")
	}
}

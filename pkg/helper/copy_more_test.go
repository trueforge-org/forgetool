package helper

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyDir_EmptyDirectory(t *testing.T) {
	td := t.TempDir()
	srcDir := filepath.Join(td, "empty_src")
	dstDir := filepath.Join(td, "empty_dst")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}

	if err := CopyDir(srcDir, dstDir, false); err != nil {
		t.Fatalf("CopyDir on empty dir failed: %v", err)
	}

	info, err := os.Stat(dstDir)
	if err != nil {
		t.Fatalf("expected dst dir to exist: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("expected dst to be a directory")
	}
}

func TestCopyDir_NestedDirectories(t *testing.T) {
	td := t.TempDir()
	srcDir := filepath.Join(td, "nested_src")
	dstDir := filepath.Join(td, "nested_dst")

	// Create nested structure: src/a/b/c/file.txt
	nested := filepath.Join(srcDir, "a", "b", "c")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nested, "file.txt"), []byte("deep"), 0644); err != nil {
		t.Fatalf("write nested file: %v", err)
	}
	// Also add a file at intermediate level
	if err := os.WriteFile(filepath.Join(srcDir, "a", "top.txt"), []byte("top"), 0644); err != nil {
		t.Fatalf("write top file: %v", err)
	}

	if err := CopyDir(srcDir, dstDir, false); err != nil {
		t.Fatalf("CopyDir nested failed: %v", err)
	}

	// Verify deep file
	data, err := os.ReadFile(filepath.Join(dstDir, "a", "b", "c", "file.txt"))
	if err != nil {
		t.Fatalf("expected deep file: %v", err)
	}
	if string(data) != "deep" {
		t.Fatalf("unexpected content: %s", data)
	}

	// Verify intermediate file
	data, err = os.ReadFile(filepath.Join(dstDir, "a", "top.txt"))
	if err != nil {
		t.Fatalf("expected top file: %v", err)
	}
	if string(data) != "top" {
		t.Fatalf("unexpected content: %s", data)
	}
}

func TestCopyDir_ReplaceExisting(t *testing.T) {
	td := t.TempDir()
	srcDir := filepath.Join(td, "src")
	dstDir := filepath.Join(td, "dst")

	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "f.txt"), []byte("new"), 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	// Create dst with existing file
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		t.Fatalf("mkdir dst: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dstDir, "f.txt"), []byte("old"), 0644); err != nil {
		t.Fatalf("write dst: %v", err)
	}

	// With replaceExisting=false, file should not be overwritten
	if err := CopyDir(srcDir, dstDir, false); err != nil {
		t.Fatalf("CopyDir no-replace failed: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dstDir, "f.txt"))
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(data) != "old" {
		t.Fatalf("expected file to not be replaced, got: %s", data)
	}

	// With replaceExisting=true, file should be overwritten
	if err := CopyDir(srcDir, dstDir, true); err != nil {
		t.Fatalf("CopyDir replace failed: %v", err)
	}
	data, err = os.ReadFile(filepath.Join(dstDir, "f.txt"))
	if err != nil {
		t.Fatalf("read dst after replace: %v", err)
	}
	if string(data) != "new" {
		t.Fatalf("expected file to be replaced, got: %s", data)
	}
}

func TestCopyFile_NonExistentSource(t *testing.T) {
	td := t.TempDir()
	err := CopyFile(filepath.Join(td, "nonexistent.txt"), filepath.Join(td, "dst.txt"), true)
	if err == nil {
		t.Fatalf("expected error when source does not exist")
	}
}

func TestCopyFile_DestinationDirMissing(t *testing.T) {
	td := t.TempDir()
	srcFile := filepath.Join(td, "src.txt")
	if err := os.WriteFile(srcFile, []byte("data"), 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	// Destination in a non-existent directory
	dstFile := filepath.Join(td, "no", "such", "dir", "dst.txt")
	err := CopyFile(srcFile, dstFile, true)
	if err == nil {
		t.Fatalf("expected error when destination directory does not exist")
	}
}

func TestCopyDirFiltered_SelectiveFilter(t *testing.T) {
	td := t.TempDir()
	srcDir := filepath.Join(td, "src")
	dstDir := filepath.Join(td, "dst")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "keep.txt"), []byte("keep"), 0644); err != nil {
		t.Fatalf("write keep: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "skip.log"), []byte("skip"), 0644); err != nil {
		t.Fatalf("write skip: %v", err)
	}

	// Filter must also match "." (root relPath) to avoid SkipDir on root.
	// Use a pattern that matches root "." and .txt files but not .log files.
	if err := CopyDirFiltered(srcDir, dstDir, false, `(^\.$|\.txt$)`); err != nil {
		t.Fatalf("CopyDirFiltered failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dstDir, "keep.txt")); err != nil {
		t.Fatalf("expected keep.txt to be copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dstDir, "skip.log")); !os.IsNotExist(err) {
		t.Fatalf("expected skip.log to NOT be copied")
	}
}

func TestCopyDirFiltered_InvalidRegex(t *testing.T) {
	td := t.TempDir()
	srcDir := filepath.Join(td, "src")
	dstDir := filepath.Join(td, "dst")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}

	err := CopyDirFiltered(srcDir, dstDir, false, "[invalid")
	if err == nil {
		t.Fatalf("expected error for invalid regex filter")
	}
}

func TestCopyDirFiltered_DirectorySkippedByFilter(t *testing.T) {
	td := t.TempDir()
	srcDir := filepath.Join(td, "src")
	dstDir := filepath.Join(td, "dst")

	// Create a subdirectory with a file inside
	subDir := filepath.Join(srcDir, "skipme")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "inner.txt"), []byte("inner"), 0644); err != nil {
		t.Fatalf("write inner: %v", err)
	}
	// Create a file at root level that matches
	if err := os.WriteFile(filepath.Join(srcDir, "root.txt"), []byte("root"), 0644); err != nil {
		t.Fatalf("write root: %v", err)
	}

	// Filter matches "." (root dir) and "root.txt" but not "skipme" directory.
	// When "skipme" doesn't match, it returns SkipDir and skips the whole subtree.
	if err := CopyDirFiltered(srcDir, dstDir, false, `(^\.$|^root\.txt$)`); err != nil {
		t.Fatalf("CopyDirFiltered failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dstDir, "root.txt")); err != nil {
		t.Fatalf("expected root.txt to be copied: %v", err)
	}
	// The "skipme" directory should be skipped by the filter
	if _, err := os.Stat(filepath.Join(dstDir, "skipme", "inner.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected skipme/inner.txt to NOT be copied")
	}
}

func TestCopyDir_NonExistentSource(t *testing.T) {
	td := t.TempDir()
	err := CopyDir(filepath.Join(td, "nonexistent"), filepath.Join(td, "dst"), false)
	if err == nil {
		t.Fatalf("expected error when source directory does not exist")
	}
}

func TestReplaceDotInFilename_NoDotReplace(t *testing.T) {
	in := "normal/path/file.txt"
	out := ReplaceDotInFilename(in)
	if out != in {
		t.Fatalf("expected no change, got: %s", out)
	}
}

func TestReplaceDotInFilename_MultipleDotReplace(t *testing.T) {
	in := "DOTREPLACEaDOTREPLACEb"
	out := ReplaceDotInFilename(in)
	if out != ".a.b" {
		t.Fatalf("expected '.a.b', got: %s", out)
	}
}

func TestCopyFile_SkipExistingPreservesContent(t *testing.T) {
	td := t.TempDir()
	srcFile := filepath.Join(td, "src.txt")
	dstFile := filepath.Join(td, "dst.txt")

	if err := os.WriteFile(srcFile, []byte("source"), 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	if err := os.WriteFile(dstFile, []byte("original"), 0644); err != nil {
		t.Fatalf("write dst: %v", err)
	}

	// replaceExisting=false should skip and preserve original content
	if err := CopyFile(srcFile, dstFile, false); err != nil {
		t.Fatalf("CopyFile skip existing failed: %v", err)
	}

	data, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(data) != "original" {
		t.Fatalf("expected original content preserved, got: %s", data)
	}
}

func TestCopyFile_OverwriteUpdatesContent(t *testing.T) {
	td := t.TempDir()
	srcFile := filepath.Join(td, "src.txt")
	dstFile := filepath.Join(td, "dst.txt")

	if err := os.WriteFile(srcFile, []byte("updated"), 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	if err := os.WriteFile(dstFile, []byte("original"), 0644); err != nil {
		t.Fatalf("write dst: %v", err)
	}

	if err := CopyFile(srcFile, dstFile, true); err != nil {
		t.Fatalf("CopyFile overwrite failed: %v", err)
	}

	data, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(data) != "updated" {
		t.Fatalf("expected updated content, got: %s", data)
	}
}

func TestCopyDir_DotReplaceInNames(t *testing.T) {
	td := t.TempDir()
	// Use subdirectories whose names don't contain "DOTREPLACE" to avoid
	// the replacement affecting the temp dir prefix in the full path.
	srcDir := filepath.Join(td, "src")
	dstDir := filepath.Join(td, "dst")

	// Create a directory with DOTREPLACE in its name
	dotDir := filepath.Join(srcDir, "DOTREPLACEconfig")
	if err := os.MkdirAll(dotDir, 0755); err != nil {
		t.Fatalf("mkdir dotdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dotDir, "DOTREPLACEsettings"), []byte("cfg"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := CopyDir(srcDir, dstDir, false); err != nil {
		t.Fatalf("CopyDir failed: %v", err)
	}

	// Verify DOTREPLACE was replaced in both dir and file names.
	// ReplaceDotInFilename operates on the full destPath, so the dest
	// prefix must not contain "DOTREPLACE" for the paths to be predictable.
	data, err := os.ReadFile(filepath.Join(dstDir, ".config", ".settings"))
	if err != nil {
		t.Fatalf("expected .config/.settings: %v", err)
	}
	if string(data) != "cfg" {
		t.Fatalf("unexpected content: %s", data)
	}
}

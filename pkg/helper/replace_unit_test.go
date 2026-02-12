package helper

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReplaceInFileNoMatch(t *testing.T) {
	td := t.TempDir()
	p := filepath.Join(td, "test.txt")
	os.WriteFile(p, []byte("hello world"), 0644)

	if err := ReplaceInFile(p, "NOTFOUND", "NEW"); err != nil {
		t.Fatalf("ReplaceInFile error: %v", err)
	}

	data, _ := os.ReadFile(p)
	if string(data) != "hello world" {
		t.Fatalf("content should be unchanged, got %q", string(data))
	}
}

func TestReplaceInFileNonExistent(t *testing.T) {
	err := ReplaceInFile("/tmp/nonexistent_replace_test_12345.txt", "old", "new")
	if err == nil {
		t.Fatalf("expected error for non-existent file")
	}
}

func TestReplaceContentBetweenLinesMissingMarkers(t *testing.T) {
	td := t.TempDir()
	target := filepath.Join(td, "target.txt")
	source := filepath.Join(td, "source.txt")

	os.WriteFile(target, []byte("line1\nline2\nline3\n"), 0644)
	os.WriteFile(source, []byte("replacement\n"), 0644)

	// Call with markers that don't exist in target
	if err := ReplaceContentBetweenLines(target, source, "MISSING_FROM", "MISSING_TILL"); err != nil {
		t.Fatalf("ReplaceContentBetweenLines error: %v", err)
	}

	data, _ := os.ReadFile(target)
	// Content should remain largely the same since markers are not found
	if !strings.Contains(string(data), "line1") {
		t.Fatalf("original content should be preserved when markers not found")
	}
}

func TestReplaceContentBetweenLinesNonExistentSource(t *testing.T) {
	td := t.TempDir()
	target := filepath.Join(td, "target.txt")
	os.WriteFile(target, []byte("content\n"), 0644)

	err := ReplaceContentBetweenLines(target, "/tmp/nonexistent_source_12345.txt", "FROM", "TILL")
	if err == nil {
		t.Fatalf("expected error for non-existent source file")
	}
}

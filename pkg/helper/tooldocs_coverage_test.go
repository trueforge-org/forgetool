package helper

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestYamlNewEncoder_CreatesEncoder(t *testing.T) {
	var buf bytes.Buffer
	enc := YamlNewEncoder(&buf)
	if enc == nil {
		t.Fatal("YamlNewEncoder returned nil")
	}
	if enc.writer == nil {
		t.Fatal("encoder writer is nil")
	}
}

func TestEncoder_EncodeWithoutIndent(t *testing.T) {
	var buf bytes.Buffer
	enc := YamlNewEncoder(&buf)

	data := map[string]string{"key": "value"}
	if err := enc.Encode(data); err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "key: value") {
		t.Fatalf("expected 'key: value' in output, got %q", out)
	}
}

func TestEncoder_EncodeWithIndent(t *testing.T) {
	var buf bytes.Buffer
	enc := YamlNewEncoder(&buf)
	enc.SetIndent(4)

	data := map[string]string{"key": "value"}
	if err := enc.Encode(data); err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "    key: value") {
		t.Fatalf("expected 4-space indented output, got %q", out)
	}
}

func TestEncoder_SetIndent(t *testing.T) {
	var buf bytes.Buffer
	enc := YamlNewEncoder(&buf)
	enc.SetIndent(2)
	if enc.indent != 2 {
		t.Fatalf("expected indent 2, got %d", enc.indent)
	}
}

func TestDeterminePaths_NoUnderscore(t *testing.T) {
	sub, file := determinePaths("simple.md")
	if sub != "" {
		t.Fatalf("expected empty subdir, got %q", sub)
	}
	if file != "simple.md" {
		t.Fatalf("expected filename 'simple.md', got %q", file)
	}
}

func TestDeterminePaths_SameNameAsDir(t *testing.T) {
	sub, file := determinePaths("talos_talos.md")
	if sub != "talos" {
		t.Fatalf("expected subdir 'talos', got %q", sub)
	}
	if file != "index.md" {
		t.Fatalf("expected 'index.md' for same-name match, got %q", file)
	}
}

func TestDeterminePaths_DifferentNameFromDir(t *testing.T) {
	sub, file := determinePaths("talos_apply.md")
	if sub != "talos" {
		t.Fatalf("expected subdir 'talos', got %q", sub)
	}
	if file != "apply.md" {
		t.Fatalf("expected 'apply.md', got %q", file)
	}
}

func TestAddYamlTitle_EmptyContent(t *testing.T) {
	out := addYamlTitle([]byte(""), false)
	s := string(out)
	if !strings.Contains(s, "---\ntitle: ") {
		t.Fatalf("expected yaml front matter even for empty content, got %q", s)
	}
}

func TestAddYamlTitle_NoSeeAlso(t *testing.T) {
	content := "## forgetool test\nLine1\nLine2\n"
	out := addYamlTitle([]byte(content), false)
	s := string(out)
	if !strings.Contains(s, "Line1") || !strings.Contains(s, "Line2") {
		t.Fatalf("expected all lines when no SEE ALSO, got %q", s)
	}
}

func TestAddYamlTitle_ForgetoolTitle(t *testing.T) {
	content := "## forgetool\nSome text\n"
	out := addYamlTitle([]byte(content), false)
	s := string(out)
	if !strings.Contains(s, "title: command") {
		t.Fatalf("expected title 'command' for ## forgetool, got %q", s)
	}
}

func TestRenameForgetoolToIndex_NoFile(t *testing.T) {
	td := t.TempDir()
	err := renameForgetoolToIndex(td)
	if err != nil {
		t.Fatalf("expected nil error when forgetool.md doesn't exist, got: %v", err)
	}
}

func TestRenameForgetoolToIndex_Success(t *testing.T) {
	td := t.TempDir()
	os.WriteFile(filepath.Join(td, "forgetool.md"), []byte("content"), 0644)

	if err := renameForgetoolToIndex(td); err != nil {
		t.Fatalf("renameForgetoolToIndex failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(td, "index.md")); os.IsNotExist(err) {
		t.Fatal("expected index.md to exist after rename")
	}
	if _, err := os.Stat(filepath.Join(td, "forgetool.md")); !os.IsNotExist(err) {
		t.Fatal("expected forgetool.md to not exist after rename")
	}
}

func TestMoveMatchingFilesToSubdirs_NoMatchingDir(t *testing.T) {
	td := t.TempDir()
	os.WriteFile(filepath.Join(td, "orphan.md"), []byte("content"), 0644)

	if err := moveMatchingFilesToSubdirs(td); err != nil {
		t.Fatalf("moveMatchingFilesToSubdirs failed: %v", err)
	}
	// File should remain in place since there's no matching subdirectory
	if _, err := os.Stat(filepath.Join(td, "orphan.md")); os.IsNotExist(err) {
		t.Fatal("expected orphan.md to remain")
	}
}

func TestMoveMatchingFilesToSubdirs_WithMatchingDir(t *testing.T) {
	td := t.TempDir()
	os.MkdirAll(filepath.Join(td, "charts"), 0755)
	os.WriteFile(filepath.Join(td, "charts.md"), []byte("content"), 0644)

	if err := moveMatchingFilesToSubdirs(td); err != nil {
		t.Fatalf("moveMatchingFilesToSubdirs failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(td, "charts", "index.md")); os.IsNotExist(err) {
		t.Fatal("expected charts/index.md after move")
	}
}

func TestMoveMatchingFilesToSubdirs_InvalidDir(t *testing.T) {
	err := moveMatchingFilesToSubdirs("/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for non-existent directory")
	}
}

func TestProcessFiles_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	outDir := t.TempDir()

	if err := processFiles(tmpDir, outDir); err != nil {
		t.Fatalf("processFiles on empty dir should succeed, got: %v", err)
	}
}

func TestProcessFiles_SkipsDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	outDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)

	if err := processFiles(tmpDir, outDir); err != nil {
		t.Fatalf("processFiles should skip directories, got: %v", err)
	}
}

func TestProcessFiles_InvalidDir(t *testing.T) {
	err := processFiles("/nonexistent/input", t.TempDir())
	if err == nil {
		t.Fatal("expected error for non-existent input dir")
	}
}

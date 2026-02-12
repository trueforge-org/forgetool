package helper

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeterminePaths(t *testing.T) {
	sub, file := determinePaths("charts_readme.md")
	if sub != "charts" || file != "readme.md" {
		t.Fatalf("unexpected split: %s %s", sub, file)
	}
	sub, file = determinePaths("charts_charts.md")
	if sub != "charts" || file != "index.md" {
		t.Fatalf("expected index mapping, got %s %s", sub, file)
	}
	sub, file = determinePaths("plain.md")
	if sub != "" || file != "plain.md" {
		t.Fatalf("expected no split for plain file, got %s %s", sub, file)
	}
}

func TestAddYamlTitle(t *testing.T) {
	primary := addYamlTitle([]byte("## forgetool\ntext\n"), true)
	if !strings.Contains(string(primary), "title: commands") {
		t.Fatalf("expected primary index title, got: %s", string(primary))
	}

	content := "## forgetool charts\nline1\n### SEE ALSO\nline2\n"
	out := addYamlTitle([]byte(content), false)
	s := string(out)
	if !strings.Contains(s, "title: charts") {
		t.Fatalf("expected derived title, got: %s", s)
	}
	if strings.Contains(s, "SEE ALSO") || strings.Contains(s, "line2") {
		t.Fatalf("expected SEE ALSO section removed: %s", s)
	}
}

func TestWriteMoveRenameAndToolDocs(t *testing.T) {
	td := t.TempDir()
	filePath := filepath.Join(td, "out", "a.md")
	entryFile := writeDirEntry(t, td, "seed.md", "x")
	if err := writeToFile(filePath, []byte("hello"), entryFile); err != nil {
		t.Fatalf("writeToFile failed: %v", err)
	}
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("expected written file: %v", err)
	}

	outDir := filepath.Join(td, "docs")
	if err := os.MkdirAll(filepath.Join(outDir, "alpha"), 0755); err != nil {
		t.Fatalf("mkdir alpha dir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "alpha.md"), []byte("x"), 0644); err != nil {
		t.Fatalf("write alpha.md failed: %v", err)
	}
	if err := moveMatchingFilesToSubdirs(outDir); err != nil {
		t.Fatalf("moveMatchingFilesToSubdirs failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "alpha", "index.md")); err != nil {
		t.Fatalf("expected moved index.md: %v", err)
	}

	if err := os.WriteFile(filepath.Join(outDir, "forgetool.md"), []byte("x"), 0644); err != nil {
		t.Fatalf("write forgetool.md failed: %v", err)
	}
	if err := renameForgetoolToIndex(outDir); err != nil {
		t.Fatalf("renameForgetoolToIndex failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "index.md")); err != nil {
		t.Fatalf("expected index.md after rename: %v", err)
	}

	tmpDir := filepath.Join(td, "tmp")
	finalOut := filepath.Join(td, "final")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		t.Fatalf("mkdir tmp failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "forgetool_forgetool.md"), []byte("## forgetool\ncmd\n"), 0644); err != nil {
		t.Fatalf("write tmp cmd doc failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "forgetool_networking_nginx.md"), []byte("## forgetool networking nginx\ntext\n"), 0644); err != nil {
		t.Fatalf("write tmp nested doc failed: %v", err)
	}

	ToolDocs(tmpDir, finalOut)
	if _, err := os.Stat(filepath.Join(finalOut, "index.md")); err != nil {
		t.Fatalf("expected final root index.md: %v", err)
	}
	if _, err := os.Stat(filepath.Join(finalOut, "networking", "nginx.md")); err != nil {
		t.Fatalf("expected transformed nested doc: %v", err)
	}
}

func writeDirEntry(t *testing.T, dir, name, content string) os.DirEntry {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("write seed file failed: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir failed: %v", err)
	}
	for _, e := range entries {
		if e.Name() == name {
			return e
		}
	}
	t.Fatalf("entry %s not found", name)
	return nil
}

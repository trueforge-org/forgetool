package helmignore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateHelmIgnore(t *testing.T) {
	tplRoot := t.TempDir()
	chartDir := t.TempDir()
	chartPath := filepath.Join(chartDir, "Chart.yaml")

	if err := os.MkdirAll(filepath.Join(tplRoot, "templates"), os.ModePerm); err != nil {
		t.Fatalf("mkdir templates failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tplRoot, "templates", "helmignore.tpl"), []byte("*.tmp\n"), 0644); err != nil {
		t.Fatalf("write template failed: %v", err)
	}
	if err := os.WriteFile(chartPath, []byte("apiVersion: v2\n"), 0644); err != nil {
		t.Fatalf("write chart path failed: %v", err)
	}

	if err := GenerateHelmIgnore(tplRoot, chartPath); err != nil {
		t.Fatalf("GenerateHelmIgnore failed: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(chartDir, ".helmignore"))
	if err != nil {
		t.Fatalf("read generated .helmignore failed: %v", err)
	}
	if string(b) != "*.tmp\n" {
		t.Fatalf("unexpected .helmignore content: %q", string(b))
	}
}

func TestGenerateHelmIgnore_MissingTemplate(t *testing.T) {
	err := GenerateHelmIgnore(t.TempDir(), filepath.Join(t.TempDir(), "Chart.yaml"))
	if err == nil {
		t.Fatalf("expected error when template is missing")
	}
}

func TestGenerateHelmIgnore_WriteError(t *testing.T) {
	tplRoot := t.TempDir()
	chartDir := t.TempDir()
	chartPath := filepath.Join(chartDir, "Chart.yaml")

	if err := os.MkdirAll(filepath.Join(tplRoot, "templates"), os.ModePerm); err != nil {
		t.Fatalf("mkdir templates failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tplRoot, "templates", "helmignore.tpl"), []byte("*.tmp\n"), 0644); err != nil {
		t.Fatalf("write template failed: %v", err)
	}
	if err := os.WriteFile(chartPath, []byte("apiVersion: v2\n"), 0644); err != nil {
		t.Fatalf("write chart failed: %v", err)
	}

	target := filepath.Join(chartDir, ".helmignore")
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatalf("mkdir target dir failed: %v", err)
	}

	if err := GenerateHelmIgnore(tplRoot, chartPath); err == nil {
		t.Fatalf("expected write error when target path is a directory")
	}
}

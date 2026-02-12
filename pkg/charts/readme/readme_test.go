package readme

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateReadme(t *testing.T) {
	tplRoot := t.TempDir()
	chartDir := t.TempDir()
	chartPath := filepath.Join(chartDir, "Chart.yaml")

	if err := os.MkdirAll(filepath.Join(tplRoot, "templates"), os.ModePerm); err != nil {
		t.Fatalf("mkdir templates failed: %v", err)
	}
	templateContent := "# CHARTPLACEHOLDER\nTrain: TRAINPLACEHOLDER\n"
	if err := os.WriteFile(filepath.Join(tplRoot, "templates", "README.md.tpl"), []byte(templateContent), 0644); err != nil {
		t.Fatalf("write template failed: %v", err)
	}
	if err := os.WriteFile(chartPath, []byte("apiVersion: v2\n"), 0644); err != nil {
		t.Fatalf("write chart path failed: %v", err)
	}

	if err := GenerateReadme(tplRoot, chartPath, "mychart", "stable"); err != nil {
		t.Fatalf("GenerateReadme failed: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(chartDir, "README.md"))
	if err != nil {
		t.Fatalf("read generated README failed: %v", err)
	}
	content := string(b)
	if !strings.Contains(content, "mychart") || !strings.Contains(content, "stable") {
		t.Fatalf("expected placeholders replaced, got: %q", content)
	}
}

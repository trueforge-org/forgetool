package chartFile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateChartFile(t *testing.T) {
	td := t.TempDir()
	root := filepath.Join(td, "root")
	chartDir := filepath.Join(root, "charts", "stable", "myapp")
	if err := os.MkdirAll(chartDir, 0755); err != nil {
		t.Fatalf("mkdir chart dir failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "templates"), 0755); err != nil {
		t.Fatalf("mkdir templates failed: %v", err)
	}

	chartPath := filepath.Join(chartDir, "Chart.yaml")
	chartYAML := `apiVersion: v2
name: myapp
version: 1.2.3
annotations: {}
sources: []
`
	if err := os.WriteFile(chartPath, []byte(chartYAML), 0644); err != nil {
		t.Fatalf("write Chart.yaml failed: %v", err)
	}
	valuesYAML := `image:
  repository: docker.io/library/nginx
  tag: 1.27.0
`
	if err := os.WriteFile(filepath.Join(chartDir, "values.yaml"), []byte(valuesYAML), 0644); err != nil {
		t.Fatalf("write values.yaml failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "templates", "README.md.tpl"), []byte("# CHARTPLACEHOLDER\nTRAINPLACEHOLDER\n"), 0644); err != nil {
		t.Fatalf("write README template failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "templates", "helmignore.tpl"), []byte("*.tmp\n"), 0644); err != nil {
		t.Fatalf("write helmignore template failed: %v", err)
	}

	if err := UpdateChartFile(chartPath, "patch"); err != nil {
		t.Fatalf("UpdateChartFile failed: %v", err)
	}

	hc := NewHelmChart()
	if err := hc.LoadFromFile(chartPath); err != nil {
		t.Fatalf("load updated chart failed: %v", err)
	}
	if hc.Metadata.Version != "1.2.4" {
		t.Fatalf("expected bumped version 1.2.4, got %s", hc.Metadata.Version)
	}
	if hc.Metadata.AppVersion == "" {
		t.Fatalf("expected appVersion to be set from image")
	}
	if hc.Metadata.Annotations["truecharts.org/train"] != "stable" {
		t.Fatalf("expected train annotation stable, got %s", hc.Metadata.Annotations["truecharts.org/train"])
	}
	if len(hc.Metadata.Sources) == 0 {
		t.Fatalf("expected non-empty sources")
	}

	readmePath := filepath.Join(chartDir, "README.md")
	if b, err := os.ReadFile(readmePath); err != nil {
		t.Fatalf("expected README generated: %v", err)
	} else if !strings.Contains(string(b), "myapp") {
		t.Fatalf("expected README content to include chart name")
	}
	if _, err := os.Stat(filepath.Join(chartDir, ".helmignore")); err != nil {
		t.Fatalf("expected .helmignore generated: %v", err)
	}
}

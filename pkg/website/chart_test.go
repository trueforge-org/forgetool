package website

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleChartYaml = `
apiVersion: v2
name: myapp
version: 9.0.1
appVersion: 1.0.0
description: An example chart
sources:
  - https://example.com/a
  - https://example.com/b
`

func TestProcessChart_FullFlow(t *testing.T) {
	root := t.TempDir()
	train, chart := "stable", "myapp"
	chartDir := filepath.Join(root, "charts", train, chart)
	writeFile(t, filepath.Join(chartDir, "Chart.yaml"), sampleChartYaml)
	writeFile(t, filepath.Join(chartDir, "icon.webp"), "icon")
	writeFile(t, filepath.Join(chartDir, "icon-small.webp"), "icon-small")
	writeFile(t, filepath.Join(chartDir, "screenshots", "shot.png"), "shot")
	writeFile(t, filepath.Join(chartDir, "docs", "install.md"), "---\ntitle: Install\n---\nbody\n")
	writeFile(t, filepath.Join(chartDir, "README.md"), "L1\nL2\nL3\n## Section\ncontent\n")

	docsBase := filepath.Join(root, "website", "truecharts", "src", "content", "docs", "charts")
	// Pre-existing safe doc.
	writeFile(t, filepath.Join(docsBase, train, chart, "CHANGELOG.md"), "history\n")
	// A stale copy in another train should be removed (chart moved trains).
	writeFile(t, filepath.Join(docsBase, "incubator", chart, "stale.md"), "stale\n")

	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	if err := ProcessChart(ChartOptions{Train: train, Chart: chart}); err != nil {
		t.Fatalf("ProcessChart: %v", err)
	}

	// Stale chart in other train removed.
	if _, err := os.Stat(filepath.Join(docsBase, "incubator", chart)); !os.IsNotExist(err) {
		t.Fatalf("stale incubator copy should be gone: err=%v", err)
	}
	// CHANGELOG restored.
	if got := readFile(t, filepath.Join(docsBase, train, chart, "CHANGELOG.md")); got != "history\n" {
		t.Fatalf("CHANGELOG not restored: %q", got)
	}
	// Index file rendered.
	idx := readFile(t, filepath.Join(docsBase, train, chart, "index.md"))
	if !strings.HasPrefix(idx, "---\ntitle: myapp\n---") {
		t.Fatalf("index missing front matter: %q", idx)
	}
	for _, want := range []string{
		"Version-9.0.1",
		"AppVersion-1.0.0",
		"An example chart",
		"## Chart Sources",
		"https://example.com/a",
		"## Available Documentation",
		"[**Install**](./install)",
		"## Readme",
		"### Section",
	} {
		if !strings.Contains(idx, want) {
			t.Fatalf("index missing %q\nfull:\n%s", want, idx)
		}
	}
	// Icons + screenshots copied.
	if got := readFile(t, filepath.Join(root, "website", "truecharts", "public", "img", "hotlink-ok", "chart-icons", chart+".webp")); got != "icon" {
		t.Fatalf("icon mismatch: %q", got)
	}
	if got := readFile(t, filepath.Join(root, "website", "truecharts", "public", "img", "hotlink-ok", "chart-icons-small", chart+".webp")); got != "icon-small" {
		t.Fatalf("small icon mismatch: %q", got)
	}
	if got := readFile(t, filepath.Join(root, "website", "truecharts", "public", "img", "hotlink-ok", "chart-screenshots", chart, "shot.png")); got != "shot" {
		t.Fatalf("screenshot mismatch: %q", got)
	}
}

func TestProcessChart_MissingChartIsSkip(t *testing.T) {
	root := t.TempDir()
	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	if err := ProcessChart(ChartOptions{Train: "stable", Chart: "ghost"}); err != nil {
		t.Fatalf("expected nil for missing chart, got %v", err)
	}
}

func TestProcessChart_RequiresFields(t *testing.T) {
	if err := ProcessChart(ChartOptions{}); err == nil {
		t.Fatal("expected error for empty fields")
	}
	if err := ProcessChart(ChartOptions{Train: "stable"}); err == nil {
		t.Fatal("expected error for missing chart")
	}
}

func TestDiscoverCharts(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "charts", "stable", "alpha", "Chart.yaml"), "name: alpha")
	writeFile(t, filepath.Join(root, "charts", "incubator", "beta", "Chart.yaml"), "name: beta")
	if err := os.MkdirAll(filepath.Join(root, "charts", "stable", "no-yaml"), 0o755); err != nil {
		t.Fatal(err)
	}
	pairs, err := DiscoverCharts(filepath.Join(root, "charts"))
	if err != nil {
		t.Fatal(err)
	}
	if len(pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %v", pairs)
	}
}

package website_test

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/website"
)

// copyTree mirrors a fixture tree into dst. It is intentionally small and
// duplicated here (instead of importing helper.CopyDir) so the integration
// tests do not depend on the production copy implementation.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	})
	if err != nil {
		t.Fatalf("copyTree %s -> %s: %v", src, dst, err)
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

// TestIntegration_ContainerDocs exercises the full container docs build
// pipeline against the testdata/website/containers fixture. It mirrors what the
// legacy container-docs.sh does on a real run.
func TestIntegration_ContainerDocs(t *testing.T) {
	root := t.TempDir()
	fixture, err := filepath.Abs(filepath.Join("..", "..", "testdata", "website", "containers"))
	if err != nil {
		t.Fatal(err)
	}
	copyTree(t, fixture, root)

	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	apps, err := website.DiscoverApps("apps")
	if err != nil {
		t.Fatalf("DiscoverApps: %v", err)
	}
	if len(apps) != 1 || apps[0] != "demoapp" {
		t.Fatalf("unexpected apps: %v", apps)
	}

	for _, app := range apps {
		if err := website.ProcessApp(website.ContainerOptions{
			App:                 app,
			ComposeTemplatePath: "templates/docker-compose.yaml.tmpl",
		}); err != nil {
			t.Fatalf("ProcessApp(%s): %v", app, err)
		}
	}

	docsDir := filepath.Join(root, "website", "containerforge", "src", "content", "docs", "containers", "demoapp")
	idx := mustRead(t, filepath.Join(docsDir, "index.md"))
	for _, want := range []string{
		"title: demoapp",
		"Version-2.5.0",
		// License dashes must be escaped to `--` for the shields.io URL.
		"License-Apache--2.0",
		"https://example.com/demoapp",
		"[**Install**](./install)",
		"[**Configuration**](./configuration)",
		"## Readme",
		"### First Section",
	} {
		if !strings.Contains(idx, want) {
			t.Fatalf("index.md missing %q\nfull:\n%s", want, idx)
		}
	}

	// Old-style heading was promoted to front matter on disk.
	cfg := mustRead(t, filepath.Join(docsDir, "configuration.md"))
	if !strings.HasPrefix(cfg, "---\ntitle: Configuration\n---\n") {
		t.Fatalf("configuration.md not promoted: %q", cfg)
	}

	// Assets present in the website public tree.
	publicBase := filepath.Join(root, "website", "containerforge", "public", "img", "hotlink-ok")
	if got := mustRead(t, filepath.Join(publicBase, "container-icons", "demoapp.webp")); !strings.Contains(got, "fake-icon-bytes") {
		t.Fatalf("icon mismatch: %q", got)
	}
	if got := mustRead(t, filepath.Join(publicBase, "container-icons-small", "demoapp.webp")); !strings.Contains(got, "fake-icon-small-bytes") {
		t.Fatalf("small icon mismatch: %q", got)
	}
	if got := mustRead(t, filepath.Join(publicBase, "container-screenshots", "demoapp", "main.png")); !strings.Contains(got, "fake-screenshot") {
		t.Fatalf("screenshot mismatch: %q", got)
	}
}

// TestIntegration_ChartDocs exercises the full chart docs build pipeline
// against the testdata/website/charts fixture. It mirrors what the legacy
// chart-docs.sh does on a real run.
func TestIntegration_ChartDocs(t *testing.T) {
	root := t.TempDir()
	fixture, err := filepath.Abs(filepath.Join("..", "..", "testdata", "website", "charts"))
	if err != nil {
		t.Fatal(err)
	}
	copyTree(t, fixture, root)

	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	pairs, err := website.DiscoverCharts("charts")
	if err != nil {
		t.Fatalf("DiscoverCharts: %v", err)
	}
	if len(pairs) != 1 || pairs[0][0] != "stable" || pairs[0][1] != "demochart" {
		t.Fatalf("unexpected pairs: %v", pairs)
	}

	for _, p := range pairs {
		if err := website.ProcessChart(website.ChartOptions{Train: p[0], Chart: p[1]}); err != nil {
			t.Fatalf("ProcessChart(%s/%s): %v", p[0], p[1], err)
		}
	}

	docsDir := filepath.Join(root, "website", "truecharts", "src", "content", "docs", "charts", "stable", "demochart")
	idx := mustRead(t, filepath.Join(docsDir, "index.md"))
	for _, want := range []string{
		"title: demochart",
		"Version-3.4.5",
		"AppVersion-1.0.0",
		"A demo chart for testing website docs generation.",
		"## Chart Sources",
		"https://example.com/demochart",
		"## Available Documentation",
		"[**Install**](./install)",
		"[**Upgrading**](./upgrading)",
		"## Readme",
		"### Introduction",
		"### Values",
	} {
		if !strings.Contains(idx, want) {
			t.Fatalf("index.md missing %q\nfull:\n%s", want, idx)
		}
	}

	upg := mustRead(t, filepath.Join(docsDir, "upgrading.md"))
	if !strings.HasPrefix(upg, "---\ntitle: Upgrading\n---\n") {
		t.Fatalf("upgrading.md not promoted: %q", upg)
	}

	publicBase := filepath.Join(root, "website", "truecharts", "public", "img", "hotlink-ok")
	if got := mustRead(t, filepath.Join(publicBase, "chart-icons", "demochart.webp")); !strings.Contains(got, "chart-icon-bytes") {
		t.Fatalf("icon mismatch: %q", got)
	}
	if got := mustRead(t, filepath.Join(publicBase, "chart-icons-small", "demochart.webp")); !strings.Contains(got, "chart-icon-small-bytes") {
		t.Fatalf("small icon mismatch: %q", got)
	}
	if got := mustRead(t, filepath.Join(publicBase, "chart-screenshots", "demochart", "main.png")); !strings.Contains(got, "chart-screenshot") {
		t.Fatalf("screenshot mismatch: %q", got)
	}
}

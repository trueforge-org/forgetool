package website

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadChartMeta_ReadError(t *testing.T) {
	if _, err := readChartMeta(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Fatal("expected read error")
	}
}

func TestReadChartMeta_ParseError(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "Chart.yaml")
	writeFile(t, f, ":\n\tnot yaml")
	if _, err := readChartMeta(f); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestRemoveChartFromAllTrains_MissingDirIsNoop(t *testing.T) {
	if err := removeChartFromAllTrains(filepath.Join(t.TempDir(), "nope"), "x"); err != nil {
		t.Fatalf("expected nil for missing dir, got %v", err)
	}
}

func TestRemoveChartFromAllTrains_SkipsNonDirEntries(t *testing.T) {
	root := t.TempDir()
	// A file at the train level should be skipped.
	writeFile(t, filepath.Join(root, "not-a-train"), "x")
	writeFile(t, filepath.Join(root, "stable", "myapp", "x.md"), "y")
	if err := removeChartFromAllTrains(root, "myapp"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "stable", "myapp")); !os.IsNotExist(err) {
		t.Fatalf("myapp should be gone: %v", err)
	}
	// Sibling file untouched.
	if _, err := os.Stat(filepath.Join(root, "not-a-train")); err != nil {
		t.Fatalf("expected file preserved: %v", err)
	}
}

func TestDiscoverCharts_ReadDirError(t *testing.T) {
	if _, err := DiscoverCharts(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("expected error")
	}
}

func TestDiscoverCharts_SkipsFiles(t *testing.T) {
	root := t.TempDir()
	// File at the train level.
	writeFile(t, filepath.Join(root, "stray-file"), "x")
	// Train dir with both a chart subdir and a stray file.
	writeFile(t, filepath.Join(root, "stable", "stray-file"), "x")
	writeFile(t, filepath.Join(root, "stable", "alpha", "Chart.yaml"), "name: alpha")
	pairs, err := DiscoverCharts(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(pairs) != 1 || pairs[0][1] != "alpha" {
		t.Fatalf("unexpected pairs: %v", pairs)
	}
}

func TestDiscoverCharts_TrainReadDirError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses perms")
	}
	root := t.TempDir()
	tdir := filepath.Join(root, "stable")
	if err := os.MkdirAll(tdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(tdir, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(tdir, 0o755) })
	if _, err := DiscoverCharts(root); err == nil {
		t.Fatal("expected ReadDir error on unreadable train")
	}
}

func TestProcessChart_StatNonNotExistError(t *testing.T) {
	root := t.TempDir()
	// Make Chart.yaml's parent path traverse a regular file → ENOTDIR
	makeFileAsDir(t, filepath.Join(root, "charts", "stable", "myapp"))
	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	if err := ProcessChart(ChartOptions{Train: "stable", Chart: "myapp"}); err == nil {
		t.Fatal("expected stat error")
	}
}

func TestProcessChart_ReadmeBodyError(t *testing.T) {
	// Make README.md a directory so readReadmeBody errors out.
	root := t.TempDir()
	chartDir := filepath.Join(root, "charts", "stable", "myapp")
	writeFile(t, filepath.Join(chartDir, "Chart.yaml"), sampleChartYaml)
	if err := os.MkdirAll(filepath.Join(chartDir, "README.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	err := ProcessChart(ChartOptions{Train: "stable", Chart: "myapp"})
	if err == nil || !strings.Contains(err.Error(), "is a directory") && !strings.Contains(err.Error(), "directory") {
		// Any error suffices — README path is a directory.
		if err == nil {
			t.Fatal("expected error")
		}
	}
}

// chartPathsForTest builds a chartPaths struct rooted at root.
func chartPathsForTest(root, train, chart string) chartPaths {
	opts := ChartOptions{Train: train, Chart: chart, ChartsDir: filepath.Join(root, "charts"), WebsiteDir: filepath.Join(root, "website")}
	return opts.paths()
}

func TestCopyChartAssets_DocsCopyError(t *testing.T) {
	root := t.TempDir()
	train, chart := "stable", "myapp"
	p := chartPathsForTest(root, train, chart)
	// docs source has a file; pre-create dst/<file> as dir so CopyFile fails.
	writeFile(t, filepath.Join(p.docsSrc, "a.md"), "x")
	if err := os.MkdirAll(filepath.Join(p.docsDir, "a.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyChartAssets(p); err == nil {
		t.Fatal("expected docs copy error")
	}
}

func TestCopyChartAssets_IconCopyError(t *testing.T) {
	root := t.TempDir()
	train, chart := "stable", "myapp"
	p := chartPathsForTest(root, train, chart)
	writeFile(t, p.icon, "icon")
	makeFileAsDir(t, filepath.Join(p.iconsDir, "blocker"))
	// Force MkdirAll for icons dir to fail by making its parent a file.
	_ = os.RemoveAll(p.iconsDir)
	makeFileAsDir(t, p.iconsDir)
	if err := copyChartAssets(p); err == nil {
		t.Fatal("expected icon copy error")
	}
}

func TestCopyChartAssets_IconSmallCopyError(t *testing.T) {
	root := t.TempDir()
	train, chart := "stable", "myapp"
	p := chartPathsForTest(root, train, chart)
	writeFile(t, p.iconSmall, "icon-small")
	makeFileAsDir(t, p.iconsSmallDir)
	if err := copyChartAssets(p); err == nil {
		t.Fatal("expected icon-small copy error")
	}
}

func TestCopyChartAssets_ScreenshotsCopyError(t *testing.T) {
	root := t.TempDir()
	train, chart := "stable", "myapp"
	p := chartPathsForTest(root, train, chart)
	writeFile(t, filepath.Join(p.screenshots, "a.png"), "shot")
	if err := os.MkdirAll(filepath.Join(p.screenshotsDir, "a.png"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyChartAssets(p); err == nil {
		t.Fatal("expected screenshot copy error")
	}
}

func TestProcessChartIndex_MetaError(t *testing.T) {
	root := t.TempDir()
	p := chartPathsForTest(root, "stable", "myapp")
	writeFile(t, p.chartYaml, ":\n\tbad")
	if err := processChartIndex(ChartOptions{Train: "stable", Chart: "myapp"}, p); err == nil {
		t.Fatal("expected meta error")
	}
}

func TestProcessChartIndex_LinksError(t *testing.T) {
	root := t.TempDir()
	p := chartPathsForTest(root, "stable", "myapp")
	writeFile(t, p.chartYaml, sampleChartYaml)
	makeFileAsDir(t, filepath.Join(p.docsDir, "blocker"))
	// docsDir crosses a regular file, ReadDir returns ENOTDIR
	if err := processChartIndex(ChartOptions{Train: "stable", Chart: "myapp"}, chartPaths{
		chartYaml: p.chartYaml,
		docsDir:   filepath.Join(p.docsDir, "blocker", "x"),
		readme:    p.readme,
		indexFile: p.indexFile,
	}); err == nil {
		t.Fatal("expected links error")
	}
}

func TestProcessChartIndex_ReadmeBodyError(t *testing.T) {
	root := t.TempDir()
	p := chartPathsForTest(root, "stable", "myapp")
	writeFile(t, p.chartYaml, sampleChartYaml)
	if err := os.MkdirAll(p.readme, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := processChartIndex(ChartOptions{Train: "stable", Chart: "myapp"}, p); err == nil {
		t.Fatal("expected readme error")
	}
}

func TestProcessChartIndex_WriteIndexError(t *testing.T) {
	root := t.TempDir()
	p := chartPathsForTest(root, "stable", "myapp")
	writeFile(t, p.chartYaml, sampleChartYaml)
	// Make indexFile a directory so WriteFile fails.
	if err := os.MkdirAll(p.indexFile, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := processChartIndex(ChartOptions{Train: "stable", Chart: "myapp"}, p); err == nil {
		t.Fatal("expected write error")
	}
}

func TestProcessChartIndex_MkdirIndexError(t *testing.T) {
	root := t.TempDir()
	p := chartPathsForTest(root, "stable", "myapp")
	writeFile(t, p.chartYaml, sampleChartYaml)
	// Make parent of indexFile a regular file so MkdirAll fails.
	makeFileAsDir(t, filepath.Dir(p.indexFile))
	if err := processChartIndex(ChartOptions{Train: "stable", Chart: "myapp"}, p); err == nil {
		t.Fatal("expected mkdir error")
	}
}

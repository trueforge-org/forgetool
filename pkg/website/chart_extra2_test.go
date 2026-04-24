package website

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveChartFromAllTrains_RemoveAllError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses perms")
	}
	root := t.TempDir()
	// Train dir contains chart subdir with a file; chmod the train dir to 0
	// so RemoveAll fails (cannot traverse).
	train := filepath.Join(root, "stable")
	chart := filepath.Join(train, "myapp")
	writeFile(t, filepath.Join(chart, "x.md"), "y")
	if err := os.Chmod(train, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(train, 0o755) })
	if err := removeChartFromAllTrains(root, "myapp"); err == nil {
		t.Fatal("expected remove error")
	}
}

// Build a chartPaths suitable for hitting late branches of processChartIndex
// without needing a real on-disk docsDir/readme.
func minimalChartPaths(root string) chartPaths {
	return chartPaths{
		chartYaml: filepath.Join(root, "Chart.yaml"),
		docsDir:   filepath.Join(root, "missing-docs"),
		readme:    filepath.Join(root, "missing-readme"),
		indexFile: filepath.Join(root, "out", "index.md"),
	}
}

func TestProcessChartIndex_ReadmeNoTrailingNewline(t *testing.T) {
	root := t.TempDir()
	p := minimalChartPaths(root)
	writeFile(t, p.chartYaml, sampleChartYaml)
	// README without trailing newline. readReadmeBody drops the first 3
	// lines, so include 3 throwaway lines and a final body line with NO "\n".
	writeFile(t, p.readme, "L1\nL2\nL3\nbody-without-newline")
	if err := processChartIndex(ChartOptions{Train: "stable", Chart: "myapp"}, p); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(p.indexFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "body-without-newline") {
		t.Fatalf("readme not embedded: %q", got)
	}
}

func TestProcessChartIndex_MkdirIndexErrorIsolated(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses perms")
	}
	root := t.TempDir()
	// docsDir & readme are absent (so collectDocsLinks/readReadmeBody return
	// empty without error). Make the indexFile parent unwritable.
	parent := filepath.Join(root, "ro")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(parent, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o755) })
	p := chartPaths{
		chartYaml: filepath.Join(root, "Chart.yaml"),
		docsDir:   filepath.Join(root, "no-docs"),
		readme:    filepath.Join(root, "no-readme"),
		indexFile: filepath.Join(parent, "out", "index.md"),
	}
	writeFile(t, p.chartYaml, sampleChartYaml)
	if err := processChartIndex(ChartOptions{Train: "stable", Chart: "myapp"}, p); err == nil {
		t.Fatal("expected mkdir error")
	}
}

// chartProcessSetup creates a chart layout under root and chdirs into root.
// It returns the train/chart pair.
func chartProcessSetup(t *testing.T, root string) (string, string) {
	t.Helper()
	train, chart := "stable", "myapp"
	chartDir := filepath.Join(root, "charts", train, chart)
	writeFile(t, filepath.Join(chartDir, "Chart.yaml"), sampleChartYaml)
	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	return train, chart
}

func TestProcessChart_RemoveChartFromAllTrainsError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses perms")
	}
	root := t.TempDir()
	train, chart := chartProcessSetup(t, root)
	docsBase := filepath.Join("website", "truecharts", "src", "content", "docs", "charts")
	// Pre-create a sibling train dir under docsBase containing the chart;
	// then chmod the sibling train dir to 0 so RemoveAll inside
	// removeChartFromAllTrains fails.
	siblingTrain := filepath.Join(docsBase, "incubator")
	writeFile(t, filepath.Join(siblingTrain, chart, "x.md"), "y")
	if err := os.Chmod(siblingTrain, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(siblingTrain, 0o755) })
	if err := ProcessChart(ChartOptions{Train: train, Chart: chart}); err == nil {
		t.Fatal("expected error from removeChartFromAllTrains")
	}
}

func TestProcessChart_MkdirDocsDirError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses perms")
	}
	root := t.TempDir()
	train, chart := chartProcessSetup(t, root)
	docsBaseTrain := filepath.Join("website", "truecharts", "src", "content", "docs", "charts", train)
	if err := os.MkdirAll(docsBaseTrain, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(docsBaseTrain, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(docsBaseTrain, 0o755) })
	err := ProcessChart(ChartOptions{Train: train, Chart: chart})
	if err == nil {
		t.Fatal("expected mkdir error")
	}
}

func TestProcessChart_CopyAssetsError(t *testing.T) {
	root := t.TempDir()
	train, chart := chartProcessSetup(t, root)
	chartDir := filepath.Join(root, "charts", train, chart)
	// Provide an icon at source plus a destination icons-dir entry that
	// already exists as a non-empty directory of the same name; copyFileIfExists
	// → helper.CopyFile fails when the destination is a directory.
	writeFile(t, filepath.Join(chartDir, "icon.webp"), "icon")
	iconsDir := filepath.Join("website", "truecharts", "public", "img", "hotlink-ok", "chart-icons")
	writeFile(t, filepath.Join(iconsDir, chart+".webp", "blocker"), "x")
	if err := ProcessChart(ChartOptions{Train: train, Chart: chart}); err == nil {
		t.Fatal("expected copyChartAssets error")
	}
}

func TestProcessChart_RestoreSafeDocsError(t *testing.T) {
	root := t.TempDir()
	train, chart := chartProcessSetup(t, root)
	// Pre-existing CHANGELOG to be saved to tmp.
	docsDir := filepath.Join("website", "truecharts", "src", "content", "docs", "charts", train, chart)
	writeFile(t, filepath.Join(docsDir, "CHANGELOG.md"), "history\n")
	// Block restore by pre-creating a non-empty directory at the destination
	// path so the rename of the saved file back fails.
	// keepDocsSafe will move the file into tmpwebsite/.../CHANGELOG.md.
	// During restoreSafeDocs we want the rename target to be a non-empty dir.
	// We need to set this up AFTER keepDocsSafe runs but before restoreSafeDocs.
	// Workaround: create the blocking directory now; it will be wiped by
	// resetDir/MkdirAll in copyChartAssets... Actually for charts there's no
	// resetDir, just os.MkdirAll on docsDir. So a pre-existing
	// docsDir/CHANGELOG.md (file) is moved out by keepDocsSafe, then
	// docsDir is recreated/populated, and restoreSafeDocs renames the saved
	// CHANGELOG back. To make the rename fail we instead create the target
	// path as a non-empty directory inside the freshly-built docs tree.
	// Easiest: place a docs/CHANGELOG.md/inside file so when copyChartAssets
	// copies the docs source it materializes CHANGELOG.md as a directory at
	// the destination, blocking the rename.
	chartDir := filepath.Join(root, "charts", train, chart)
	writeFile(t, filepath.Join(chartDir, "docs", "CHANGELOG.md", "child"), "x")
	if err := ProcessChart(ChartOptions{Train: train, Chart: chart}); err == nil {
		t.Fatal("expected restoreSafeDocs error")
	}
}

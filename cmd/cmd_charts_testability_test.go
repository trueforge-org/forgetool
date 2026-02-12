package cmd

import (
	"errors"
	"io/fs"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/trueforge-org/forgetool/pkg/charts/changelog"
	"github.com/trueforge-org/forgetool/pkg/charts/website"
	"github.com/trueforge-org/forgetool/pkg/helper"
)

func TestRunChartsBumpUsesArgs(t *testing.T) {
	old := chartsBumpVersion
	t.Cleanup(func() { chartsBumpVersion = old })

	var gotVersion, gotKind string
	chartsBumpVersion = func(version string, kind string) error {
		gotVersion = version
		gotKind = kind
		return nil
	}

	if err := runChartsBump([]string{"1.2.3", "patch"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotVersion != "1.2.3" || gotKind != "patch" {
		t.Fatalf("unexpected args passed: %s %s", gotVersion, gotKind)
	}
}

func TestRunChartsDepsCallsLoadAndWalk(t *testing.T) {
	oldLoad := chartsDepsLoadGPGKey
	oldWalk := chartsDepsWalkCharts
	t.Cleanup(func() {
		chartsDepsLoadGPGKey = oldLoad
		chartsDepsWalkCharts = oldWalk
	})

	loaded := false
	chartsDepsLoadGPGKey = func() error {
		loaded = true
		return nil
	}

	var gotPaths []string
	var gotBump string
	var gotMode helper.WalkMode
	chartsDepsWalkCharts = func(paths []string, _ func(string, string) error, bump string, mode helper.WalkMode) error {
		gotPaths = append([]string{}, paths...)
		gotBump = bump
		gotMode = mode
		return nil
	}

	args := []string{"./charts/a", "./charts/b"}
	if err := runChartsDeps(args); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !loaded {
		t.Fatalf("expected gpg key load to be called")
	}
	if !reflect.DeepEqual(gotPaths, args) {
		t.Fatalf("unexpected paths: %#v", gotPaths)
	}
	if gotBump != "" {
		t.Fatalf("expected empty bump, got %q", gotBump)
	}
	if gotMode != helper.SyncMode {
		t.Fatalf("expected sync mode, got %v", gotMode)
	}
}

func TestRunChartsDepsReturnsLoadError(t *testing.T) {
	oldLoad := chartsDepsLoadGPGKey
	t.Cleanup(func() { chartsDepsLoadGPGKey = oldLoad })

	want := errors.New("load failed")
	chartsDepsLoadGPGKey = func() error { return want }

	err := runChartsDeps(nil)
	if !errors.Is(err, want) {
		t.Fatalf("expected load error, got %v", err)
	}
}

func TestParseChartsGenMetaArgs(t *testing.T) {
	bump, paths := parseChartsGenMetaArgs([]string{"minor", "./charts"})
	if bump != "minor" {
		t.Fatalf("expected minor bump, got %q", bump)
	}
	if len(paths) != 1 || paths[0] != "./charts" {
		t.Fatalf("unexpected paths: %#v", paths)
	}

	bump, paths = parseChartsGenMetaArgs([]string{"./charts"})
	if bump != "" {
		t.Fatalf("expected empty bump, got %q", bump)
	}
	if len(paths) != 1 || paths[0] != "./charts" {
		t.Fatalf("unexpected paths without bump: %#v", paths)
	}
}

func TestRunChartsGenMetaUsesParsedValues(t *testing.T) {
	oldWalk := chartsGenMetaWalkCharts
	t.Cleanup(func() { chartsGenMetaWalkCharts = oldWalk })

	var gotPaths []string
	var gotBump string
	var gotMode helper.WalkMode
	chartsGenMetaWalkCharts = func(paths []string, _ func(string, string) error, bump string, mode helper.WalkMode) error {
		gotPaths = append([]string{}, paths...)
		gotBump = bump
		gotMode = mode
		return nil
	}

	if err := runChartsGenMeta([]string{"major", "./charts"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotBump != "major" {
		t.Fatalf("expected major bump, got %q", gotBump)
	}
	if len(gotPaths) != 1 || gotPaths[0] != "./charts" {
		t.Fatalf("unexpected paths: %#v", gotPaths)
	}
	if gotMode != helper.SyncMode {
		t.Fatalf("expected sync mode, got %v", gotMode)
	}
}

func TestRunChartsGenChartListErrorPaths(t *testing.T) {
	oldWalk := chartsGenChartListWalkCharts2
	oldFactory := chartsGenChartListOptionsFactory
	oldGet := chartsGenChartListGetChartData
	oldWrite := chartsGenChartListWrite
	t.Cleanup(func() {
		chartsGenChartListWalkCharts2 = oldWalk
		chartsGenChartListOptionsFactory = oldFactory
		chartsGenChartListGetChartData = oldGet
		chartsGenChartListWrite = oldWrite
	})

	chartsGenChartListOptionsFactory = func() *website.ChartListOptions {
		return &website.ChartListOptions{OutputPath: "ignored"}
	}
	chartsGenChartListGetChartData = func(_ *website.ChartListOptions) fs.WalkDirFunc {
		return func(string, fs.DirEntry, error) error { return nil }
	}

	walkErr := errors.New("walk failed")
	chartsGenChartListWalkCharts2 = func([]string, fs.WalkDirFunc, helper.WalkMode) error {
		return walkErr
	}
	if err := runChartsGenChartList([]string{"./charts"}); err == nil || !strings.Contains(err.Error(), "failed to generate chart list") {
		t.Fatalf("expected wrapped walk error, got %v", err)
	}

	chartsGenChartListWalkCharts2 = func([]string, fs.WalkDirFunc, helper.WalkMode) error { return nil }
	writeErr := errors.New("write failed")
	chartsGenChartListWrite = func(_ *website.ChartListOptions) error { return writeErr }
	if err := runChartsGenChartList([]string{"./charts"}); err == nil || !strings.Contains(err.Error(), "failed to write chart list") {
		t.Fatalf("expected wrapped write error, got %v", err)
	}
}

func TestRunChartsGenChartListSuccess(t *testing.T) {
	oldWalk := chartsGenChartListWalkCharts2
	oldFactory := chartsGenChartListOptionsFactory
	oldGet := chartsGenChartListGetChartData
	oldWrite := chartsGenChartListWrite
	t.Cleanup(func() {
		chartsGenChartListWalkCharts2 = oldWalk
		chartsGenChartListOptionsFactory = oldFactory
		chartsGenChartListGetChartData = oldGet
		chartsGenChartListWrite = oldWrite
	})

	calledWalk := false
	calledWrite := false
	chartsGenChartListOptionsFactory = func() *website.ChartListOptions {
		return &website.ChartListOptions{OutputPath: "ignored"}
	}
	chartsGenChartListGetChartData = func(_ *website.ChartListOptions) fs.WalkDirFunc {
		return func(string, fs.DirEntry, error) error { return nil }
	}
	chartsGenChartListWalkCharts2 = func([]string, fs.WalkDirFunc, helper.WalkMode) error {
		calledWalk = true
		return nil
	}
	chartsGenChartListWrite = func(_ *website.ChartListOptions) error {
		calledWrite = true
		return nil
	}

	if err := runChartsGenChartList([]string{"./charts"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !calledWalk || !calledWrite {
		t.Fatalf("expected both walk and write to be called")
	}
}

func TestRunChartsGenChangelog(t *testing.T) {
	oldGenerate := chartsGenChangelogGenerate
	oldRender := chartsGenChangelogRender
	t.Cleanup(func() {
		chartsGenChangelogGenerate = oldGenerate
		chartsGenChangelogRender = oldRender
	})

	if err := runChartsGenChangelog([]string{"only", "two"}); err == nil {
		t.Fatalf("expected missing-args error")
	}

	chartsGenChangelogGenerate = func(opts *changelog.ChangelogOptions) error {
		if opts.RepoPath != "repo" || opts.TemplatePath != "tmpl" || opts.ChartsDir != "charts" {
			t.Fatalf("unexpected options: %#v", opts)
		}
		return nil
	}
	chartsGenChangelogRender = func(*changelog.ChangelogOptions) error { return nil }
	if err := runChartsGenChangelog([]string{"repo", "tmpl", "charts"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	genErr := errors.New("gen failed")
	chartsGenChangelogGenerate = func(*changelog.ChangelogOptions) error { return genErr }
	if err := runChartsGenChangelog([]string{"repo", "tmpl", "charts"}); err == nil || !strings.Contains(err.Error(), "generate changelog") {
		t.Fatalf("expected wrapped generate error, got %v", err)
	}

	chartsGenChangelogGenerate = func(*changelog.ChangelogOptions) error { return nil }
	rendErr := errors.New("render failed")
	chartsGenChangelogRender = func(*changelog.ChangelogOptions) error { return rendErr }
	if err := runChartsGenChangelog([]string{"repo", "tmpl", "charts"}); err == nil || !strings.Contains(err.Error(), "render changelog") {
		t.Fatalf("expected wrapped render error, got %v", err)
	}
}

func TestRunChartsTagClean(t *testing.T) {
	old := chartsTagClean
	t.Cleanup(func() { chartsTagClean = old })

	got := ""
	chartsTagClean = func(tag string) error {
		got = tag
		return nil
	}
	if err := runChartsTagClean([]string{"tag@sha"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "tag@sha" {
		t.Fatalf("unexpected tag: %q", got)
	}
}

func TestRunGenToolDocs(t *testing.T) {
	oldMkdir := genToolDocsMkdirAll
	oldMarkdown := genToolDocsMarkdown
	oldWriter := genToolDocsWriter
	t.Cleanup(func() {
		genToolDocsMkdirAll = oldMkdir
		genToolDocsMarkdown = oldMarkdown
		genToolDocsWriter = oldWriter
	})

	mkdirCalled := false
	genToolDocsMkdirAll = func(path string, _ fs.FileMode) error {
		mkdirCalled = path == helper.DocsCache
		return nil
	}
	markdownCalled := false
	genToolDocsMarkdown = func(cmd *cobra.Command, path string) error {
		markdownCalled = cmd == RootCmd && path == helper.DocsCache
		return nil
	}
	writerCalled := false
	genToolDocsWriter = func(tmpDir string, outputDir string) {
		writerCalled = tmpDir == helper.DocsCache && outputDir == filepath.Clean("./docs")
	}

	if err := runGenToolDocs(filepath.Clean("./docs")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mkdirCalled || !markdownCalled || !writerCalled {
		t.Fatalf("expected mkdir, markdown and writer to all be called")
	}

	mkdirErr := errors.New("mkdir failed")
	genToolDocsMkdirAll = func(string, fs.FileMode) error { return mkdirErr }
	if err := runGenToolDocs("./docs"); !errors.Is(err, mkdirErr) {
		t.Fatalf("expected mkdir error, got %v", err)
	}
}

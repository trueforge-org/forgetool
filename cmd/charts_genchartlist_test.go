package cmd

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/charts/website"
	"github.com/trueforge-org/forgetool/pkg/helper"
)

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

func TestGenChartListCommandRunCallsRunner(t *testing.T) {
	oldRunner := chartsGenChartListRunner
	t.Cleanup(func() { chartsGenChartListRunner = oldRunner })

	called := false
	chartsGenChartListRunner = func(args []string) error {
		called = true
		if len(args) != 1 || args[0] != "./charts" {
			t.Fatalf("unexpected args: %#v", args)
		}
		return nil
	}

	genChartListCmd.Run(genChartListCmd, []string{"./charts"})

	if !called {
		t.Fatalf("expected command Run to call runner")
	}
}

func TestGenChartListCommandRunCallsOnError(t *testing.T) {
	oldRunner := chartsGenChartListRunner
	oldOnError := chartsGenChartListOnError
	t.Cleanup(func() {
		chartsGenChartListRunner = oldRunner
		chartsGenChartListOnError = oldOnError
	})

	want := errors.New("boom")
	chartsGenChartListRunner = func([]string) error { return want }
	called := false
	chartsGenChartListOnError = func(err error) {
		called = true
		if !errors.Is(err, want) {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	genChartListCmd.Run(genChartListCmd, []string{"./charts"})

	if !called {
		t.Fatalf("expected command Run to call error handler")
	}
}

func TestRunChartsGenChartListWithDefaultFixture(t *testing.T) {
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

	chartsGenChartListWalkCharts2 = helper.WalkCharts2
	chartsGenChartListOptionsFactory = func() *website.ChartListOptions {
		return &website.ChartListOptions{OutputPath: "./charts.json", TrainFilter: []string{}}
	}
	chartsGenChartListGetChartData = func(opts *website.ChartListOptions) fs.WalkDirFunc { return opts.GetChartData }
	chartsGenChartListWrite = func(opts *website.ChartListOptions) error { return opts.WriteChartList() }

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp failed: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	fixtureChartsRoot, err := filepath.Abs(filepath.Join(oldWD, "..", "testdata", "cmd", "charts"))
	if err != nil {
		t.Fatalf("fixture abs path failed: %v", err)
	}

	if err := runChartsGenChartList([]string{fixtureChartsRoot}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmp, "charts.json"))
	if err != nil {
		t.Fatalf("read charts.json failed: %v", err)
	}

	var out website.ChartList
	if err := json.Unmarshal(content, &out); err != nil {
		t.Fatalf("parse charts.json failed: %v", err)
	}
	if out.TotalCount < 1 {
		t.Fatalf("expected at least one chart in output")
	}
}

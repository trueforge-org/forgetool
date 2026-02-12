package cmd

import (
	"errors"
	"io/fs"
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

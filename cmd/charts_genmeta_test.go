package cmd

import (
	"errors"
	"testing"

	"github.com/trueforge-org/forgetool/v4/pkg/helper"
)

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

func TestGenMetaCommandRunCallsRunner(t *testing.T) {
	oldRunner := chartsGenMetaRunner
	t.Cleanup(func() { chartsGenMetaRunner = oldRunner })

	called := false
	chartsGenMetaRunner = func(args []string) error {
		called = true
		if len(args) != 2 || args[0] != "minor" || args[1] != "./charts" {
			t.Fatalf("unexpected args: %#v", args)
		}
		return nil
	}

	genMetaCmd.Run(genMetaCmd, []string{"minor", "./charts"})

	if !called {
		t.Fatalf("expected command Run to call runner")
	}
}

func TestGenMetaCommandRunCallsOnError(t *testing.T) {
	oldRunner := chartsGenMetaRunner
	oldOnError := chartsGenMetaOnError
	t.Cleanup(func() {
		chartsGenMetaRunner = oldRunner
		chartsGenMetaOnError = oldOnError
	})

	want := errors.New("boom")
	chartsGenMetaRunner = func([]string) error { return want }
	called := false
	chartsGenMetaOnError = func(err error) {
		called = true
		if !errors.Is(err, want) {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	genMetaCmd.Run(genMetaCmd, []string{"minor", "./charts"})

	if !called {
		t.Fatalf("expected command Run to call error handler")
	}
}

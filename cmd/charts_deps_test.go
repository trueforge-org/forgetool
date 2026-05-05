package cmd

import (
	"errors"
	"reflect"
	"testing"

	"github.com/trueforge-org/forgetool/v4/pkg/helper"
)

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

func TestRunChartsDepsReturnsWalkError(t *testing.T) {
	oldLoad := chartsDepsLoadGPGKey
	oldWalk := chartsDepsWalkCharts
	t.Cleanup(func() {
		chartsDepsLoadGPGKey = oldLoad
		chartsDepsWalkCharts = oldWalk
	})

	chartsDepsLoadGPGKey = func() error { return nil }
	want := errors.New("walk failed")
	chartsDepsWalkCharts = func(_ []string, _ func(string, string) error, _ string, _ helper.WalkMode) error {
		return want
	}

	err := runChartsDeps([]string{"./charts/a"})
	if !errors.Is(err, want) {
		t.Fatalf("expected walk error, got %v", err)
	}
}

func TestDepsCommandRunCallsRunner(t *testing.T) {
	oldRunner := chartsDepsRunner
	t.Cleanup(func() { chartsDepsRunner = oldRunner })

	called := false
	chartsDepsRunner = func(args []string) error {
		called = true
		if len(args) != 2 || args[0] != "a" || args[1] != "b" {
			t.Fatalf("unexpected args: %#v", args)
		}
		return nil
	}

	depsCmd.Run(depsCmd, []string{"a", "b"})

	if !called {
		t.Fatalf("expected deps command run to call runner")
	}
}

func TestDepsCommandRunCallsOnError(t *testing.T) {
	oldRunner := chartsDepsRunner
	oldOnError := chartsDepsOnError
	t.Cleanup(func() {
		chartsDepsRunner = oldRunner
		chartsDepsOnError = oldOnError
	})

	want := errors.New("boom")
	chartsDepsRunner = func([]string) error { return want }
	called := false
	chartsDepsOnError = func(err error) {
		called = true
		if !errors.Is(err, want) {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	depsCmd.Run(depsCmd, []string{"a"})

	if !called {
		t.Fatalf("expected deps command run to call error handler")
	}
}

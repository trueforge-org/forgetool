package cmd

import (
	"errors"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/charts/version"
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

func TestRunChartsBumpReturnsError(t *testing.T) {
	old := chartsBumpVersion
	t.Cleanup(func() { chartsBumpVersion = old })

	want := errors.New("bump failed")
	chartsBumpVersion = func(string, string) error { return want }

	err := runChartsBump([]string{"1.2.3", "patch"})
	if !errors.Is(err, want) {
		t.Fatalf("expected bump error, got %v", err)
	}
}

func TestBumperRunCallsRunner(t *testing.T) {
	oldRunner := chartsBumpRunner
	t.Cleanup(func() { chartsBumpRunner = oldRunner })

	called := false
	chartsBumpRunner = func(args []string) error {
		called = true
		if len(args) != 2 || args[0] != "1.2.3" || args[1] != "patch" {
			t.Fatalf("unexpected args: %#v", args)
		}
		return nil
	}

	bumper.Run(bumper, []string{"1.2.3", "patch"})

	if !called {
		t.Fatalf("expected command Run to call runner")
	}
}

func TestBumperRunCallsOnError(t *testing.T) {
	oldRunner := chartsBumpRunner
	oldOnError := chartsBumpOnError
	t.Cleanup(func() {
		chartsBumpRunner = oldRunner
		chartsBumpOnError = oldOnError
	})

	want := errors.New("boom")
	chartsBumpRunner = func([]string) error { return want }
	called := false
	chartsBumpOnError = func(err error) {
		called = true
		if !errors.Is(err, want) {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	bumper.Run(bumper, []string{"1.2.3", "patch"})

	if !called {
		t.Fatalf("expected command Run to call error handler")
	}
}

func TestRunChartsBumpWithDefaultImplementation(t *testing.T) {
	old := chartsBumpVersion
	t.Cleanup(func() { chartsBumpVersion = old })

	chartsBumpVersion = version.Bump

	if err := runChartsBump([]string{"1.2.3", "patch"}); err != nil {
		t.Fatalf("expected default bump implementation to succeed, got %v", err)
	}
}

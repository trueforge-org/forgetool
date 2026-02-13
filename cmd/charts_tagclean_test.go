package cmd

import (
	"errors"
	"testing"
)

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

func TestRunChartsTagCleanReturnsError(t *testing.T) {
	old := chartsTagClean
	t.Cleanup(func() { chartsTagClean = old })

	want := errors.New("clean failed")
	chartsTagClean = func(tag string) error {
		if tag != "tag@sha" {
			t.Fatalf("unexpected tag: %q", tag)
		}
		return want
	}

	err := runChartsTagClean([]string{"tag@sha"})
	if !errors.Is(err, want) {
		t.Fatalf("expected clean error, got %v", err)
	}
}

func TestTagCleanerRunCallsRunner(t *testing.T) {
	oldRunner := chartsTagCleanRunner
	t.Cleanup(func() { chartsTagCleanRunner = oldRunner })

	called := false
	chartsTagCleanRunner = func(args []string) error {
		called = true
		if len(args) != 1 || args[0] != "tag@sha" {
			t.Fatalf("unexpected args: %#v", args)
		}
		return nil
	}

	tagCleaner.Run(tagCleaner, []string{"tag@sha"})

	if !called {
		t.Fatalf("expected command Run to call runner")
	}
}

func TestTagCleanerRunCallsOnError(t *testing.T) {
	oldRunner := chartsTagCleanRunner
	oldOnError := chartsTagCleanOnError
	t.Cleanup(func() {
		chartsTagCleanRunner = oldRunner
		chartsTagCleanOnError = oldOnError
	})

	want := errors.New("boom")
	chartsTagCleanRunner = func([]string) error { return want }
	called := false
	chartsTagCleanOnError = func(err error) {
		called = true
		if !errors.Is(err, want) {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	tagCleaner.Run(tagCleaner, []string{"tag@sha"})

	if !called {
		t.Fatalf("expected command Run to call error handler")
	}
}

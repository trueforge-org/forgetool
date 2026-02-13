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

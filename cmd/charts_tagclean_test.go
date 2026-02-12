package cmd

import "testing"

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

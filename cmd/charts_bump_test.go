package cmd

import "testing"

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

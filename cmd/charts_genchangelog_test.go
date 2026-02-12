package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/charts/changelog"
)

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

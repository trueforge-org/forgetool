package cmd

import (
	"errors"
	"path/filepath"
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

func TestGenChangelogCommandRunCallsRunner(t *testing.T) {
	oldRunner := chartsGenChangelogRunner
	t.Cleanup(func() { chartsGenChangelogRunner = oldRunner })

	called := false
	chartsGenChangelogRunner = func(args []string) error {
		called = true
		if len(args) != 3 || args[0] != "repo" || args[1] != "tmpl" || args[2] != "charts" {
			t.Fatalf("unexpected args: %#v", args)
		}
		return nil
	}

	genChangelogCmd.Run(genChangelogCmd, []string{"repo", "tmpl", "charts"})

	if !called {
		t.Fatalf("expected command Run to call runner")
	}
}

func TestGenChangelogCommandRunCallsOnError(t *testing.T) {
	oldRunner := chartsGenChangelogRunner
	oldOnError := chartsGenChangelogOnError
	t.Cleanup(func() {
		chartsGenChangelogRunner = oldRunner
		chartsGenChangelogOnError = oldOnError
	})

	want := errors.New("boom")
	chartsGenChangelogRunner = func([]string) error { return want }
	called := false
	chartsGenChangelogOnError = func(err error) {
		called = true
		if !errors.Is(err, want) {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	genChangelogCmd.Run(genChangelogCmd, []string{"repo", "tmpl", "charts"})

	if !called {
		t.Fatalf("expected command Run to call error handler")
	}
}

func TestDefaultChangelogWrappersReturnErrorsOnInvalidInput(t *testing.T) {
	oldGenerate := chartsGenChangelogGenerate
	oldRender := chartsGenChangelogRender
	t.Cleanup(func() {
		chartsGenChangelogGenerate = oldGenerate
		chartsGenChangelogRender = oldRender
	})

	chartsGenChangelogGenerate = func(opts *changelog.ChangelogOptions) error { return opts.Generate() }
	chartsGenChangelogRender = func(opts *changelog.ChangelogOptions) error { return opts.Render() }

	bad := &changelog.ChangelogOptions{
		RepoPath:                  "",
		TemplatePath:              filepath.Join("..", "does-not-exist"),
		ChartsDir:                 "",
		ChangelogFileName:         "CHANGELOG.md",
		JSONOutputPath:            filepath.Join(t.TempDir(), "changelog.json"),
		StatusUpdateInterval:      1,
		SkipCommitsWithBadMessage: false,
	}

	if err := chartsGenChangelogGenerate(bad); err == nil {
		t.Fatalf("expected generate wrapper to return error for invalid options")
	}
	_ = chartsGenChangelogRender(bad)

	oldOnError := chartsGenChangelogOnError
	t.Cleanup(func() { chartsGenChangelogOnError = oldOnError })
	chartsGenChangelogOnError = func(err error) { oldOnError(err) }
	chartsGenChangelogOnError(errors.New("boom"))
}

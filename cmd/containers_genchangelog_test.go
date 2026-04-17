package cmd

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/changelog"
)

func TestRunContainerGenChangelog(t *testing.T) {
	oldGenerate := containerGenChangelogGenerate
	oldRender := containerGenChangelogRender
	t.Cleanup(func() {
		containerGenChangelogGenerate = oldGenerate
		containerGenChangelogRender = oldRender
	})

	if err := runContainerGenChangelog([]string{"only", "two"}); err == nil {
		t.Fatalf("expected missing-args error")
	}

	containerGenChangelogGenerate = func(opts *changelog.ChangelogOptions) error {
		if opts.RepoPath != "repo" || opts.TemplatePath != "tmpl" || opts.AppsDir != "apps" {
			t.Fatalf("unexpected options: %#v", opts)
		}
		if opts.AppType != changelog.AppTypeContainer {
			t.Fatalf("expected AppTypeContainer, got %s", opts.AppType)
		}
		return nil
	}
	containerGenChangelogRender = func(*changelog.ChangelogOptions) error { return nil }
	if err := runContainerGenChangelog([]string{"repo", "tmpl", "apps"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	genErr := errors.New("gen failed")
	containerGenChangelogGenerate = func(*changelog.ChangelogOptions) error { return genErr }
	if err := runContainerGenChangelog([]string{"repo", "tmpl", "apps"}); err == nil || !strings.Contains(err.Error(), "generate changelog") {
		t.Fatalf("expected wrapped generate error, got %v", err)
	}

	containerGenChangelogGenerate = func(*changelog.ChangelogOptions) error { return nil }
	rendErr := errors.New("render failed")
	containerGenChangelogRender = func(*changelog.ChangelogOptions) error { return rendErr }
	if err := runContainerGenChangelog([]string{"repo", "tmpl", "apps"}); err == nil || !strings.Contains(err.Error(), "render changelog") {
		t.Fatalf("expected wrapped render error, got %v", err)
	}
}

func TestContainerGenChangelogCommandRunCallsRunner(t *testing.T) {
	oldRunner := containerGenChangelogRunner
	t.Cleanup(func() { containerGenChangelogRunner = oldRunner })

	called := false
	containerGenChangelogRunner = func(args []string) error {
		called = true
		if len(args) != 3 || args[0] != "repo" || args[1] != "tmpl" || args[2] != "apps" {
			t.Fatalf("unexpected args: %#v", args)
		}
		return nil
	}

	containerGenChangelogCmd.Run(containerGenChangelogCmd, []string{"repo", "tmpl", "apps"})

	if !called {
		t.Fatalf("expected command Run to call runner")
	}
}

func TestContainerGenChangelogCommandRunCallsOnError(t *testing.T) {
	oldRunner := containerGenChangelogRunner
	oldOnError := containerGenChangelogOnError
	t.Cleanup(func() {
		containerGenChangelogRunner = oldRunner
		containerGenChangelogOnError = oldOnError
	})

	want := errors.New("boom")
	containerGenChangelogRunner = func([]string) error { return want }
	called := false
	containerGenChangelogOnError = func(err error) {
		called = true
		if !errors.Is(err, want) {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	containerGenChangelogCmd.Run(containerGenChangelogCmd, []string{"repo", "tmpl", "apps"})

	if !called {
		t.Fatalf("expected command Run to call error handler")
	}
}

func TestDefaultContainerChangelogWrappersReturnErrorsOnInvalidInput(t *testing.T) {
	oldGenerate := containerGenChangelogGenerate
	oldRender := containerGenChangelogRender
	t.Cleanup(func() {
		containerGenChangelogGenerate = oldGenerate
		containerGenChangelogRender = oldRender
	})

	containerGenChangelogGenerate = defaultContainerGenChangelogGenerate
	containerGenChangelogRender = defaultContainerGenChangelogRender

	bad := &changelog.ChangelogOptions{
		RepoPath:                  "",
		TemplatePath:              filepath.Join("..", "does-not-exist"),
		AppsDir:                   "",
		ChangelogFileName:         "CHANGELOG.md",
		JSONOutputPath:            filepath.Join(t.TempDir(), "changelog.json"),
		StatusUpdateInterval:      1,
		SkipCommitsWithBadMessage: false,
		AppType:                   changelog.AppTypeContainer,
	}

	if err := containerGenChangelogGenerate(bad); err == nil {
		t.Fatalf("expected generate wrapper to return error for invalid options")
	}
	_ = containerGenChangelogRender(bad)
}

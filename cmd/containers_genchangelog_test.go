package cmd

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/changelog"
)

func TestRunContainersGenChangelog(t *testing.T) {
	oldGenerate := containersGenChangelogGenerate
	oldRender := containersGenChangelogRender
	t.Cleanup(func() {
		containersGenChangelogGenerate = oldGenerate
		containersGenChangelogRender = oldRender
	})

	if err := runContainersGenChangelog([]string{"only", "two"}); err == nil {
		t.Fatalf("expected missing-args error")
	}

	containersGenChangelogGenerate = func(opts *changelog.ChangelogOptions) error {
		if opts.RepoPath != "repo" || opts.TemplatePath != "tmpl" || opts.AppsDir != "apps" {
			t.Fatalf("unexpected options: %#v", opts)
		}
		if opts.AppType != changelog.AppTypeContainer {
			t.Fatalf("expected AppTypeContainer, got %s", opts.AppType)
		}
		return nil
	}
	containersGenChangelogRender = func(*changelog.ChangelogOptions) error { return nil }
	if err := runContainersGenChangelog([]string{"repo", "tmpl", "apps"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	genErr := errors.New("gen failed")
	containersGenChangelogGenerate = func(*changelog.ChangelogOptions) error { return genErr }
	if err := runContainersGenChangelog([]string{"repo", "tmpl", "apps"}); err == nil || !strings.Contains(err.Error(), "generate changelog") {
		t.Fatalf("expected wrapped generate error, got %v", err)
	}

	containersGenChangelogGenerate = func(*changelog.ChangelogOptions) error { return nil }
	rendErr := errors.New("render failed")
	containersGenChangelogRender = func(*changelog.ChangelogOptions) error { return rendErr }
	if err := runContainersGenChangelog([]string{"repo", "tmpl", "apps"}); err == nil || !strings.Contains(err.Error(), "render changelog") {
		t.Fatalf("expected wrapped render error, got %v", err)
	}
}

func TestContainersGenChangelogCommandRunCallsRunner(t *testing.T) {
	oldRunner := containersGenChangelogRunner
	t.Cleanup(func() { containersGenChangelogRunner = oldRunner })

	called := false
	containersGenChangelogRunner = func(args []string) error {
		called = true
		if len(args) != 3 || args[0] != "repo" || args[1] != "tmpl" || args[2] != "apps" {
			t.Fatalf("unexpected args: %#v", args)
		}
		return nil
	}

	containersGenChangelogCmd.Run(containersGenChangelogCmd, []string{"repo", "tmpl", "apps"})

	if !called {
		t.Fatalf("expected command Run to call runner")
	}
}

func TestContainersGenChangelogCommandRunCallsOnError(t *testing.T) {
	oldRunner := containersGenChangelogRunner
	oldOnError := containersGenChangelogOnError
	t.Cleanup(func() {
		containersGenChangelogRunner = oldRunner
		containersGenChangelogOnError = oldOnError
	})

	want := errors.New("boom")
	containersGenChangelogRunner = func([]string) error { return want }
	called := false
	containersGenChangelogOnError = func(err error) {
		called = true
		if !errors.Is(err, want) {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	containersGenChangelogCmd.Run(containersGenChangelogCmd, []string{"repo", "tmpl", "apps"})

	if !called {
		t.Fatalf("expected command Run to call error handler")
	}
}

func TestDefaultContainersChangelogWrappersReturnErrorsOnInvalidInput(t *testing.T) {
	oldGenerate := containersGenChangelogGenerate
	oldRender := containersGenChangelogRender
	t.Cleanup(func() {
		containersGenChangelogGenerate = oldGenerate
		containersGenChangelogRender = oldRender
	})

	containersGenChangelogGenerate = defaultContainersGenChangelogGenerate
	containersGenChangelogRender = defaultContainersGenChangelogRender

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

	if err := containersGenChangelogGenerate(bad); err == nil {
		t.Fatalf("expected generate wrapper to return error for invalid options")
	}
	_ = containersGenChangelogRender(bad)
}

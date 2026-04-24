package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/website"
)

func setupContainersGenDocsStubs(t *testing.T) {
	t.Helper()
	oldPrep := containersGenDocsPrepareWebsite
	oldDisc := containersGenDocsDiscoverApps
	oldProc := containersGenDocsProcessApp
	oldFin := containersGenDocsFinalizeWebsite
	t.Cleanup(func() {
		containersGenDocsPrepareWebsite = oldPrep
		containersGenDocsDiscoverApps = oldDisc
		containersGenDocsProcessApp = oldProc
		containersGenDocsFinalizeWebsite = oldFin
	})
	containersGenDocsPrepareWebsite = func(website.ContainerOptions) error { return nil }
	containersGenDocsDiscoverApps = func(string) ([]string, error) { return nil, nil }
	containersGenDocsProcessApp = func(website.ContainerOptions) error { return nil }
	containersGenDocsFinalizeWebsite = func(website.ContainerOptions, string) error { return nil }
}

func TestRunContainersGenDocs_PrepareError(t *testing.T) {
	setupContainersGenDocsStubs(t)
	containersGenDocsPrepareWebsite = func(website.ContainerOptions) error { return errors.New("boom") }
	if err := runContainersGenDocs(nil); err == nil || !strings.Contains(err.Error(), "prepare website") {
		t.Fatalf("expected prepare error, got %v", err)
	}
}

func TestRunContainersGenDocs_DiscoverError(t *testing.T) {
	setupContainersGenDocsStubs(t)
	containersGenDocsDiscoverApps = func(string) ([]string, error) { return nil, errors.New("nope") }
	if err := runContainersGenDocs(nil); err == nil || !strings.Contains(err.Error(), "discover apps") {
		t.Fatalf("expected discover error, got %v", err)
	}
}

func TestRunContainersGenDocs_DiscoverSuccess(t *testing.T) {
	setupContainersGenDocsStubs(t)
	containersGenDocsDiscoverApps = func(string) ([]string, error) { return []string{"sonarr"}, nil }
	called := 0
	containersGenDocsProcessApp = func(opts website.ContainerOptions) error {
		called++
		if opts.App != "sonarr" {
			t.Fatalf("unexpected app: %s", opts.App)
		}
		return nil
	}
	if err := runContainersGenDocs(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called != 1 {
		t.Fatalf("expected 1 ProcessApp call, got %d", called)
	}
}

func TestRunContainersGenDocs_ProcessError(t *testing.T) {
	setupContainersGenDocsStubs(t)
	containersGenDocsProcessApp = func(website.ContainerOptions) error { return errors.New("fail") }
	if err := runContainersGenDocs([]string{"sonarr"}); err == nil || !strings.Contains(err.Error(), "process sonarr") {
		t.Fatalf("expected process error, got %v", err)
	}
}

func TestRunContainersGenDocs_FinalizeError(t *testing.T) {
	setupContainersGenDocsStubs(t)
	containersGenDocsFinalizeWebsite = func(website.ContainerOptions, string) error { return errors.New("end") }
	if err := runContainersGenDocs([]string{"sonarr"}); err == nil || !strings.Contains(err.Error(), "finalize website") {
		t.Fatalf("expected finalize error, got %v", err)
	}
}

func TestRunContainersGenDocs_HappyPathExplicit(t *testing.T) {
	setupContainersGenDocsStubs(t)
	if err := runContainersGenDocs([]string{"sonarr", "radarr"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContainersGenDocsCmdRunCallsRunner(t *testing.T) {
	oldRunner := containersGenDocsRunner
	t.Cleanup(func() { containersGenDocsRunner = oldRunner })
	called := false
	containersGenDocsRunner = func([]string) error { called = true; return nil }
	containersGenDocsCmd.Run(containersGenDocsCmd, []string{"sonarr"})
	if !called {
		t.Fatalf("expected runner to be called")
	}
}

func TestContainersGenDocsCmdRunCallsOnError(t *testing.T) {
	oldRunner := containersGenDocsRunner
	oldOnError := containersGenDocsOnError
	t.Cleanup(func() {
		containersGenDocsRunner = oldRunner
		containersGenDocsOnError = oldOnError
	})
	want := errors.New("boom")
	containersGenDocsRunner = func([]string) error { return want }
	called := false
	containersGenDocsOnError = func(err error) {
		called = true
		if !errors.Is(err, want) {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	containersGenDocsCmd.Run(containersGenDocsCmd, nil)
	if !called {
		t.Fatalf("expected onError to be called")
	}
}

package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/containertest"
)

func TestRunContainerTestPassesArgumentsToRunner(t *testing.T) {
	oldRunner := containerTestRunChecksFn
	t.Cleanup(func() { containerTestRunChecksFn = oldRunner })

	called := false
	containerTestRunChecksFn = func(ctx context.Context, image string, yamlPath string, config *containertest.ContainerConfig) error {
		called = true
		if image != "ghcr.io/trueforge-org/app:latest" {
			t.Fatalf("expected image to be passed through, got %q", image)
		}
		if yamlPath != "./container-test.yaml" {
			t.Fatalf("expected yamlPath to be passed through, got %q", yamlPath)
		}
		if config == nil {
			t.Fatalf("expected container config to be set")
		}
		if got := config.Env["KEY"]; got != "VALUE" {
			t.Fatalf("expected env KEY=VALUE, got %q", got)
		}
		return nil
	}

	err := runContainerTest(context.Background(), "ghcr.io/trueforge-org/app:latest", "./container-test.yaml", []string{"KEY=VALUE"})
	if err != nil {
		t.Fatalf("runContainerTest returned unexpected error: %v", err)
	}
	if !called {
		t.Fatalf("expected runner function to be called")
	}
}

func TestRunContainerTestReturnsErrorForInvalidEnv(t *testing.T) {
	err := runContainerTest(context.Background(), "image", "./container-test.yaml", []string{"INVALID"})
	if err == nil {
		t.Fatalf("expected error for invalid --env value")
	}
}

func TestRunContainerTestReturnsErrorWhenImageMissing(t *testing.T) {
	err := runContainerTest(context.Background(), "", "./container-test.yaml", nil)
	if err == nil {
		t.Fatalf("expected error when --image is missing")
	}
}

func TestRunContainerTestReturnsErrorWhenYAMLMissing(t *testing.T) {
	err := runContainerTest(context.Background(), "image", "", nil)
	if err == nil {
		t.Fatalf("expected error when --yaml is missing")
	}
}

func TestRunContainerTestWrapsRunnerError(t *testing.T) {
	oldRunner := containerTestRunChecksFn
	t.Cleanup(func() { containerTestRunChecksFn = oldRunner })

	containerTestRunChecksFn = func(ctx context.Context, image string, yamlPath string, config *containertest.ContainerConfig) error {
		return errors.New("boom")
	}

	err := runContainerTest(context.Background(), "image", "./container-test.yaml", nil)
	if err == nil {
		t.Fatalf("expected error when runner fails")
	}
}

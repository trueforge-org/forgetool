package cmd

import (
	"context"
	"errors"
	"testing"

	containertest "github.com/trueforge-org/forgetool/v4/pkg/containers/test"
)

func TestRunContainersTestPassesArgumentsToRunner(t *testing.T) {
	oldRunner := containersTestRunChecksFn
	t.Cleanup(func() { containersTestRunChecksFn = oldRunner })

	called := false
	containersTestRunChecksFn = func(ctx context.Context, image string, yamlPath string, config *containertest.ContainerConfig) error {
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

	err := runContainersTest(context.Background(), "ghcr.io/trueforge-org/app:latest", "./container-test.yaml", []string{"KEY=VALUE"})
	if err != nil {
		t.Fatalf("runContainersTest returned unexpected error: %v", err)
	}
	if !called {
		t.Fatalf("expected runner function to be called")
	}
}

func TestRunContainersTestReturnsErrorForInvalidEnv(t *testing.T) {
	err := runContainersTest(context.Background(), "image", "./container-test.yaml", []string{"INVALID"})
	if err == nil {
		t.Fatalf("expected error for invalid --env value")
	}
}

func TestRunContainersTestReturnsErrorForEmptyEnvKey(t *testing.T) {
	err := runContainersTest(context.Background(), "image", "./container-test.yaml", []string{"=VALUE"})
	if err == nil {
		t.Fatalf("expected error for empty --env key")
	}
}

func TestRunContainersTestReturnsErrorWhenImageMissing(t *testing.T) {
	err := runContainersTest(context.Background(), "", "./container-test.yaml", nil)
	if err == nil {
		t.Fatalf("expected error when --image is missing")
	}
}

func TestRunContainersTestReturnsErrorWhenConfigMissing(t *testing.T) {
	err := runContainersTest(context.Background(), "image", "", nil)
	if err == nil {
		t.Fatalf("expected error when --config is missing")
	}
}

func TestRunContainersTestWrapsRunnerError(t *testing.T) {
	oldRunner := containersTestRunChecksFn
	t.Cleanup(func() { containersTestRunChecksFn = oldRunner })

	containersTestRunChecksFn = func(ctx context.Context, image string, yamlPath string, config *containertest.ContainerConfig) error {
		return errors.New("boom")
	}

	err := runContainersTest(context.Background(), "image", "./container-test.yaml", nil)
	if err == nil {
		t.Fatalf("expected error when runner fails")
	}
}

func TestContainersTestCmdRunEDelegatesToRunContainerTest(t *testing.T) {
	oldRunner := containersTestRunChecksFn
	oldImage := containersTestImage
	oldConfig := containersTestConfigPath
	oldEnv := containersTestEnvPairs
	t.Cleanup(func() {
		containersTestRunChecksFn = oldRunner
		containersTestImage = oldImage
		containersTestConfigPath = oldConfig
		containersTestEnvPairs = oldEnv
	})

	called := false
	containersTestRunChecksFn = func(ctx context.Context, image string, yamlPath string, config *containertest.ContainerConfig) error {
		called = true
		if image != "img" || yamlPath != "cfg.yaml" {
			t.Fatalf("unexpected delegated args: image=%q yaml=%q", image, yamlPath)
		}
		if got := config.Env["A"]; got != "1" {
			t.Fatalf("unexpected delegated env A=%q", got)
		}
		return nil
	}

	containersTestImage = "img"
	containersTestConfigPath = "cfg.yaml"
	containersTestEnvPairs = []string{"A=1"}

	if err := containersTestCmd.RunE(containersTestCmd, nil); err != nil {
		t.Fatalf("unexpected RunE error: %v", err)
	}
	if !called {
		t.Fatalf("expected runner to be called from RunE")
	}
}

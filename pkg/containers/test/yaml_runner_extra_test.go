package containertest

import (
	"context"
	"testing"
	"time"
)

// Exercise the runner-level "if config.TimeoutSeconds > 0" branch in
// RunChecksFromYAML. With Runners non-empty AND TimeoutSeconds > 0 the
// per-runner context-with-timeout block is taken.
func TestRunChecksFromYAML_RunnerTimeoutAndDeadlineHonored(t *testing.T) {
	setYAMLRunnerSeams(t)

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			TimeoutSeconds: 1, // < minYAMLTimeoutSeconds → bumped to floor.
			Runners: []RunnerConfig{
				{ExpectedOutput: "ok"},
			},
		}, nil
	}

	checkRunnerOutputFn = func(ctx context.Context, _ string, _ *ContainerConfig, _, _ string, _ *int) error {
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatalf("expected runner deadline")
		}
		if time.Until(deadline) < 179*time.Second {
			t.Fatalf("expected runner timeout floor, got %v", time.Until(deadline))
		}
		return nil
	}
	checkStandardRunFn = func(context.Context, string, *ContainerConfig) error { return nil }

	// Parent ctx with a much shorter deadline so the "honoring configured
	// timeout" branch is taken.
	parent, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := RunChecksFromYAML(parent, "img", "cfg.yaml", nil); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

package cmd

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRunCheckcryptPassesExpectedFlags(t *testing.T) {
	oldFn := checkcryptCheckFilesAndReportEncryption
	t.Cleanup(func() { checkcryptCheckFilesAndReportEncryption = oldFn })

	called := false
	checkcryptCheckFilesAndReportEncryption = func(precommit bool, decrypt bool) error {
		called = true
		if precommit {
			t.Fatalf("expected precommit=false")
		}
		if decrypt {
			t.Fatalf("expected decrypt=false")
		}
		return nil
	}

	if err := runCheckcrypt(); err != nil {
		t.Fatalf("runCheckcrypt returned unexpected error: %v", err)
	}
	if !called {
		t.Fatalf("expected checker function to be called")
	}
}

func TestCheckcryptCommandRunInvokesExitOnError(t *testing.T) {
	oldFn := checkcryptCheckFilesAndReportEncryption
	oldExit := checkcryptExit
	t.Cleanup(func() {
		checkcryptCheckFilesAndReportEncryption = oldFn
		checkcryptExit = oldExit
	})

	checkcryptCheckFilesAndReportEncryption = func(bool, bool) error {
		return errors.New("boom")
	}

	exitCode := -1
	checkcryptExit = func(code int) {
		exitCode = code
	}

	checkcrypt.Run(checkcrypt, nil)

	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
}

func TestRunPrecommitPassesExpectedFlags(t *testing.T) {
	oldFn := precommitCheckFilesAndReportEncryption
	t.Cleanup(func() { precommitCheckFilesAndReportEncryption = oldFn })

	called := false
	precommitCheckFilesAndReportEncryption = func(precommit bool, decrypt bool) error {
		called = true
		if !precommit {
			t.Fatalf("expected precommit=true")
		}
		if !decrypt {
			t.Fatalf("expected decrypt=true")
		}
		return nil
	}

	if err := runPrecommit(); err != nil {
		t.Fatalf("runPrecommit returned unexpected error: %v", err)
	}
	if !called {
		t.Fatalf("expected checker function to be called")
	}
}

func TestRunAdvTestCommandAppliesAllManifests(t *testing.T) {
	oldApply := advTestKubectlApply
	t.Cleanup(func() { advTestKubectlApply = oldApply })

	manifestPaths := []string{"a.yaml", "b.yaml", "c.yaml"}
	var called []string
	advTestKubectlApply = func(_ context.Context, filePath string) error {
		called = append(called, filePath)
		return nil
	}

	if err := runAdvTestCommand(context.Background(), manifestPaths); err != nil {
		t.Fatalf("runAdvTestCommand returned unexpected error: %v", err)
	}

	if len(called) != len(manifestPaths) {
		t.Fatalf("expected %d apply calls, got %d", len(manifestPaths), len(called))
	}
	for i := range manifestPaths {
		if called[i] != manifestPaths[i] {
			t.Fatalf("expected call %d to %q, got %q", i, manifestPaths[i], called[i])
		}
	}
}

func TestRunAdvTestCommandReturnsWrappedError(t *testing.T) {
	oldApply := advTestKubectlApply
	t.Cleanup(func() { advTestKubectlApply = oldApply })

	advTestKubectlApply = func(_ context.Context, filePath string) error {
		if filePath == "fail.yaml" {
			return errors.New("apply failed")
		}
		return nil
	}

	err := runAdvTestCommand(context.Background(), []string{"ok.yaml", "fail.yaml"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "fail.yaml") {
		t.Fatalf("expected wrapped error to include failing file, got: %v", err)
	}
}

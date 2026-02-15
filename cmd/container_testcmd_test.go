package cmd

import (
	"errors"
	"testing"
)

func TestRunContainerTestDelegatesToRunner(t *testing.T) {
	oldRunner := containerTestRunner
	t.Cleanup(func() {
		containerTestRunner = oldRunner
	})

	calledWith := ""
	containerTestRunner = func(path string) error {
		calledWith = path
		return nil
	}

	if err := runContainerTest("/tmp/container-test.yaml"); err != nil {
		t.Fatalf("runContainerTest returned error: %v", err)
	}
	if calledWith != "/tmp/container-test.yaml" {
		t.Fatalf("expected runner path %q, got %q", "/tmp/container-test.yaml", calledWith)
	}
}

func TestRunContainerTestReturnsRunnerError(t *testing.T) {
	oldRunner := containerTestRunner
	t.Cleanup(func() {
		containerTestRunner = oldRunner
	})

	expectedErr := errors.New("boom")
	containerTestRunner = func(string) error {
		return expectedErr
	}

	if err := runContainerTest("/tmp/container-test.yaml"); !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

func TestContainerTestCommandDefaultConfigFlag(t *testing.T) {
	flag := containerTestCmd.Flags().Lookup("config")
	if flag == nil {
		t.Fatal("expected --config flag to exist")
	}
	if flag.DefValue != "container-test.yaml" {
		t.Fatalf("expected default config value %q, got %q", "container-test.yaml", flag.DefValue)
	}
}

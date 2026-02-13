package cmd

import (
	"errors"
	"testing"
)

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

func TestPrecommitRunExitsWithCodeOneOnError(t *testing.T) {
	oldFn := precommitCheckFilesAndReportEncryption
	oldExit := precommitExit
	t.Cleanup(func() {
		precommitCheckFilesAndReportEncryption = oldFn
		precommitExit = oldExit
	})

	checkerCalled := false
	exitCalled := false
	exitCode := 0

	precommitCheckFilesAndReportEncryption = func(precommit bool, decrypt bool) error {
		checkerCalled = true
		return errors.New("failed")
	}
	precommitExit = func(code int) {
		exitCalled = true
		exitCode = code
	}

	precommit.Run(precommit, []string{})

	if !checkerCalled {
		t.Fatalf("expected checker function to be called")
	}
	if !exitCalled {
		t.Fatalf("expected exit to be called")
	}
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
}

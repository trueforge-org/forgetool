package cmd

import (
	"errors"
	"os"
	"os/exec"
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

func TestHelperCheckcryptProcess(t *testing.T) {
	if os.Getenv("GO_WANT_CHECKCRYPT_HELPER") != "1" {
		return
	}
	checkcrypt.Run(checkcrypt, []string{})
	os.Exit(0)
}

func TestCheckcryptRunExitsNonZero(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperCheckcryptProcess")
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "GO_WANT_CHECKCRYPT_HELPER=1")
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected non-zero exit from checkcrypt command")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() == 0 {
		t.Fatalf("expected non-zero exit code, got: %v", err)
	}
}

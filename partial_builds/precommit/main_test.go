package main

import (
	"os"
	"os/exec"
	"testing"
)

func TestHelperProcessMain(t *testing.T) {
	if os.Getenv("GO_WANT_PRECOMMIT_HELPER") != "1" {
		return
	}
	main()
	os.Exit(0)
}

func TestMain_ExitsNonZeroOnCheckError(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcessMain")
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "GO_WANT_PRECOMMIT_HELPER=1")
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected subprocess to exit non-zero")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %T", err)
	}
	if exitErr.ExitCode() == 0 {
		t.Fatalf("expected non-zero exit code")
	}
}

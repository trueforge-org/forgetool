package cmd

import (
	"os"
	"os/exec"
	"testing"
)

func TestHelperTalosResetExitProcess(t *testing.T) {
	if os.Getenv("GO_WANT_TALOS_RESET_EXIT_HELPER") != "1" {
		return
	}
	reset.Run(reset, []string{"all"})
	os.Exit(0)
}

func TestTalosResetRunExitsNonZero(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperTalosResetExitProcess")
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "GO_WANT_TALOS_RESET_EXIT_HELPER=1")
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected non-zero exit for talos-reset")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() == 0 {
		t.Fatalf("expected non-zero exit code for talos-reset, got %v", err)
	}
}

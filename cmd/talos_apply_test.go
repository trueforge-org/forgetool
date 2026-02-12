package cmd

import (
	"os"
	"os/exec"
	"testing"
)

func TestHelperTalosApplyExitProcess(t *testing.T) {
	if os.Getenv("GO_WANT_TALOS_APPLY_EXIT_HELPER") != "1" {
		return
	}
	apply.Run(apply, []string{"all"})
	os.Exit(0)
}

func TestTalosApplyRunExitsNonZero(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperTalosApplyExitProcess")
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "GO_WANT_TALOS_APPLY_EXIT_HELPER=1")
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected non-zero exit for talos-apply")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() == 0 {
		t.Fatalf("expected non-zero exit code for talos-apply, got %v", err)
	}
}

package cmd

import (
	"os"
	"os/exec"
	"testing"
)

func TestHelperTalosHealthExitProcess(t *testing.T) {
	if os.Getenv("GO_WANT_TALOS_HEALTH_EXIT_HELPER") != "1" {
		return
	}
	health.Run(health, nil)
	os.Exit(0)
}

func TestTalosHealthRunExitsNonZero(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperTalosHealthExitProcess")
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "GO_WANT_TALOS_HEALTH_EXIT_HELPER=1")
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected non-zero exit for talos-health")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() == 0 {
		t.Fatalf("expected non-zero exit code for talos-health, got %v", err)
	}
}

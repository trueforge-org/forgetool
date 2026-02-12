package cmd

import (
	"os"
	"os/exec"
	"testing"
)

func TestHelperTalosUpgradeExitProcess(t *testing.T) {
	if os.Getenv("GO_WANT_TALOS_UPGRADE_EXIT_HELPER") != "1" {
		return
	}
	upgrade.Run(upgrade, []string{"all"})
	os.Exit(0)
}

func TestTalosUpgradeRunExitsNonZero(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperTalosUpgradeExitProcess")
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "GO_WANT_TALOS_UPGRADE_EXIT_HELPER=1")
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected non-zero exit for talos-upgrade")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() == 0 {
		t.Fatalf("expected non-zero exit code for talos-upgrade, got %v", err)
	}
}

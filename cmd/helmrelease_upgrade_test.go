package cmd

import (
	"os"
	"os/exec"
	"testing"
)

func TestHelperHelmreleaseUpgradeExitProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HR_UPGRADE_EXIT_HELPER") != "1" {
		return
	}
	hrupgrade.Run(hrupgrade, []string{"./clusters/main/kubernetes/apps/demo/app/helm-release.yaml"})
	os.Exit(0)
}

func TestHelmreleaseUpgradeRunExitsNonZero(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperHelmreleaseUpgradeExitProcess")
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "GO_WANT_HR_UPGRADE_EXIT_HELPER=1")
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected non-zero exit for hr-upgrade")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() == 0 {
		t.Fatalf("expected non-zero exit code for hr-upgrade, got %v", err)
	}
}

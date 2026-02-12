package cmd

import (
	"os"
	"os/exec"
	"testing"
)

func TestHelperTalosKubeconfigExitProcess(t *testing.T) {
	if os.Getenv("GO_WANT_TALOS_KUBECONFIG_EXIT_HELPER") != "1" {
		return
	}
	kubeconfig.Run(kubeconfig, []string{"all"})
	os.Exit(0)
}

func TestTalosKubeconfigRunExitsNonZero(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperTalosKubeconfigExitProcess")
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "GO_WANT_TALOS_KUBECONFIG_EXIT_HELPER=1")
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected non-zero exit for talos-kubeconfig")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() == 0 {
		t.Fatalf("expected non-zero exit code for talos-kubeconfig, got %v", err)
	}
}

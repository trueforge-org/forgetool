package cmd

import (
	"os"
	"os/exec"
	"testing"
)

func TestHelperFluxBootstrapExitProcess(t *testing.T) {
	if os.Getenv("GO_WANT_FLUX_BOOTSTRAP_EXIT_HELPER") != "1" {
		return
	}
	fluxbootstrap.Run(fluxbootstrap, nil)
	os.Exit(0)
}

func TestFluxBootstrapRunExitsNonZero(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperFluxBootstrapExitProcess")
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "GO_WANT_FLUX_BOOTSTRAP_EXIT_HELPER=1")
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected non-zero exit for flux-bootstrap")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() == 0 {
		t.Fatalf("expected non-zero exit code for flux-bootstrap, got %v", err)
	}
}

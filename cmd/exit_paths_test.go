package cmd

import (
	"os"
	"os/exec"
	"testing"
)

func TestHelperCommandExitProcess(t *testing.T) {
	mode := os.Getenv("GO_WANT_CMD_EXIT_HELPER")
	switch mode {
	case "talos-apply":
		apply.Run(apply, []string{"all"})
	case "talos-upgrade":
		upgrade.Run(upgrade, []string{"all"})
	case "talos-kubeconfig":
		kubeconfig.Run(kubeconfig, []string{"all"})
	case "talos-reset":
		reset.Run(reset, []string{"all"})
	case "talos-health":
		health.Run(health, nil)
	case "adv-test":
		testcmd.Run(testcmd, nil)
	case "hr-install":
		hrinstall.Run(hrinstall, []string{"./clusters/main/kubernetes/apps/demo/app/helm-release.yaml"})
	case "hr-upgrade":
		hrupgrade.Run(hrupgrade, []string{"./clusters/main/kubernetes/apps/demo/app/helm-release.yaml"})
	case "flux-bootstrap":
		fluxbootstrap.Run(fluxbootstrap, nil)
	default:
		return
	}
	os.Exit(0)
}

func TestCommandRunExitPaths(t *testing.T) {
	modes := []string{
		"talos-apply",
		"talos-upgrade",
		"talos-kubeconfig",
		"talos-reset",
		"talos-health",
		"adv-test",
		"hr-install",
		"hr-upgrade",
		"flux-bootstrap",
	}

	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=TestHelperCommandExitProcess")
			cmd.Dir = t.TempDir()
			cmd.Env = append(os.Environ(), "GO_WANT_CMD_EXIT_HELPER="+mode)
			err := cmd.Run()
			if err == nil {
				t.Fatalf("expected non-zero exit for mode %s", mode)
			}
			exitErr, ok := err.(*exec.ExitError)
			if !ok || exitErr.ExitCode() == 0 {
				t.Fatalf("expected non-zero exit code for mode %s, got %v", mode, err)
			}
		})
	}
}

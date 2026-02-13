package nodestatus

import (
	"fmt"
	"strings"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/helper"
	talosctlpkg "github.com/trueforge-org/forgetool/pkg/talosctl"
)

func writeFakeTalos(t *testing.T, stubFunc func(args []string) (string, error)) {
	t.Helper()
	talosctlpkg.SetExecutor(func(args []string, silent bool) (string, error) {
		return stubFunc(args)
	})
	t.Cleanup(func() {
		talosctlpkg.SetExecutor(func(args []string, silent bool) (string, error) {
			commandSlice := append([]string{talosctlpkg.CommandPrefix()}, args...)
			return helper.RunCommand(commandSlice, silent)
		})
	})
}

func TestStatusAndHealthSuccess(t *testing.T) {
	oldCache := helper.CacheDir
	oldClusterPath := helper.ClusterPath
	helper.CacheDir = t.TempDir()
	helper.ClusterPath = t.TempDir()
	t.Cleanup(func() {
		helper.CacheDir = oldCache
		helper.ClusterPath = oldClusterPath
	})

	writeFakeTalos(t, func(args []string) (string, error) {
		// Join args to check what's being requested
		argsStr := strings.Join(args, " ")
		if strings.Contains(argsStr, "jsonpath={.spec.stage}") {
			return "running", nil
		}
		if strings.Contains(argsStr, "jsonpath={.spec.status.ready}") {
			return "true", nil
		}
		return "unknown", nil
	})

	cmd := baseStatusCMD("10.0.0.10")
	if len(cmd) < 6 || !strings.Contains(strings.Join(cmd, " "), "machinestatus") {
		t.Fatalf("unexpected base command: %v", cmd)
	}

	status, err := CheckStatus("10.0.0.10")
	if err != nil {
		t.Fatalf("CheckStatus failed: %v", err)
	}
	if !strings.Contains(status, "running") {
		t.Fatalf("expected running status, got %q", status)
	}

	ready, err := CheckReadyStatus("10.0.0.10", false)
	if err != nil {
		t.Fatalf("CheckReadyStatus failed: %v", err)
	}
	if !strings.Contains(ready, "true") {
		t.Fatalf("expected ready true, got %q", ready)
	}

	if err := CheckHealth("10.0.0.10", "", true); err != nil {
		t.Fatalf("CheckHealth failed: %v", err)
	}
}

func TestCheckNeedBootstrapInsecureFallback(t *testing.T) {
	oldCache := helper.CacheDir
	oldClusterPath := helper.ClusterPath
	helper.CacheDir = t.TempDir()
	helper.ClusterPath = t.TempDir()
	t.Cleanup(func() {
		helper.CacheDir = oldCache
		helper.ClusterPath = oldClusterPath
	})

	writeFakeTalos(t, func(args []string) (string, error) {
		// Check if --insecure flag is present
		for _, arg := range args {
			if arg == "--insecure" {
				return "maintenance", nil
			}
		}
		// Return error with message in output
		return "certificate signed by unknown authority", fmt.Errorf("certificate signed by unknown authority")
	})

	need, err := CheckNeedBootstrap("10.0.0.20")
	if err != nil {
		t.Fatalf("CheckNeedBootstrap failed: %v", err)
	}
	if !need {
		t.Fatalf("expected bootstrap to be needed on maintenance status")
	}
}

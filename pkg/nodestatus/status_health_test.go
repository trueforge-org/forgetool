package nodestatus

import (
	"strings"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/helper"
	talosctlpkg "github.com/trueforge-org/forgetool/pkg/talosctl"
)

func writeFakeTalos(t *testing.T, script string) {
	t.Helper()
	talosctlpkg.SetExecutor(func(args []string, silent bool) (string, error) {
		commandSlice := append([]string{"sh", "-c", script, "talosctl"}, args...)
		return helper.RunCommand(commandSlice, silent)
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

	script := "#!/bin/sh\ncase \"$*\" in\n  *\"jsonpath={.spec.stage}\"*) echo running ;;\n  *\"jsonpath={.spec.status.ready}\"*) echo true ;;\n  *) echo unknown ;;\nesac\n"
	writeFakeTalos(t, script)

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

	script := "#!/bin/sh\nif echo \"$*\" | grep -q -- '--insecure'; then\n  echo maintenance\n  exit 0\nfi\necho 'certificate signed by unknown authority' 1>&2\nexit 1\n"
	writeFakeTalos(t, script)

	need, err := CheckNeedBootstrap("10.0.0.20")
	if err != nil {
		t.Fatalf("CheckNeedBootstrap failed: %v", err)
	}
	if !need {
		t.Fatalf("expected bootstrap to be needed on maintenance status")
	}
}

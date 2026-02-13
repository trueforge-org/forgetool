package cmd

import (
	"errors"
	"reflect"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

func TestRunTalosUpgradeCallsDependencies(t *testing.T) {
	oldDecrypt := talosUpgradeDecryptFiles
	oldLoadEnv := talosUpgradeLoadTalEnv
	oldLoadCfg := talosUpgradeLoadTalConfig
	oldGenUpgrade := talosUpgradeGenUpgrade
	oldExecCmds := talosUpgradeExecCmds
	oldGenKubeUpgrade := talosUpgradeGenKubeUpgrade
	oldExecCmd := talosUpgradeExecCmd
	oldGenPlain := talosUpgradeGenPlain
	oldTalEnv := helper.TalEnv
	t.Cleanup(func() {
		talosUpgradeDecryptFiles = oldDecrypt
		talosUpgradeLoadTalEnv = oldLoadEnv
		talosUpgradeLoadTalConfig = oldLoadCfg
		talosUpgradeGenUpgrade = oldGenUpgrade
		talosUpgradeExecCmds = oldExecCmds
		talosUpgradeGenKubeUpgrade = oldGenKubeUpgrade
		talosUpgradeExecCmd = oldExecCmd
		talosUpgradeGenPlain = oldGenPlain
		helper.TalEnv = oldTalEnv
	})

	helper.TalEnv = map[string]string{"VIP_IP": "10.0.0.1"}
	talosUpgradeDecryptFiles = func() error { return errors.New("decrypt") }
	talosUpgradeLoadTalEnv = func(bool) error { return nil }
	loadedCfg := false
	talosUpgradeLoadTalConfig = func() { loadedCfg = true }
	talosUpgradeGenUpgrade = func(node string, extraArgs []string) []string {
		if node != "" || !reflect.DeepEqual(extraArgs, []string{"--stage"}) {
			t.Fatalf("unexpected upgrade args")
		}
		return []string{"upgrade-cmd"}
	}
	execCmdsCalled := false
	talosUpgradeExecCmds = func(cmds []string, healthcheck bool) error {
		execCmdsCalled = true
		if !reflect.DeepEqual(cmds, []string{"upgrade-cmd"}) || !healthcheck {
			t.Fatalf("unexpected execcmds args")
		}
		return nil
	}
	talosUpgradeGenKubeUpgrade = func(vip string) string {
		if vip != "10.0.0.1" {
			t.Fatalf("unexpected vip: %s", vip)
		}
		return "kube-upgrade-cmd"
	}
	talosUpgradeGenPlain = func(action string, node string, extraArgs []string) []string {
		if action != "health" || node != "10.0.0.1" || !reflect.DeepEqual(extraArgs, []string{"-f"}) {
			t.Fatalf("unexpected plain args")
		}
		return []string{"kubeconfig-cmd"}
	}

	var execs []string
	talosUpgradeExecCmd = func(cmd string) { execs = append(execs, cmd) }

	runTalosUpgrade([]string{"all", "--stage"})

	if !loadedCfg || !execCmdsCalled {
		t.Fatalf("expected load config and execcmds calls")
	}
	if !reflect.DeepEqual(execs, []string{"kube-upgrade-cmd", "kubeconfig-cmd"}) {
		t.Fatalf("unexpected exec command sequence: %#v", execs)
	}
}

func TestTalosUpgradeCommandRunCallsHelper(t *testing.T) {
	oldDecrypt := talosUpgradeDecryptFiles
	oldLoadEnv := talosUpgradeLoadTalEnv
	oldLoadCfg := talosUpgradeLoadTalConfig
	oldGenUpgrade := talosUpgradeGenUpgrade
	oldExecCmds := talosUpgradeExecCmds
	oldGenKubeUpgrade := talosUpgradeGenKubeUpgrade
	oldExecCmd := talosUpgradeExecCmd
	oldGenPlain := talosUpgradeGenPlain
	oldTalEnv := helper.TalEnv
	t.Cleanup(func() {
		talosUpgradeDecryptFiles = oldDecrypt
		talosUpgradeLoadTalEnv = oldLoadEnv
		talosUpgradeLoadTalConfig = oldLoadCfg
		talosUpgradeGenUpgrade = oldGenUpgrade
		talosUpgradeExecCmds = oldExecCmds
		talosUpgradeGenKubeUpgrade = oldGenKubeUpgrade
		talosUpgradeExecCmd = oldExecCmd
		talosUpgradeGenPlain = oldGenPlain
		helper.TalEnv = oldTalEnv
	})

	helper.TalEnv = map[string]string{"VIP_IP": "10.0.0.1"}
	talosUpgradeDecryptFiles = func() error { return nil }
	talosUpgradeLoadTalEnv = func(bool) error { return nil }
	talosUpgradeLoadTalConfig = func() {}
	talosUpgradeGenUpgrade = func(string, []string) []string { return []string{"upgrade"} }
	talosUpgradeExecCmds = func([]string, bool) error { return nil }
	talosUpgradeGenKubeUpgrade = func(string) string { return "kube" }
	talosUpgradeGenPlain = func(string, string, []string) []string { return []string{"kcfg"} }
	called := false
	talosUpgradeExecCmd = func(string) { called = true }

	upgrade.Run(upgrade, []string{"10.0.0.2"})
	if !called {
		t.Fatalf("expected command Run to invoke helper flow")
	}
}

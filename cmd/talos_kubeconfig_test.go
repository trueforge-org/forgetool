package cmd

import (
	"errors"
	"reflect"
	"testing"
)

func TestRunTalosKubeconfigCallsDependencies(t *testing.T) {
	oldDecrypt := talosKubeconfigDecryptFiles
	oldLoadEnv := talosKubeconfigLoadTalEnv
	oldLoadCfg := talosKubeconfigLoadTalConfig
	oldGenPlain := talosKubeconfigGenPlain
	oldExecCmds := talosKubeconfigExecCmds
	t.Cleanup(func() {
		talosKubeconfigDecryptFiles = oldDecrypt
		talosKubeconfigLoadTalEnv = oldLoadEnv
		talosKubeconfigLoadTalConfig = oldLoadCfg
		talosKubeconfigGenPlain = oldGenPlain
		talosKubeconfigExecCmds = oldExecCmds
	})

	talosKubeconfigDecryptFiles = func() error { return errors.New("decrypt") }
	talosKubeconfigLoadTalEnv = func(bool) error { return nil }
	loadedCfg := false
	talosKubeconfigLoadTalConfig = func() { loadedCfg = true }
	talosKubeconfigGenPlain = func(action string, node string, extraArgs []string) []string {
		if action != "kubeconfig" || node != "" || !reflect.DeepEqual(extraArgs, []string{"--flag"}) {
			t.Fatalf("unexpected gen plain args")
		}
		return []string{"kubeconfig-cmd"}
	}
	execCalled := false
	talosKubeconfigExecCmds = func(cmds []string, healthcheck bool) error {
		execCalled = true
		if !reflect.DeepEqual(cmds, []string{"kubeconfig-cmd"}) || !healthcheck {
			t.Fatalf("unexpected exec args")
		}
		return nil
	}

	runTalosKubeconfig([]string{"all", "--flag"})
	if !loadedCfg || !execCalled {
		t.Fatalf("expected load config and exec calls")
	}
}

func TestTalosKubeconfigCommandRunCallsHelper(t *testing.T) {
	oldDecrypt := talosKubeconfigDecryptFiles
	oldLoadEnv := talosKubeconfigLoadTalEnv
	oldLoadCfg := talosKubeconfigLoadTalConfig
	oldGenPlain := talosKubeconfigGenPlain
	oldExecCmds := talosKubeconfigExecCmds
	t.Cleanup(func() {
		talosKubeconfigDecryptFiles = oldDecrypt
		talosKubeconfigLoadTalEnv = oldLoadEnv
		talosKubeconfigLoadTalConfig = oldLoadCfg
		talosKubeconfigGenPlain = oldGenPlain
		talosKubeconfigExecCmds = oldExecCmds
	})

	talosKubeconfigDecryptFiles = func() error { return nil }
	talosKubeconfigLoadTalEnv = func(bool) error { return nil }
	talosKubeconfigLoadTalConfig = func() {}
	talosKubeconfigGenPlain = func(string, string, []string) []string { return []string{"cmd"} }
	called := false
	talosKubeconfigExecCmds = func([]string, bool) error { called = true; return nil }

	kubeconfig.Run(kubeconfig, []string{"10.0.0.2"})
	if !called {
		t.Fatalf("expected command Run to invoke helper flow")
	}
}

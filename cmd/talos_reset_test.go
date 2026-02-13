package cmd

import (
	"errors"
	"reflect"
	"testing"
)

func TestRunTalosResetCallsDependencies(t *testing.T) {
	oldDecrypt := talosResetDecryptFiles
	oldLoadEnv := talosResetLoadTalEnv
	oldLoadCfg := talosResetLoadTalConfig
	oldGenPlain := talosResetGenPlain
	oldExecCmds := talosResetExecCmds
	t.Cleanup(func() {
		talosResetDecryptFiles = oldDecrypt
		talosResetLoadTalEnv = oldLoadEnv
		talosResetLoadTalConfig = oldLoadCfg
		talosResetGenPlain = oldGenPlain
		talosResetExecCmds = oldExecCmds
	})

	talosResetDecryptFiles = func() error { return errors.New("decrypt") }
	talosResetLoadTalEnv = func(bool) error { return nil }
	loadedCfg := false
	talosResetLoadTalConfig = func() { loadedCfg = true }
	talosResetGenPlain = func(action string, node string, extraArgs []string) []string {
		if action != "reset" || node != "" || !reflect.DeepEqual(extraArgs, []string{"--graceful"}) {
			t.Fatalf("unexpected gen plain args")
		}
		return []string{"reset-cmd"}
	}
	execCalled := false
	talosResetExecCmds = func(cmds []string, healthcheck bool) error {
		execCalled = true
		if !reflect.DeepEqual(cmds, []string{"reset-cmd"}) || !healthcheck {
			t.Fatalf("unexpected exec args")
		}
		return nil
	}

	runTalosReset([]string{"all", "--graceful"})
	if !loadedCfg || !execCalled {
		t.Fatalf("expected load config and exec calls")
	}
}

func TestTalosResetCommandRunCallsHelper(t *testing.T) {
	oldDecrypt := talosResetDecryptFiles
	oldLoadEnv := talosResetLoadTalEnv
	oldLoadCfg := talosResetLoadTalConfig
	oldGenPlain := talosResetGenPlain
	oldExecCmds := talosResetExecCmds
	t.Cleanup(func() {
		talosResetDecryptFiles = oldDecrypt
		talosResetLoadTalEnv = oldLoadEnv
		talosResetLoadTalConfig = oldLoadCfg
		talosResetGenPlain = oldGenPlain
		talosResetExecCmds = oldExecCmds
	})

	talosResetDecryptFiles = func() error { return nil }
	talosResetLoadTalEnv = func(bool) error { return nil }
	talosResetLoadTalConfig = func() {}
	talosResetGenPlain = func(string, string, []string) []string { return []string{"cmd"} }
	called := false
	talosResetExecCmds = func([]string, bool) error { called = true; return nil }

	reset.Run(reset, []string{"10.0.0.2"})
	if !called {
		t.Fatalf("expected command Run to invoke helper flow")
	}
}

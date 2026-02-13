package cmd

import (
	"os"
	"path/filepath"
	"testing"

	talhelperCfg "github.com/budimanjojo/talhelper/v3/pkg/config"
	"github.com/trueforge-org/forgetool/pkg/helper"
	"github.com/trueforge-org/forgetool/pkg/talassist"
)

func TestBootstrapFunc_TalosOnlyPath(t *testing.T) {
	oldPrompt := talosBootstrapGetYesOrNo
	oldGenPlain := talosBootstrapGenPlain
	oldExecCmd := talosBootstrapExecCmd
	oldStdin := os.Stdin
	oldTalConfigFile := helper.TalosConfigFile
	oldTalConfig := talassist.TalConfig
	defer func() {
		talosBootstrapGetYesOrNo = oldPrompt
		talosBootstrapGenPlain = oldGenPlain
		talosBootstrapExecCmd = oldExecCmd
		os.Stdin = oldStdin
		helper.TalosConfigFile = oldTalConfigFile
		talassist.TalConfig = oldTalConfig
	}()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe failed: %v", err)
	}
	if _, err := w.Write([]byte("n\n")); err != nil {
		t.Fatalf("write prompt input failed: %v", err)
	}
	_ = w.Close()
	os.Stdin = r

	helper.TalosConfigFile = filepath.Join(t.TempDir(), "talosconfig")
	talassist.TalConfig = &talhelperCfg.TalhelperConfig{Nodes: []talhelperCfg.Node{{Hostname: "cp1", IPAddress: "10.0.0.10"}}}
	talosBootstrapGetYesOrNo = func(string, bool) bool { return false }
	talosBootstrapGenPlain = func(action string, node string, extraArgs []string) []string {
		if action != "bootstrap" || node != "10.0.0.10" {
			t.Fatalf("unexpected gen plain input: %s %s", action, node)
		}
		return []string{"bootstrap cmd"}
	}
	execCalled := false
	talosBootstrapExecCmd = func(cmd string) {
		execCalled = true
		if cmd != "bootstrap cmd" {
			t.Fatalf("unexpected bootstrap command: %s", cmd)
		}
	}

	bootstrapfunc(nil, []string{})

	if !execCalled {
		t.Fatalf("expected bootstrap command execution")
	}
}

func TestBootstrapFunc_ForgeBootstrapPath(t *testing.T) {
	oldPrompt := talosBootstrapGetYesOrNo
	oldLoadEnv := talosBootstrapLoadTalEnv
	oldLoadConfig := talosBootstrapLoadTalConfig
	oldRunBootstrap := talosBootstrapRunBootstrap
	t.Cleanup(func() {
		talosBootstrapGetYesOrNo = oldPrompt
		talosBootstrapLoadTalEnv = oldLoadEnv
		talosBootstrapLoadTalConfig = oldLoadConfig
		talosBootstrapRunBootstrap = oldRunBootstrap
	})

	loadedEnv := false
	loadedCfg := false
	calledArgs := []string{}
	talosBootstrapGetYesOrNo = func(string, bool) bool { return true }
	talosBootstrapLoadTalEnv = func(bool) error { loadedEnv = true; return nil }
	talosBootstrapLoadTalConfig = func() { loadedCfg = true }
	talosBootstrapRunBootstrap = func(args []string) { calledArgs = append([]string{}, args...) }

	bootstrapfunc(nil, []string{"--foo", "bar"})

	if !loadedEnv || !loadedCfg {
		t.Fatalf("expected tal env and config loaders to be called")
	}
	if len(calledArgs) != 2 || calledArgs[0] != "--foo" || calledArgs[1] != "bar" {
		t.Fatalf("unexpected bootstrap args: %#v", calledArgs)
	}
}

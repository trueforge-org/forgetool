package cmd

import (
	"errors"
	"os"
	"os/exec"
	"reflect"
	"testing"

	talhelperCfg "github.com/budimanjojo/talhelper/v3/pkg/config"
	"github.com/trueforge-org/forgetool/pkg/helper"
	"github.com/trueforge-org/forgetool/pkg/talassist"
)

func TestParseTalosApplyArgs(t *testing.T) {
	node, extra := parseTalosApplyArgs([]string{"all", "--foo"})
	if node != "" {
		t.Fatalf("expected empty node for all, got %q", node)
	}
	if !reflect.DeepEqual(extra, []string{"--foo"}) {
		t.Fatalf("unexpected extra args: %#v", extra)
	}

	node, extra = parseTalosApplyArgs([]string{"10.0.0.2", "--bar"})
	if node != "10.0.0.2" {
		t.Fatalf("unexpected node: %q", node)
	}
	if !reflect.DeepEqual(extra, []string{"--bar"}) {
		t.Fatalf("unexpected extra args: %#v", extra)
	}
}

func TestRunTalosApplyRunningCallsRunApply(t *testing.T) {
	oldDecrypt := talosApplyDecryptFiles
	oldLoadEnv := talosApplyLoadTalEnv
	oldLoadConfig := talosApplyLoadTalConfig
	oldWait := talosApplyWaitForHealth
	oldRunApply := talosApplyRunApply
	oldTalConfig := talassist.TalConfig
	t.Cleanup(func() {
		talosApplyDecryptFiles = oldDecrypt
		talosApplyLoadTalEnv = oldLoadEnv
		talosApplyLoadTalConfig = oldLoadConfig
		talosApplyWaitForHealth = oldWait
		talosApplyRunApply = oldRunApply
		talassist.TalConfig = oldTalConfig
	})

	talassist.TalConfig = &talhelperCfg.TalhelperConfig{Nodes: []talhelperCfg.Node{{IPAddress: "10.0.0.1"}}}
	talosApplyDecryptFiles = func() error { return nil }
	talosApplyLoadTalEnv = func(bool) error { return nil }
	talosApplyLoadTalConfig = func() {}
	talosApplyWaitForHealth = func(node string, states []string) (string, error) {
		if node != "10.0.0.1" {
			t.Fatalf("unexpected node: %s", node)
		}
		if !reflect.DeepEqual(states, []string{"running", "maintenance"}) {
			t.Fatalf("unexpected states: %#v", states)
		}
		return "running", nil
	}

	called := false
	talosApplyRunApply = func(kubeconfig bool, node string, extraArgs []string) {
		called = true
		if !kubeconfig || node != "10.0.0.2" || !reflect.DeepEqual(extraArgs, []string{"--x"}) {
			t.Fatalf("unexpected runapply args: %v %s %#v", kubeconfig, node, extraArgs)
		}
	}

	runTalosApply([]string{"10.0.0.2", "--x"})
	if !called {
		t.Fatalf("expected RunApply call")
	}
}

func TestRunTalosApplyMaintenanceBootstrapFlow(t *testing.T) {
	oldDecrypt := talosApplyDecryptFiles
	oldLoadEnv := talosApplyLoadTalEnv
	oldLoadConfig := talosApplyLoadTalConfig
	oldWait := talosApplyWaitForHealth
	oldNeedBootstrap := talosApplyCheckNeedBootstrap
	oldPrompt := talosApplyGetYesOrNo
	oldRunBootstrap := talosApplyRunBootstrap
	oldRunApply := talosApplyRunApply
	oldTalConfig := talassist.TalConfig
	t.Cleanup(func() {
		talosApplyDecryptFiles = oldDecrypt
		talosApplyLoadTalEnv = oldLoadEnv
		talosApplyLoadTalConfig = oldLoadConfig
		talosApplyWaitForHealth = oldWait
		talosApplyCheckNeedBootstrap = oldNeedBootstrap
		talosApplyGetYesOrNo = oldPrompt
		talosApplyRunBootstrap = oldRunBootstrap
		talosApplyRunApply = oldRunApply
		talassist.TalConfig = oldTalConfig
	})

	talassist.TalConfig = &talhelperCfg.TalhelperConfig{Nodes: []talhelperCfg.Node{{IPAddress: "10.0.0.1"}}}
	talosApplyDecryptFiles = func() error { return nil }
	talosApplyLoadTalEnv = func(bool) error { return nil }
	talosApplyLoadTalConfig = func() {}
	talosApplyWaitForHealth = func(string, []string) (string, error) { return "maintenance", nil }
	talosApplyCheckNeedBootstrap = func(string) (bool, error) { return true, nil }

	promptCount := 0
	talosApplyGetYesOrNo = func(string, bool) bool {
		promptCount++
		return true
	}

	bootstrapped := false
	talosApplyRunBootstrap = func(args []string) {
		bootstrapped = true
		if !reflect.DeepEqual(args, []string{"--flag"}) {
			t.Fatalf("unexpected bootstrap args: %#v", args)
		}
	}

	applied := false
	talosApplyRunApply = func(kubeconfig bool, node string, extraArgs []string) {
		applied = true
		if kubeconfig {
			t.Fatalf("expected kubeconfig=false on post-bootstrap all apply")
		}
		if node != "" || !reflect.DeepEqual(extraArgs, []string{"--flag"}) {
			t.Fatalf("unexpected runapply args: %q %#v", node, extraArgs)
		}
	}

	runTalosApply([]string{"all", "--flag"})
	if !bootstrapped || !applied || promptCount != 2 {
		t.Fatalf("expected bootstrap+apply flow with two prompts")
	}
}

func TestRunTalosApplyMaintenanceNonInteractiveSkipsBootstrapPrompt(t *testing.T) {
	oldDecrypt := talosApplyDecryptFiles
	oldLoadEnv := talosApplyLoadTalEnv
	oldLoadConfig := talosApplyLoadTalConfig
	oldWait := talosApplyWaitForHealth
	oldNeedBootstrap := talosApplyCheckNeedBootstrap
	oldPrompt := talosApplyGetYesOrNo
	oldRunBootstrap := talosApplyRunBootstrap
	oldRunApply := talosApplyRunApply
	oldNonInteractive := helper.NonInteractive
	oldTalConfig := talassist.TalConfig
	t.Cleanup(func() {
		talosApplyDecryptFiles = oldDecrypt
		talosApplyLoadTalEnv = oldLoadEnv
		talosApplyLoadTalConfig = oldLoadConfig
		talosApplyWaitForHealth = oldWait
		talosApplyCheckNeedBootstrap = oldNeedBootstrap
		talosApplyGetYesOrNo = oldPrompt
		talosApplyRunBootstrap = oldRunBootstrap
		talosApplyRunApply = oldRunApply
		helper.NonInteractive = oldNonInteractive
		talassist.TalConfig = oldTalConfig
	})

	helper.NonInteractive = true
	talassist.TalConfig = &talhelperCfg.TalhelperConfig{Nodes: []talhelperCfg.Node{{IPAddress: "10.0.0.1"}}}
	talosApplyDecryptFiles = func() error { return nil }
	talosApplyLoadTalEnv = func(bool) error { return nil }
	talosApplyLoadTalConfig = func() {}
	talosApplyWaitForHealth = func(string, []string) (string, error) { return "maintenance", nil }
	talosApplyCheckNeedBootstrap = func(string) (bool, error) { return true, nil }

	prompts := 0
	talosApplyGetYesOrNo = func(question string, defaultValue bool) bool {
		prompts++
		if prompts == 1 && question != "Do you want to apply config to all remaining clusternodes as well? (yes/no) [y/n]: " {
			t.Fatalf("unexpected first prompt: %q", question)
		}
		if !defaultValue {
			t.Fatalf("expected defaultValue=true in non-interactive branch")
		}
		return defaultValue
	}

	bootstrapped := false
	talosApplyRunBootstrap = func(args []string) {
		bootstrapped = true
		if !reflect.DeepEqual(args, []string{"--flag"}) {
			t.Fatalf("unexpected bootstrap args: %#v", args)
		}
	}

	applied := false
	talosApplyRunApply = func(bool, string, []string) { applied = true }

	runTalosApply([]string{"all", "--flag"})

	if !bootstrapped {
		t.Fatalf("expected bootstrap in non-interactive mode")
	}
	if prompts != 1 {
		t.Fatalf("expected one post-bootstrap decision call, got %d", prompts)
	}
	if !applied {
		t.Fatalf("expected post-bootstrap apply when defaultValue=true")
	}
}

func TestRunTalosApplyMaintenanceNoBootstrapCallsApply(t *testing.T) {
	oldDecrypt := talosApplyDecryptFiles
	oldLoadEnv := talosApplyLoadTalEnv
	oldLoadConfig := talosApplyLoadTalConfig
	oldWait := talosApplyWaitForHealth
	oldNeedBootstrap := talosApplyCheckNeedBootstrap
	oldRunApply := talosApplyRunApply
	oldTalConfig := talassist.TalConfig
	t.Cleanup(func() {
		talosApplyDecryptFiles = oldDecrypt
		talosApplyLoadTalEnv = oldLoadEnv
		talosApplyLoadTalConfig = oldLoadConfig
		talosApplyWaitForHealth = oldWait
		talosApplyCheckNeedBootstrap = oldNeedBootstrap
		talosApplyRunApply = oldRunApply
		talassist.TalConfig = oldTalConfig
	})

	talassist.TalConfig = &talhelperCfg.TalhelperConfig{Nodes: []talhelperCfg.Node{{IPAddress: "10.0.0.1"}}}
	talosApplyDecryptFiles = func() error { return nil }
	talosApplyLoadTalEnv = func(bool) error { return nil }
	talosApplyLoadTalConfig = func() {}
	talosApplyWaitForHealth = func(string, []string) (string, error) { return "maintenance", nil }
	talosApplyCheckNeedBootstrap = func(string) (bool, error) { return false, nil }

	called := false
	talosApplyRunApply = func(kubeconfig bool, node string, extraArgs []string) {
		called = true
		if !kubeconfig || node != "10.0.0.2" || !reflect.DeepEqual(extraArgs, []string{"--y"}) {
			t.Fatalf("unexpected runapply args")
		}
	}

	runTalosApply([]string{"10.0.0.2", "--y"})
	if !called {
		t.Fatalf("expected apply to be called")
	}
}

func TestRunTalosApplyStopsOnHealthError(t *testing.T) {
	oldDecrypt := talosApplyDecryptFiles
	oldLoadEnv := talosApplyLoadTalEnv
	oldLoadConfig := talosApplyLoadTalConfig
	oldWait := talosApplyWaitForHealth
	oldRunApply := talosApplyRunApply
	oldTalConfig := talassist.TalConfig
	t.Cleanup(func() {
		talosApplyDecryptFiles = oldDecrypt
		talosApplyLoadTalEnv = oldLoadEnv
		talosApplyLoadTalConfig = oldLoadConfig
		talosApplyWaitForHealth = oldWait
		talosApplyRunApply = oldRunApply
		talassist.TalConfig = oldTalConfig
	})

	talassist.TalConfig = &talhelperCfg.TalhelperConfig{Nodes: []talhelperCfg.Node{{IPAddress: "10.0.0.1"}}}
	talosApplyDecryptFiles = func() error { return nil }
	talosApplyLoadTalEnv = func(bool) error { return nil }
	talosApplyLoadTalConfig = func() {}
	talosApplyWaitForHealth = func(string, []string) (string, error) { return "", errors.New("health failed") }

	called := false
	talosApplyRunApply = func(bool, string, []string) { called = true }

	runTalosApply([]string{"10.0.0.2"})
	if called {
		t.Fatalf("did not expect apply call when health check errors")
	}
}

func TestTalosApplyCommandRunCallsHelper(t *testing.T) {
	oldDecrypt := talosApplyDecryptFiles
	oldLoadEnv := talosApplyLoadTalEnv
	oldLoadConfig := talosApplyLoadTalConfig
	oldWait := talosApplyWaitForHealth
	oldRunApply := talosApplyRunApply
	oldTalConfig := talassist.TalConfig
	t.Cleanup(func() {
		talosApplyDecryptFiles = oldDecrypt
		talosApplyLoadTalEnv = oldLoadEnv
		talosApplyLoadTalConfig = oldLoadConfig
		talosApplyWaitForHealth = oldWait
		talosApplyRunApply = oldRunApply
		talassist.TalConfig = oldTalConfig
	})

	talassist.TalConfig = &talhelperCfg.TalhelperConfig{Nodes: []talhelperCfg.Node{{IPAddress: "10.0.0.1"}}}
	talosApplyDecryptFiles = func() error { return nil }
	talosApplyLoadTalEnv = func(bool) error { return nil }
	talosApplyLoadTalConfig = func() {}
	talosApplyWaitForHealth = func(string, []string) (string, error) { return "running", nil }

	called := false
	talosApplyRunApply = func(bool, string, []string) { called = true }

	apply.Run(apply, []string{"10.0.0.2"})
	if !called {
		t.Fatalf("expected apply command Run to call helper flow")
	}
}

func TestRunTalosApplyMaintenanceBootstrapCheckErrorStops(t *testing.T) {
	oldDecrypt := talosApplyDecryptFiles
	oldLoadEnv := talosApplyLoadTalEnv
	oldLoadConfig := talosApplyLoadTalConfig
	oldWait := talosApplyWaitForHealth
	oldNeedBootstrap := talosApplyCheckNeedBootstrap
	oldRunApply := talosApplyRunApply
	oldTalConfig := talassist.TalConfig
	t.Cleanup(func() {
		talosApplyDecryptFiles = oldDecrypt
		talosApplyLoadTalEnv = oldLoadEnv
		talosApplyLoadTalConfig = oldLoadConfig
		talosApplyWaitForHealth = oldWait
		talosApplyCheckNeedBootstrap = oldNeedBootstrap
		talosApplyRunApply = oldRunApply
		talassist.TalConfig = oldTalConfig
	})

	talassist.TalConfig = &talhelperCfg.TalhelperConfig{Nodes: []talhelperCfg.Node{{IPAddress: "10.0.0.1"}}}
	talosApplyDecryptFiles = func() error { return nil }
	talosApplyLoadTalEnv = func(bool) error { return nil }
	talosApplyLoadTalConfig = func() {}
	talosApplyWaitForHealth = func(string, []string) (string, error) { return "maintenance", nil }
	talosApplyCheckNeedBootstrap = func(string) (bool, error) { return false, errors.New("boom") }

	called := false
	talosApplyRunApply = func(bool, string, []string) { called = true }

	runTalosApply([]string{"10.0.0.2"})
	if called {
		t.Fatalf("did not expect apply call on bootstrap-check error")
	}
}

func TestRunTalosApplyMaintenanceBootstrapDeclined(t *testing.T) {
	oldDecrypt := talosApplyDecryptFiles
	oldLoadEnv := talosApplyLoadTalEnv
	oldLoadConfig := talosApplyLoadTalConfig
	oldWait := talosApplyWaitForHealth
	oldNeedBootstrap := talosApplyCheckNeedBootstrap
	oldPrompt := talosApplyGetYesOrNo
	oldRunBootstrap := talosApplyRunBootstrap
	oldRunApply := talosApplyRunApply
	oldTalConfig := talassist.TalConfig
	t.Cleanup(func() {
		talosApplyDecryptFiles = oldDecrypt
		talosApplyLoadTalEnv = oldLoadEnv
		talosApplyLoadTalConfig = oldLoadConfig
		talosApplyWaitForHealth = oldWait
		talosApplyCheckNeedBootstrap = oldNeedBootstrap
		talosApplyGetYesOrNo = oldPrompt
		talosApplyRunBootstrap = oldRunBootstrap
		talosApplyRunApply = oldRunApply
		talassist.TalConfig = oldTalConfig
	})

	talassist.TalConfig = &talhelperCfg.TalhelperConfig{Nodes: []talhelperCfg.Node{{IPAddress: "10.0.0.1"}}}
	talosApplyDecryptFiles = func() error { return errors.New("decrypt failed") }
	talosApplyLoadTalEnv = func(bool) error { return nil }
	talosApplyLoadTalConfig = func() {}
	talosApplyWaitForHealth = func(string, []string) (string, error) { return "maintenance", nil }
	talosApplyCheckNeedBootstrap = func(string) (bool, error) { return true, nil }
	talosApplyGetYesOrNo = func(string, bool) bool { return false }

	bootstrapped := false
	talosApplyRunBootstrap = func([]string) { bootstrapped = true }
	applied := false
	talosApplyRunApply = func(bool, string, []string) { applied = true }

	runTalosApply([]string{"all"})
	if bootstrapped || applied {
		t.Fatalf("expected no bootstrap/apply when first prompt is declined")
	}
}

func TestRunApplyWithoutKubeconfig(t *testing.T) {
	oldGenApply := talosApplyGenApply
	oldExecCmds := talosApplyExecCmds
	oldGenPlain := talosApplyGenPlain
	oldExecCmd := talosApplyExecCmd
	t.Cleanup(func() {
		talosApplyGenApply = oldGenApply
		talosApplyExecCmds = oldExecCmds
		talosApplyGenPlain = oldGenPlain
		talosApplyExecCmd = oldExecCmd
	})

	talosApplyGenApply = func(node string, extraArgs []string) []string {
		if node != "10.0.0.2" {
			t.Fatalf("unexpected node: %s", node)
		}
		if !reflect.DeepEqual(extraArgs, []string{"--immediate"}) {
			t.Fatalf("unexpected extra args: %#v", extraArgs)
		}
		return []string{"apply-cmd"}
	}

	execCmdsCalled := false
	talosApplyExecCmds = func(cmds []string, healthcheck bool) error {
		execCmdsCalled = true
		if !reflect.DeepEqual(cmds, []string{"apply-cmd"}) {
			t.Fatalf("unexpected cmds: %#v", cmds)
		}
		if !healthcheck {
			t.Fatalf("expected healthcheck=true")
		}
		return nil
	}

	genPlainCalled := false
	execCmdCalled := false
	talosApplyGenPlain = func(string, string, []string) []string {
		genPlainCalled = true
		return []string{"kubeconfig-cmd"}
	}
	talosApplyExecCmd = func(string) { execCmdCalled = true }

	RunApply(false, "10.0.0.2", []string{"--immediate"})

	if !execCmdsCalled {
		t.Fatalf("expected ExecCmds to be called")
	}
	if genPlainCalled || execCmdCalled {
		t.Fatalf("did not expect kubeconfig command when kubeconfig=false")
	}
}

func TestRunApplyWithKubeconfig(t *testing.T) {
	oldGenApply := talosApplyGenApply
	oldExecCmds := talosApplyExecCmds
	oldGenPlain := talosApplyGenPlain
	oldExecCmd := talosApplyExecCmd
	oldTalEnv := helper.TalEnv
	t.Cleanup(func() {
		talosApplyGenApply = oldGenApply
		talosApplyExecCmds = oldExecCmds
		talosApplyGenPlain = oldGenPlain
		talosApplyExecCmd = oldExecCmd
		helper.TalEnv = oldTalEnv
	})

	helper.TalEnv = map[string]string{"VIP_IP": "10.0.0.1"}
	talosApplyGenApply = func(string, []string) []string { return []string{"apply-cmd"} }
	talosApplyExecCmds = func([]string, bool) error { return nil }

	genPlainCalled := false
	talosApplyGenPlain = func(action string, node string, extraArgs []string) []string {
		genPlainCalled = true
		if action != "kubeconfig" || node != "10.0.0.1" {
			t.Fatalf("unexpected plain args: %s %s", action, node)
		}
		if !reflect.DeepEqual(extraArgs, []string{"-f"}) {
			t.Fatalf("unexpected kubeconfig extra args: %#v", extraArgs)
		}
		return []string{"kubeconfig-cmd"}
	}

	executed := ""
	talosApplyExecCmd = func(cmd string) { executed = cmd }

	RunApply(true, "", nil)

	if !genPlainCalled {
		t.Fatalf("expected kubeconfig command generation")
	}
	if executed != "kubeconfig-cmd" {
		t.Fatalf("expected kubeconfig-cmd execution, got %q", executed)
	}
}

func TestHelperTalosApplyExitProcess(t *testing.T) {
	if os.Getenv("GO_WANT_TALOS_APPLY_EXIT_HELPER") != "1" {
		return
	}
	apply.Run(apply, []string{"all"})
	os.Exit(0)
}

func TestTalosApplyRunExitsNonZero(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperTalosApplyExitProcess")
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "GO_WANT_TALOS_APPLY_EXIT_HELPER=1")
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected non-zero exit for talos-apply")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() == 0 {
		t.Fatalf("expected non-zero exit code for talos-apply, got %v", err)
	}
}

package cmd

import (
	"os"
	"os/exec"
	"reflect"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

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

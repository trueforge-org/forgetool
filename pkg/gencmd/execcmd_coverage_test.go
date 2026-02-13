package gencmd

import (
	"errors"
	"testing"
	"time"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

func TestExecCmdCoveragePaths(t *testing.T) {
	resetGencmdHooks(t)
	sleepFn = func(time.Duration) {}

	runTalosctlCommandFn = func([]string, bool) (string, error) { return "ok", nil }
	ExecCmd("talosctl health")

	runTalosctlCommandFn = func([]string, bool) (string, error) { return "err", errors.New("fail") }
	ExecCmd("talosctl health")

	calls := 0
	runTalosctlCommandFn = func([]string, bool) (string, error) {
		calls++
		if calls == 1 {
			return "x", errors.New("fail")
		}
		return "ok", nil
	}
	ExecCmd("talosctl bootstrap")

	calls = 0
	runTalosctlCommandFn = func([]string, bool) (string, error) {
		calls++
		switch calls {
		case 1:
			return "bootstrap is not available yet", errors.New("fail")
		case 2:
			return "bootstrap is not available yet", errors.New("fail")
		default:
			return "done", nil
		}
	}
	ExecCmd("talosctl bootstrap")

	resetGencmdHooks(t)
	sleepFn = func(time.Duration) {}
	bootstrapRetryTimeout = 0
	nowFn = func() time.Time { return time.Unix(0, 0) }
	sinceFn = func(time.Time) time.Duration { return time.Hour }
	runTalosctlCommandFn = func([]string, bool) (string, error) {
		return "bootstrap is not available yet", errors.New("still")
	}
	ExecCmd("talosctl bootstrap")
}

func TestBuildTodoCmdsAndExecCmds(t *testing.T) {
	resetGencmdHooks(t)
	sleepFn = func(time.Duration) {}
	helper.TalEnv["VIP_IP"] = "10.0.0.9"

	cmds, skipped := buildTodoCmds([]string{"a"}, false)
	if len(cmds) != 1 || skipped {
		t.Fatalf("unexpected no-healthcheck result: %v %v", cmds, skipped)
	}

	checkNodeHealthFn = func(string, string, bool) error { return nil }
	getYesOrNoFn = func(string) bool { return false }
	cmds, skipped = buildTodoCmds([]string{"a"}, true)
	if !skipped || len(cmds) != 1 {
		t.Fatalf("unexpected skipped result: %v %v", cmds, skipped)
	}

	checkNodeHealthFn = func(string, string, bool) error { return errors.New("bad") }
	getYesOrNoFn = func(string) bool { return true }
	cmds, skipped = buildTodoCmds([]string{"a"}, true)
	if !skipped || len(cmds) != 1 {
		t.Fatalf("unexpected unhealthy-continue result: %v %v", cmds, skipped)
	}

	resetGencmdHooks(t)
	osExitFn = func(int) { panic(exitPanic{}) }
	checkNodeHealthFn = func(string, string, bool) error { return errors.New("bad") }
	getYesOrNoFn = func(string) bool { return false }
	expectExitPanic(t, func() {
		_, _ = buildTodoCmds([]string{"a"}, true)
	})

	resetGencmdHooks(t)
	sleepFn = func(time.Duration) {}
	genConfigFn = func([]string) error { return nil }
	extractNodeFn = func(string) string { return "node1" }
	runTalosctlCommandFn = func([]string, bool) (string, error) { return "", nil }
	checkNodeHealthFn = func(string, string, bool) error { return nil }
	genPlainFn = func(command, node string, extra []string) []string { return []string{"talosctl " + command} }
	execCmdFn = func(string) {}
	if err := ExecCmds([]string{"talosctl apply -n node1"}, true); err != nil {
		t.Fatalf("unexpected ExecCmds error: %v", err)
	}

	if err := ExecCmds([]string{"talosctl upgrade -n node1"}, true); err != nil {
		t.Fatalf("unexpected ExecCmds error: %v", err)
	}
}

func TestRunNodeCommandAndPostHealth(t *testing.T) {
	resetGencmdHooks(t)
	runTalosctlCommandFn = func([]string, bool) (string, error) { return "", nil }
	runNodeCommand("talosctl get", "n1")

	calls := 0
	runTalosctlCommandFn = func(args []string, silent bool) (string, error) {
		calls++
		if calls == 1 {
			return "certificate signed by unknown authority", errors.New("cert")
		}
		return "", nil
	}
	runNodeCommand("talosctl get", "n1")

	resetGencmdHooks(t)
	getYesOrNoFn = func(string) bool { return true }
	runTalosctlCommandFn = func([]string, bool) (string, error) { return "other", errors.New("boom") }
	runNodeCommand("talosctl get", "n1")

	resetGencmdHooks(t)
	osExitFn = func(int) { panic(exitPanic{}) }
	getYesOrNoFn = func(string) bool { return false }
	runTalosctlCommandFn = func([]string, bool) (string, error) { return "other", errors.New("boom") }
	expectExitPanic(t, func() {
		runNodeCommand("talosctl get", "n1")
	})

	resetGencmdHooks(t)
	checkNodeHealthFn = func(string, string, bool) error { return nil }
	checkNodePostCommandHealth("n1")

	checkNodeHealthFn = func(string, string, bool) error { return errors.New("bad") }
	getYesOrNoFn = func(string) bool { return true }
	checkNodePostCommandHealth("n1")

	osExitFn = func(int) { panic(exitPanic{}) }
	getYesOrNoFn = func(string) bool { return false }
	expectExitPanic(t, func() {
		checkNodePostCommandHealth("n1")
	})
}

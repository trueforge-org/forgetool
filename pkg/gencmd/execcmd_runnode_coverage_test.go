package gencmd

import (
	"errors"
	"strings"
	"testing"
)

func TestRunNodeCommand_CertInErrorTriggersInsecureRetry(t *testing.T) {
	resetGencmdHooks(t)

	calls := 0
	seenInsecure := false
	runTalosctlCommandFn = func(args []string, silent bool) (string, error) {
		calls++
		for _, arg := range args {
			if arg == "--insecure" {
				seenInsecure = true
			}
		}
		if calls == 1 {
			return "other output", errors.New("certificate signed by unknown authority")
		}
		return "done", nil
	}

	runNodeCommand("talosctl get machinetype", "n1")
	if calls != 2 {
		t.Fatalf("expected retry call, got %d calls", calls)
	}
	if !seenInsecure {
		t.Fatal("expected insecure retry argument")
	}
}

func TestRunNodeCommand_InsecureRetryErrorBranch(t *testing.T) {
	resetGencmdHooks(t)

	calls := 0
	missingInsecure := false
	runTalosctlCommandFn = func(args []string, silent bool) (string, error) {
		calls++
		if calls == 1 {
			return "certificate signed by unknown authority", errors.New("cert")
		}
		if !strings.Contains(strings.Join(args, " "), "--insecure") {
			missingInsecure = true
		}
		return "", errors.New("still failing")
	}

	runNodeCommand("talosctl get machinetype", "n1")
	if calls != 2 {
		t.Fatalf("expected two calls, got %d", calls)
	}
	if missingInsecure {
		t.Fatal("expected insecure retry args on second call")
	}
}

func TestRunNodeCommand_CertInErrorRetryAlsoFails(t *testing.T) {
	resetGencmdHooks(t)

	calls := 0
	seenInsecure := false
	runTalosctlCommandFn = func(args []string, silent bool) (string, error) {
		calls++
		for _, arg := range args {
			if arg == "--insecure" {
				seenInsecure = true
			}
		}
		if calls == 1 {
			return "non-cert-output", errors.New("certificate signed by unknown authority")
		}
		return "", errors.New("retry still failing")
	}

	runNodeCommand("talosctl get machinetype", "n1")
	if calls != 2 {
		t.Fatalf("expected two calls, got %d", calls)
	}
	if !seenInsecure {
		t.Fatal("expected insecure flag on retry call")
	}
}

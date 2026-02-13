package talosctl

import (
	"runtime"
	"strings"
	"testing"
)

func TestRun_DefaultExecutorCoverage(t *testing.T) {
	previousExecutor := executor
	t.Cleanup(func() {
		executor = previousExecutor
	})

	out, err := Run([]string{"version"}, true)
	if err == nil && strings.TrimSpace(out) == "" {
		t.Fatalf("expected default executor to return output and/or error")
	}
}

func TestCommandPrefix(t *testing.T) {
	if got := CommandPrefix(); got != "talosctl" {
		t.Fatalf("expected command prefix talosctl, got %q", got)
	}
}

func TestSetExecutorAndRun(t *testing.T) {
	previousExecutor := executor
	t.Cleanup(func() {
		executor = previousExecutor
	})

	called := false
	SetExecutor(func(args []string, silent bool) (string, error) {
		called = true
		if len(args) != 2 || args[0] != "health" || args[1] != "--insecure" {
			t.Fatalf("unexpected args: %v", args)
		}
		if !silent {
			t.Fatalf("expected silent=true")
		}
		return "ok", nil
	})

	out, err := Run([]string{"health", "--insecure"}, true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out != "ok" {
		t.Fatalf("expected output ok, got %q", out)
	}
	if !called {
		t.Fatalf("expected custom executor to be called")
	}

	called = false
	SetExecutor(nil)
	_, _ = Run([]string{"health", "--insecure"}, true)
	if !called {
		t.Fatalf("expected executor to remain unchanged after SetExecutor(nil)")
	}
}

func TestRunCommandDelegatesWhenPrefixed(t *testing.T) {
	previousExecutor := executor
	t.Cleanup(func() {
		executor = previousExecutor
	})

	called := false
	SetExecutor(func(args []string, silent bool) (string, error) {
		called = true
		if len(args) != 1 || args[0] != "version" {
			t.Fatalf("unexpected delegated args: %v", args)
		}
		return "delegated", nil
	})

	out, err := RunCommand([]string{CommandPrefix(), "version"}, true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out != "delegated" {
		t.Fatalf("expected delegated output, got %q", out)
	}
	if !called {
		t.Fatalf("expected prefixed command to delegate to executor")
	}
}

func TestRunCommandBypassesExecutorWithoutPrefix(t *testing.T) {
	previousExecutor := executor
	t.Cleanup(func() {
		executor = previousExecutor
	})

	called := false
	SetExecutor(func(args []string, silent bool) (string, error) {
		called = true
		return "unexpected", nil
	})

	var command []string
	if runtime.GOOS == "windows" {
		command = []string{"cmd", "/C", "echo", "hello"}
	} else {
		command = []string{"echo", "hello"}
	}

	out, err := RunCommand(command, true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if called {
		t.Fatalf("expected non-prefixed command to bypass executor")
	}
	if !strings.Contains(out, "hello") {
		t.Fatalf("expected output to contain hello, got %q", out)
	}
}

func TestRunCommandReturnsErrorForEmptySlice(t *testing.T) {
	_, err := RunCommand([]string{}, true)
	if err == nil {
		t.Fatalf("expected error for empty commandSlice, got nil")
	}
	if !strings.Contains(err.Error(), "commandSlice cannot be empty") {
		t.Fatalf("expected error message to mention empty commandSlice, got: %v", err)
	}
}


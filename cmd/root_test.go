package cmd

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// captureStderr replaces os.Stderr for the duration of fn and returns the captured output.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	old := os.Stderr
	os.Stderr = w
	t.Cleanup(func() { os.Stderr = old })

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()
	_ = w.Close()
	return <-done
}

func TestExecute_Success(t *testing.T) {
	old := RootCmd
	t.Cleanup(func() { RootCmd = old })
	RootCmd = &cobra.Command{Use: "forgetool", SilenceUsage: true, SilenceErrors: true,
		Run: func(cmd *cobra.Command, args []string) {}}
	if err := Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecute_NonUsageError(t *testing.T) {
	old := RootCmd
	t.Cleanup(func() { RootCmd = old })
	want := errors.New("plain failure")
	RootCmd = &cobra.Command{Use: "forgetool", SilenceUsage: true, SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error { return want }}

	out := captureStderr(t, func() {
		if err := Execute(); err == nil {
			t.Fatalf("expected error")
		}
	})
	if !strings.Contains(out, "plain failure") {
		t.Fatalf("expected error printed to stderr, got %q", out)
	}
	if strings.Contains(out, "Usage:") {
		t.Fatalf("did not expect usage to be printed, got %q", out)
	}
}

func TestExecute_UsageErrorPrintsUsage(t *testing.T) {
	old := RootCmd
	t.Cleanup(func() { RootCmd = old })
	RootCmd = &cobra.Command{Use: "forgetool", SilenceUsage: true, SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error { return errors.New("unknown flag: --bogus") }}

	out := captureStderr(t, func() {
		if err := Execute(); err == nil {
			t.Fatalf("expected error")
		}
	})
	if !strings.Contains(out, "unknown flag") {
		t.Fatalf("expected error printed, got %q", out)
	}
	if !strings.Contains(out, "Usage:") {
		t.Fatalf("expected usage to be printed, got %q", out)
	}
}

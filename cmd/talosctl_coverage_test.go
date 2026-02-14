package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCaptureProcessIO_PipeFailures(t *testing.T) {
	oldPipeFn := talosctlPipeFn
	t.Cleanup(func() { talosctlPipeFn = oldPipeFn })

	calls := 0
	talosctlPipeFn = func() (*os.File, *os.File, error) {
		calls++
		if calls == 1 {
			return nil, nil, errors.New("pipe-stdout")
		}
		return oldPipeFn()
	}

	if _, err := captureProcessIO(func() error { return nil }); err == nil {
		t.Fatal("expected first pipe error")
	}

	calls = 0
	talosctlPipeFn = func() (*os.File, *os.File, error) {
		calls++
		if calls == 2 {
			return nil, nil, errors.New("pipe-stderr")
		}
		return oldPipeFn()
	}

	if _, err := captureProcessIO(func() error { return nil }); err == nil {
		t.Fatal("expected second pipe error")
	}
}

func TestCaptureProcessIO_CapturesStdoutStderrAndRunError(t *testing.T) {
	out, err := captureProcessIO(func() error {
		_, _ = fmt.Fprint(os.Stdout, "out")
		_, _ = fmt.Fprint(os.Stderr, "err")
		return errors.New("runfail")
	})
	if err == nil {
		t.Fatal("expected run error")
	}
	if !strings.Contains(out, "out") || !strings.Contains(out, "err") {
		t.Fatalf("expected combined stdout/stderr, got %q", out)
	}
}

func TestRunTalosctlArgs_NonSilentErrorPath(t *testing.T) {
	oldInternal := internalTalosctl
	t.Cleanup(func() { internalTalosctl = oldInternal })

	root := &cobra.Command{Use: "talosctl", SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(&cobra.Command{
		Use: "boom",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _ = fmt.Fprint(cmd.ErrOrStderr(), "stderr-out")
			return errors.New("boom")
		},
	})
	internalTalosctl = root

	out, err := runTalosctlArgs([]string{"boom"}, false)
	if err == nil {
		t.Fatal("expected command error")
	}
	if !strings.Contains(out, "stderr-out") {
		t.Fatalf("expected stderr output, got %q", out)
	}
}

func TestTalosctlCommandRunE_OutputAndEmptyOutput(t *testing.T) {
	oldInternal := internalTalosctl
	t.Cleanup(func() { internalTalosctl = oldInternal })

	root := &cobra.Command{Use: "talosctl", SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(&cobra.Command{
		Use: "emit",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _ = fmt.Fprint(cmd.OutOrStdout(), "from-run-e")
			return nil
		},
	})
	root.AddCommand(&cobra.Command{
		Use: "empty",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	})
	internalTalosctl = root

	buf := &bytes.Buffer{}
	cmdWithOutput := &cobra.Command{}
	cmdWithOutput.SetOut(buf)
	if err := talosctl.RunE(cmdWithOutput, []string{"emit"}); err != nil {
		t.Fatalf("unexpected RunE error for emit: %v", err)
	}
	if !strings.Contains(buf.String(), "from-run-e") {
		t.Fatalf("expected output from runTalosctlArgs path, got %q", buf.String())
	}

	emptyBuf := &bytes.Buffer{}
	cmdEmpty := &cobra.Command{}
	cmdEmpty.SetOut(emptyBuf)
	if err := talosctl.RunE(cmdEmpty, []string{"empty"}); err != nil {
		t.Fatalf("unexpected RunE error for empty: %v", err)
	}
	if emptyBuf.String() != "" {
		t.Fatalf("expected no output for empty command, got %q", emptyBuf.String())
	}
}

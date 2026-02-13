package cmd

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/siderolabs/talos/cmd/talosctl/cmd/mgmt"
	"github.com/siderolabs/talos/cmd/talosctl/cmd/talos"
	"github.com/spf13/cobra"
)

func TestTalosctlCommandConfig(t *testing.T) {
	if talosctl.Use != "talosctl" {
		t.Fatalf("expected use %q, got %q", "talosctl", talosctl.Use)
	}
	if !talosctl.SilenceUsage {
		t.Fatalf("expected SilenceUsage to be true")
	}
	if !talosctl.SilenceErrors {
		t.Fatalf("expected SilenceErrors to be true")
	}
	if !talosctl.DisableFlagParsing {
		t.Fatalf("expected DisableFlagParsing to be true")
	}
}

func TestRunTalosctlArgsSilentMode(t *testing.T) {
	out, err := runTalosctlArgs([]string{"--help"}, true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out == "" {
		t.Fatalf("expected help output")
	}
}

func TestInternalTalosctlHasExpectedSubcommands(t *testing.T) {
	registered := make(map[string]bool)
	for _, command := range internalTalosctl.Commands() {
		registered[command.Name()] = true
	}

	for _, command := range mgmt.Commands {
		if !registered[command.Name()] {
			t.Fatalf("expected mgmt command %q to be registered", command.Name())
		}
	}

	for _, command := range talos.Commands {
		if !registered[command.Name()] {
			t.Fatalf("expected talos command %q to be registered", command.Name())
		}
	}
}

func TestRunTalosctlArgsSilentCapturesProcessStdout(t *testing.T) {
	oldInternal := internalTalosctl
	t.Cleanup(func() {
		internalTalosctl = oldInternal
	})

	root := &cobra.Command{
		Use:           "talosctl",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(&cobra.Command{
		Use: "emit",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _ = fmt.Fprint(os.Stdout, "maintenance")
			return nil
		},
	})

	internalTalosctl = root

	out, err := runTalosctlArgs([]string{"emit"}, true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !strings.Contains(out, "maintenance") {
		t.Fatalf("expected output to contain maintenance, got %q", out)
	}
}

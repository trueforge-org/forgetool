package cmd

import "testing"

func TestFluxCommandConfig(t *testing.T) {
	if fluxCmd.Use != "flux" {
		t.Fatalf("expected use %q, got %q", "flux", fluxCmd.Use)
	}
	if !fluxCmd.SilenceUsage {
		t.Fatalf("expected SilenceUsage to be true")
	}
	if !fluxCmd.SilenceErrors {
		t.Fatalf("expected SilenceErrors to be true")
	}
}

func TestFluxCommandHasExpectedSubcommands(t *testing.T) {
	found := false
	for _, command := range fluxCmd.Commands() {
		if command.Name() == "bootstrap" {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected subcommand %q to be registered", "bootstrap")
	}
}

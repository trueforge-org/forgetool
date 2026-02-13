package cmd

import "testing"

func TestTalosCommandConfig(t *testing.T) {
	if talosCmd.Use != "talos" {
		t.Fatalf("expected use %q, got %q", "talos", talosCmd.Use)
	}
	if !talosCmd.SilenceUsage {
		t.Fatalf("expected SilenceUsage to be true")
	}
	if !talosCmd.SilenceErrors {
		t.Fatalf("expected SilenceErrors to be true")
	}
}

func TestTalosCommandHasExpectedSubcommands(t *testing.T) {
	expected := []string{"apply", "bootstrap", "health", "kubeconfig", "reset", "upgrade"}
	registered := make(map[string]bool)
	for _, command := range talosCmd.Commands() {
		registered[command.Name()] = true
	}

	for _, name := range expected {
		if !registered[name] {
			t.Fatalf("expected subcommand %q to be registered", name)
		}
	}
}

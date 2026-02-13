package cmd

import "testing"

func TestHelmreleaseCommandConfig(t *testing.T) {
	if helmrelease.Use != "helmrelease" {
		t.Fatalf("expected use %q, got %q", "helmrelease", helmrelease.Use)
	}
	if !helmrelease.SilenceUsage {
		t.Fatalf("expected SilenceUsage to be true")
	}
	if !helmrelease.SilenceErrors {
		t.Fatalf("expected SilenceErrors to be true")
	}
}

func TestHelmreleaseCommandHasExpectedSubcommands(t *testing.T) {
	expected := []string{"install", "upgrade"}
	registered := make(map[string]bool)
	for _, command := range helmrelease.Commands() {
		registered[command.Name()] = true
	}

	for _, name := range expected {
		if !registered[name] {
			t.Fatalf("expected subcommand %q to be registered", name)
		}
	}
}

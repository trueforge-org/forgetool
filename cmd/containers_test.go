package cmd

import "testing"

func TestContainerCommandConfig(t *testing.T) {
	if containerCmd.Use != "container" {
		t.Fatalf("expected use %q, got %q", "container", containerCmd.Use)
	}
	if !containerCmd.SilenceUsage {
		t.Fatalf("expected SilenceUsage to be true")
	}
	if !containerCmd.SilenceErrors {
		t.Fatalf("expected SilenceErrors to be true")
	}
}

func TestContainerCommandHasExpectedSubcommands(t *testing.T) {
	expected := []string{"test"}
	registered := make(map[string]bool)
	for _, command := range containerCmd.Commands() {
		registered[command.Name()] = true
	}

	for _, name := range expected {
		if !registered[name] {
			t.Fatalf("expected subcommand %q to be registered", name)
		}
	}
}

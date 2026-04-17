package cmd

import "testing"

func TestContainersCommandConfig(t *testing.T) {
	if containersCmd.Use != "containers" {
		t.Fatalf("expected use %q, got %q", "containers", containersCmd.Use)
	}
	if !containersCmd.SilenceUsage {
		t.Fatalf("expected SilenceUsage to be true")
	}
	if !containersCmd.SilenceErrors {
		t.Fatalf("expected SilenceErrors to be true")
	}
}

func TestContainersCommandHasExpectedSubcommands(t *testing.T) {
	expected := []string{"test"}
	registered := make(map[string]bool)
	for _, command := range containersCmd.Commands() {
		registered[command.Name()] = true
	}

	for _, name := range expected {
		if !registered[name] {
			t.Fatalf("expected subcommand %q to be registered", name)
		}
	}
}

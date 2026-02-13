package cmd

import "testing"

func TestClusterCommandConfig(t *testing.T) {
	if clusterCmd.Use != "cluster" {
		t.Fatalf("expected use %q, got %q", "cluster", clusterCmd.Use)
	}
	if !clusterCmd.SilenceUsage {
		t.Fatalf("expected SilenceUsage to be true")
	}
	if !clusterCmd.SilenceErrors {
		t.Fatalf("expected SilenceErrors to be true")
	}
}

func TestClusterCommandHasExpectedSubcommands(t *testing.T) {
	expected := []string{"genconfig", "init"}
	registered := make(map[string]bool)
	for _, command := range clusterCmd.Commands() {
		registered[command.Name()] = true
	}

	for _, name := range expected {
		if !registered[name] {
			t.Fatalf("expected subcommand %q to be registered", name)
		}
	}
}

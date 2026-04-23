package cmd

import "testing"

func TestAdvCommandConfig(t *testing.T) {
	if adv.Use != "adv" {
		t.Fatalf("expected use %q, got %q", "adv", adv.Use)
	}
	if !adv.SilenceUsage {
		t.Fatalf("expected SilenceUsage to be true")
	}
	if !adv.SilenceErrors {
		t.Fatalf("expected SilenceErrors to be true")
	}
}

func TestAdvCommandHasExpectedSubcommands(t *testing.T) {
	expected := []string{"gentooldocs"}
	registered := make(map[string]bool)
	for _, command := range adv.Commands() {
		registered[command.Name()] = true
	}

	for _, name := range expected {
		if !registered[name] {
			t.Fatalf("expected subcommand %q to be registered", name)
		}
	}
}

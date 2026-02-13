package cmd

import "testing"

func TestChartsCommandConfig(t *testing.T) {
	if charts.Use != "charts" {
		t.Fatalf("expected use %q, got %q", "charts", charts.Use)
	}
	if !charts.SilenceUsage {
		t.Fatalf("expected SilenceUsage to be true")
	}
	if !charts.SilenceErrors {
		t.Fatalf("expected SilenceErrors to be true")
	}
}

func TestChartsCommandHasExpectedSubcommands(t *testing.T) {
	expected := []string{"bump", "deps", "genchangelog", "genchartlist", "genmeta", "tagcleaner"}
	registered := make(map[string]bool)
	for _, command := range charts.Commands() {
		registered[command.Name()] = true
	}

	for _, name := range expected {
		if !registered[name] {
			t.Fatalf("expected subcommand %q to be registered", name)
		}
	}
}

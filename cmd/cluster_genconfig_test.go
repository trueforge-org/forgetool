package cmd

import "testing"

func TestGenConfigCommandConfig(t *testing.T) {
	if genConfig.Use != "genconfig" {
		t.Fatalf("expected use %q, got %q", "genconfig", genConfig.Use)
	}
	if genConfig.Run == nil {
		t.Fatalf("expected Run handler to be set")
	}
}

func TestGenConfigCommandRegisteredOnCluster(t *testing.T) {
	found := false
	for _, command := range clusterCmd.Commands() {
		if command.Name() == "genconfig" {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected genconfig command to be registered on cluster command")
	}
}

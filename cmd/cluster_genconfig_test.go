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

func TestRunClusterGenConfigCallsRunner(t *testing.T) {
	oldRunner := clusterGenConfigRunner
	t.Cleanup(func() { clusterGenConfigRunner = oldRunner })

	called := false
	clusterGenConfigRunner = func(args []string) error {
		called = true
		if len(args) != 2 || args[0] != "a" || args[1] != "b" {
			t.Fatalf("unexpected args: %#v", args)
		}
		return nil
	}

	runClusterGenConfig([]string{"a", "b"})

	if !called {
		t.Fatalf("expected runClusterGenConfig to call runner")
	}
}

func TestGenConfigCommandRunCallsRunner(t *testing.T) {
	oldRunner := clusterGenConfigRunner
	t.Cleanup(func() { clusterGenConfigRunner = oldRunner })

	called := false
	clusterGenConfigRunner = func(args []string) error {
		called = true
		if len(args) != 1 || args[0] != "chart" {
			t.Fatalf("unexpected args: %#v", args)
		}
		return nil
	}

	genConfig.Run(genConfig, []string{"chart"})

	if !called {
		t.Fatalf("expected command Run to call runner")
	}
}

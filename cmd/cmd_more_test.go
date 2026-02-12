package cmd

import (
	"os"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

func TestExecuteReturnsNil(t *testing.T) {
	oldArgs := os.Args
	oldClusterName := helper.ClusterName
	defer func() {
		os.Args = oldArgs
		helper.ClusterName = oldClusterName
	}()

	os.Args = []string{"forgetool"}
	if err := Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
}

func TestRootCmdHasExpectedSubcommands(t *testing.T) {
	expected := []string{"cluster", "talos", "flux", "charts", "info", "encrypt", "decrypt", "checkcrypt", "adv", "helmrelease"}
	cmds := RootCmd.Commands()
	registered := make(map[string]bool)
	for _, c := range cmds {
		registered[c.Name()] = true
	}
	for _, name := range expected {
		if !registered[name] {
			t.Errorf("expected subcommand %q to be registered on RootCmd", name)
		}
	}
}

func TestClusterFlagDefaultValue(t *testing.T) {
	f := RootCmd.PersistentFlags().Lookup("cluster")
	if f == nil {
		t.Fatal("expected --cluster flag to be defined")
	}
	if f.DefValue != "main" {
		t.Fatalf("expected default cluster flag value to be %q, got %q", "main", f.DefValue)
	}
}

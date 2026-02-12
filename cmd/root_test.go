package cmd

import (
	"os"
	"path/filepath"
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

func TestExecuteParsesClusterFlag(t *testing.T) {
	oldArgs := os.Args
	oldClusterName := helper.ClusterName
	oldClusterPath := helper.ClusterPath
	oldClusterEnvFile := helper.ClusterEnvFile
	oldTalConfigFile := helper.TalConfigFile
	oldTalosPath := helper.TalosPath
	oldKubernetesPath := helper.KubernetesPath
	oldTalosGenerated := helper.TalosGenerated
	oldTalosConfigFile := helper.TalosConfigFile
	oldTalSecretFile := helper.TalSecretFile
	defer func() {
		os.Args = oldArgs
		helper.ClusterName = oldClusterName
		helper.ClusterPath = oldClusterPath
		helper.ClusterEnvFile = oldClusterEnvFile
		helper.TalConfigFile = oldTalConfigFile
		helper.TalosPath = oldTalosPath
		helper.KubernetesPath = oldKubernetesPath
		helper.TalosGenerated = oldTalosGenerated
		helper.TalosConfigFile = oldTalosConfigFile
		helper.TalSecretFile = oldTalSecretFile
	}()

	os.Args = []string{"forgetool", "--cluster", "testcluster"}
	if err := Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if helper.ClusterName != "testcluster" {
		t.Fatalf("expected cluster name to be testcluster, got %s", helper.ClusterName)
	}
	if helper.ClusterPath != filepath.Join("./clusters", "testcluster") {
		t.Fatalf("unexpected cluster path: %s", helper.ClusterPath)
	}
}

func TestSmokeCmd(t *testing.T) {}

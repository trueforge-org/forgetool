package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

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

package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/siderolabs/talos/cmd/talosctl/cmd/common"
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

func TestExecuteWithUnknownCommandPrintsUsage(t *testing.T) {
	oldArgs := os.Args
	oldSuppressErrors := common.SuppressErrors
	oldStderr := os.Stderr
	defer func() {
		os.Args = oldArgs
		common.SuppressErrors = oldSuppressErrors
		os.Stderr = oldStderr
	}()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() failed: %v", err)
	}
	os.Stderr = w

	common.SuppressErrors = false
	os.Args = []string{"forgetool", "definitely-not-a-real-command"}
	if err := Execute(); err == nil {
		t.Fatal("expected Execute to return error for unknown command")
	}

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close stderr writer: %v", err)
	}
	outputBytes, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read stderr output: %v", err)
	}
	output := string(outputBytes)
	if !strings.Contains(output, "unknown command") {
		t.Fatalf("expected stderr to contain unknown command error, got: %q", output)
	}
	if !strings.Contains(output, "Usage:") {
		t.Fatalf("expected stderr to contain usage output, got: %q", output)
	}
}

func TestExecuteWithEmptyClusterDoesNotUpdatePaths(t *testing.T) {
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

	helper.ClusterPath = "sentinel-cluster-path"
	helper.ClusterEnvFile = "sentinel-cluster-env"
	helper.TalConfigFile = "sentinel-tal-config"
	helper.TalosPath = "sentinel-talos-path"
	helper.KubernetesPath = "sentinel-kubernetes-path"
	helper.TalosGenerated = "sentinel-generated"
	helper.TalosConfigFile = "sentinel-talosconfig"
	helper.TalSecretFile = "sentinel-talsecret"

	os.Args = []string{"forgetool", "--cluster", ""}
	if err := Execute(); err != nil {
		t.Fatalf("Execute returned error for empty cluster: %v", err)
	}

	if helper.ClusterName != "" {
		t.Fatalf("expected cluster name to be empty, got %q", helper.ClusterName)
	}
	if helper.ClusterPath != "sentinel-cluster-path" {
		t.Fatalf("expected cluster path to remain unchanged, got %q", helper.ClusterPath)
	}
	if helper.ClusterEnvFile != "sentinel-cluster-env" {
		t.Fatalf("expected cluster env path to remain unchanged, got %q", helper.ClusterEnvFile)
	}
	if helper.TalConfigFile != "sentinel-tal-config" {
		t.Fatalf("expected tal config path to remain unchanged, got %q", helper.TalConfigFile)
	}
	if helper.TalosPath != "sentinel-talos-path" {
		t.Fatalf("expected talos path to remain unchanged, got %q", helper.TalosPath)
	}
	if helper.KubernetesPath != "sentinel-kubernetes-path" {
		t.Fatalf("expected kubernetes path to remain unchanged, got %q", helper.KubernetesPath)
	}
	if helper.TalosGenerated != "sentinel-generated" {
		t.Fatalf("expected talos generated path to remain unchanged, got %q", helper.TalosGenerated)
	}
	if helper.TalosConfigFile != "sentinel-talosconfig" {
		t.Fatalf("expected talosconfig path to remain unchanged, got %q", helper.TalosConfigFile)
	}
	if helper.TalSecretFile != "sentinel-talsecret" {
		t.Fatalf("expected talsecret path to remain unchanged, got %q", helper.TalSecretFile)
	}
}

func TestSmokeCmd(t *testing.T) {}

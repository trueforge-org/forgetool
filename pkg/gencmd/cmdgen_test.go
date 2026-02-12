package gencmd

import (
	"path/filepath"
	"strings"
	"testing"

	talhelperCfg "github.com/budimanjojo/talhelper/v3/pkg/config"
	"github.com/trueforge-org/forgetool/pkg/helper"
	"github.com/trueforge-org/forgetool/pkg/talassist"
)

func withTalConfigFixture(t *testing.T) {
	t.Helper()
	talassist.TalConfig = &talhelperCfg.TalhelperConfig{
		ClusterName: "demo",
		Nodes: []talhelperCfg.Node{
			{Hostname: "cp1", IPAddress: "10.0.0.10"},
			{Hostname: "wk1", IPAddress: "10.0.0.11"},
		},
	}
}

func TestGenApply_AllNodes(t *testing.T) {
	withTalConfigFixture(t)
	oldGen := helper.TalosGenerated
	oldCfg := helper.TalosConfigFile
	helper.TalosGenerated = filepath.Join(t.TempDir(), "generated")
	helper.TalosConfigFile = filepath.Join(t.TempDir(), "talosconfig")
	t.Cleanup(func() {
		helper.TalosGenerated = oldGen
		helper.TalosConfigFile = oldCfg
	})

	cmds := GenApply("", nil)
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(cmds))
	}
	if !strings.Contains(cmds[0], "apply machineconfig") || !strings.Contains(cmds[1], "apply machineconfig") {
		t.Fatalf("expected apply machineconfig in commands: %v", cmds)
	}
	if !strings.Contains(cmds[0], "demo-cp1.yaml") || !strings.Contains(cmds[1], "demo-wk1.yaml") {
		t.Fatalf("expected node-specific files in commands: %v", cmds)
	}
}

func TestGenApply_SingleNode(t *testing.T) {
	withTalConfigFixture(t)
	oldGen := helper.TalosGenerated
	oldCfg := helper.TalosConfigFile
	helper.TalosGenerated = filepath.Join(t.TempDir(), "generated")
	helper.TalosConfigFile = filepath.Join(t.TempDir(), "talosconfig")
	t.Cleanup(func() {
		helper.TalosGenerated = oldGen
		helper.TalosConfigFile = oldCfg
	})

	cmds := GenApply("10.0.0.11", nil)
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if !strings.Contains(cmds[0], "demo-wk1.yaml") || !strings.Contains(cmds[0], "-n 10.0.0.11") {
		t.Fatalf("unexpected single-node command: %s", cmds[0])
	}
}

func TestGenPlainAndGenKubeUpgrade(t *testing.T) {
	withTalConfigFixture(t)
	oldCfg := helper.TalosConfigFile
	helper.TalosConfigFile = filepath.Join(t.TempDir(), "talosconfig")
	t.Cleanup(func() {
		helper.TalosConfigFile = oldCfg
	})

	all := GenPlain("health", "", nil)
	if len(all) != 2 {
		t.Fatalf("expected 2 health commands, got %d", len(all))
	}
	single := GenPlain("reboot", "10.0.0.10", []string{"--graceful", "true"})
	if len(single) != 1 {
		t.Fatalf("expected 1 reboot command, got %d", len(single))
	}
	if !strings.Contains(single[0], "reboot") || !strings.Contains(single[0], "--graceful true") {
		t.Fatalf("unexpected reboot command: %s", single[0])
	}

	k := GenKubeUpgrade("10.0.0.10")
	if !strings.Contains(k, "upgrade-k8s") || !strings.Contains(k, "-n 10.0.0.10") {
		t.Fatalf("unexpected kube upgrade command: %s", k)
	}
}

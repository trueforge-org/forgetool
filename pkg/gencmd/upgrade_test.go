package gencmd

import (
	"strings"
	"testing"

	"github.com/trueforge-org/forgetool/embed"
	"github.com/trueforge-org/forgetool/pkg/helper"
)

func TestGenKubeUpgrade_Format(t *testing.T) {
	oldCfg := helper.TalosConfigFile
	helper.TalosConfigFile = "/tmp/test/talosconfig"
	t.Cleanup(func() {
		helper.TalosConfigFile = oldCfg
	})

	result := GenKubeUpgrade("10.0.0.10")
	talosPath := embed.GetTalosExec()

	if !strings.HasPrefix(result, talosPath) {
		t.Fatalf("expected command to start with talos path %q, got: %s", talosPath, result)
	}
	if !strings.Contains(result, "upgrade-k8s") {
		t.Fatalf("expected command to contain 'upgrade-k8s', got: %s", result)
	}
	if !strings.Contains(result, "--talosconfig /tmp/test/talosconfig") {
		t.Fatalf("expected command to contain talosconfig path, got: %s", result)
	}
	if !strings.Contains(result, "-n 10.0.0.10") {
		t.Fatalf("expected command to contain node IP, got: %s", result)
	}
}

func TestGenKubeUpgrade_DifferentNodes(t *testing.T) {
	oldCfg := helper.TalosConfigFile
	helper.TalosConfigFile = "/tmp/test/talosconfig"
	t.Cleanup(func() {
		helper.TalosConfigFile = oldCfg
	})

	tests := []struct {
		name   string
		nodeIP string
	}{
		{"control plane node", "10.0.0.10"},
		{"worker node", "10.0.0.11"},
		{"different subnet", "192.168.1.100"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenKubeUpgrade(tt.nodeIP)
			if !strings.Contains(result, "-n "+tt.nodeIP) {
				t.Fatalf("expected command to target node %s, got: %s", tt.nodeIP, result)
			}
		})
	}
}

func TestGenKubeUpgrade_UsesHelperTalosConfigFile(t *testing.T) {
	oldCfg := helper.TalosConfigFile
	t.Cleanup(func() {
		helper.TalosConfigFile = oldCfg
	})

	helper.TalosConfigFile = "/custom/path/talosconfig"
	result := GenKubeUpgrade("10.0.0.10")
	if !strings.Contains(result, "--talosconfig /custom/path/talosconfig") {
		t.Fatalf("expected command to use custom talosconfig path, got: %s", result)
	}

	helper.TalosConfigFile = "/another/path/cfg"
	result = GenKubeUpgrade("10.0.0.10")
	if !strings.Contains(result, "--talosconfig /another/path/cfg") {
		t.Fatalf("expected command to reflect updated talosconfig path, got: %s", result)
	}
}

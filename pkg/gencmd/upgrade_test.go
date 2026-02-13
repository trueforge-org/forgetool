package gencmd

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	talhelperCfg "github.com/budimanjojo/talhelper/v3/pkg/config"
	"github.com/trueforge-org/forgetool/pkg/helper"
	talosctlpkg "github.com/trueforge-org/forgetool/pkg/talosctl"
)

func TestGenKubeUpgrade_Format(t *testing.T) {
	oldCfg := helper.TalosConfigFile
	helper.TalosConfigFile = "/tmp/test/talosconfig"
	t.Cleanup(func() {
		helper.TalosConfigFile = oldCfg
	})

	result := GenKubeUpgrade("10.0.0.10")
	talosPath := talosctlpkg.CommandPrefix()

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

func TestGenUpgrade_GeneratesCommandsAndReplacesTalosctlPath(t *testing.T) {
	resetGencmdHooks(t)

	called := 0
	generateUpgradeCommandFn = func(_ *talhelperCfg.TalhelperConfig, _ string, node string, extraFlags []string, force bool) error {
		called++
		if node != "10.0.0.10" {
			t.Fatalf("expected node 10.0.0.10, got %q", node)
		}
		if force {
			t.Fatal("expected force to be false")
		}

		seenPreserve := false
		for _, flag := range extraFlags {
			if flag == "--preserve" {
				seenPreserve = true
				break
			}
		}
		if !seenPreserve {
			t.Fatalf("expected --preserve in extra flags, got %v", extraFlags)
		}

		_, _ = os.Stdout.Write([]byte("talosctl upgrade --nodes 10.0.0.10;\n"))
		_, _ = os.Stdout.Write([]byte("talosctl version;\n"))
		return nil
	}

	result := GenUpgrade("10.0.0.10", []string{"--image", "ghcr.io/talos"})
	if called != 1 {
		t.Fatalf("expected generator to be called once, got %d", called)
	}
	if len(result) != 2 {
		t.Fatalf("expected two commands, got %v", result)
	}

	talosPath := talosctlpkg.CommandPrefix()
	if !strings.HasPrefix(result[0], talosPath+" upgrade --nodes 10.0.0.10") {
		t.Fatalf("expected first command to use talos path %q, got %q", talosPath, result[0])
	}
	if !strings.HasPrefix(result[1], talosPath+" version") {
		t.Fatalf("expected second command to use talos path %q, got %q", talosPath, result[1])
	}
}

func TestGenUpgrade_GeneratorErrorTriggersFatalAndExit(t *testing.T) {
	resetGencmdHooks(t)

	fatalCalls := 0
	var fatalErr error
	upgradeFatalFn = func(err error) {
		fatalCalls++
		fatalErr = err
	}
	osExitFn = func(int) { panic(exitPanic{}) }

	generateUpgradeCommandFn = func(*talhelperCfg.TalhelperConfig, string, string, []string, bool) error {
		return errors.New("generator failed")
	}

	expectExitPanic(t, func() {
		_ = GenUpgrade("10.0.0.10", nil)
	})

	if fatalCalls != 1 {
		t.Fatalf("expected fatal handler to be called once, got %d", fatalCalls)
	}
	if fatalErr == nil || !strings.Contains(fatalErr.Error(), "failed to generate talosctl upgrade command") {
		t.Fatalf("expected wrapped fatal error message, got %v", fatalErr)
	}
}

func TestGenUpgrade_PipeErrorTriggersFatalAndExit(t *testing.T) {
	resetGencmdHooks(t)

	fatalCalls := 0
	var fatalErr error
	upgradeFatalFn = func(err error) {
		fatalCalls++
		fatalErr = err
	}
	osExitFn = func(int) { panic(exitPanic{}) }
	upgradePipeFn = func() (*os.File, *os.File, error) {
		return nil, nil, errors.New("pipe failed")
	}

	expectExitPanic(t, func() {
		_ = GenUpgrade("10.0.0.10", nil)
	})

	if fatalCalls != 1 {
		t.Fatalf("expected fatal handler to be called once, got %d", fatalCalls)
	}
	if fatalErr == nil || !strings.Contains(fatalErr.Error(), "failed to create pipe") {
		t.Fatalf("expected pipe failure fatal error, got %v", fatalErr)
	}
}

func TestGenUpgrade_ReadErrorTriggersFatalAndExit(t *testing.T) {
	resetGencmdHooks(t)

	fatalCalls := 0
	var fatalErr error
	upgradeFatalFn = func(err error) {
		fatalCalls++
		fatalErr = err
	}
	osExitFn = func(int) { panic(exitPanic{}) }
	upgradeReadAllFn = func(_ io.Reader) ([]byte, error) {
		return nil, errors.New("read failed")
	}

	expectExitPanic(t, func() {
		_ = GenUpgrade("10.0.0.10", nil)
	})

	if fatalCalls != 1 {
		t.Fatalf("expected fatal handler to be called once, got %d", fatalCalls)
	}
	if fatalErr == nil || !strings.Contains(fatalErr.Error(), "failed to read command output") {
		t.Fatalf("expected read failure fatal error, got %v", fatalErr)
	}
}

func TestGenUpgrade_CoversCloseErrorBranches(t *testing.T) {
	resetGencmdHooks(t)

	readerCloseCalls := 0
	writerCloseCalls := 0
	upgradeCloseWriterFn = func(file *os.File) error {
		writerCloseCalls++
		_ = file.Close()
		return errors.New("writer close")
	}
	upgradeCloseReaderFn = func(_ *os.File) error {
		readerCloseCalls++
		return errors.New("reader close")
	}

	result := GenUpgrade("10.0.0.10", nil)
	if len(result) != 0 {
		t.Fatalf("expected no commands, got %v", result)
	}
	if writerCloseCalls != 1 {
		t.Fatalf("expected writer close hook once, got %d", writerCloseCalls)
	}
	if readerCloseCalls != 1 {
		t.Fatalf("expected reader close hook once, got %d", readerCloseCalls)
	}
}

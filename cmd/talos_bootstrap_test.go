package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	talhelperCfg "github.com/budimanjojo/talhelper/v3/pkg/config"
	"github.com/trueforge-org/forgetool/pkg/helper"
	"github.com/trueforge-org/forgetool/pkg/talassist"
)

func talosExecNameForTest() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	if goos == "windows" {
		if goarch == "amd64" {
			return "talosctl-windows-amd64.exe"
		}
		return "talosctl-windows-arm64.exe"
	}
	if goos == "linux" {
		if goarch == "amd64" {
			return "talosctl-linux-amd64"
		}
		return "talosctl-linux-arm64"
	}
	if goos == "freebsd" {
		if goarch == "amd64" {
			return "talosctl-freebsd-amd64"
		}
		return "talosctl-freebsd-arm64"
	}
	if goarch == "amd64" {
		return "talosctl-darwin-amd64"
	}
	return "talosctl-darwin-arm64"
}

func TestBootstrapFunc_TalosOnlyPath(t *testing.T) {
	oldStdin := os.Stdin
	oldCacheDir := helper.CacheDir
	oldTalConfigFile := helper.TalosConfigFile
	oldTalConfig := talassist.TalConfig
	defer func() {
		os.Stdin = oldStdin
		helper.CacheDir = oldCacheDir
		helper.TalosConfigFile = oldTalConfigFile
		talassist.TalConfig = oldTalConfig
	}()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe failed: %v", err)
	}
	if _, err := w.Write([]byte("n\n")); err != nil {
		t.Fatalf("write prompt input failed: %v", err)
	}
	_ = w.Close()
	os.Stdin = r

	helper.CacheDir = t.TempDir()
	helper.TalosConfigFile = filepath.Join(t.TempDir(), "talosconfig")
	talassist.TalConfig = &talhelperCfg.TalhelperConfig{Nodes: []talhelperCfg.Node{{Hostname: "cp1", IPAddress: "10.0.0.10"}}}

	execPath := filepath.Join(helper.CacheDir, talosExecNameForTest())
	if err := os.MkdirAll(filepath.Dir(execPath), 0755); err != nil {
		t.Fatalf("mkdir cache failed: %v", err)
	}
	if err := os.WriteFile(execPath, []byte("#!/bin/sh\necho bootstrap ok\n"), 0755); err != nil {
		t.Fatalf("write fake talos exec failed: %v", err)
	}

	bootstrapfunc(nil, []string{})
}

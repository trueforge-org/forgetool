package talhelperutil

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

type exitPanic struct{}

func resetTalhelperutilHooks(t *testing.T) {
	t.Helper()
	talhelperutilFatalFn = defaultTalhelperutilFatal
	talhelperutilExitFn = func(int) {}
}

func expectExitPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected exit panic")
		}
	}()
	fn()
}

func TestExtractIPs(t *testing.T) {
	resetTalhelperutilHooks(t)
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	cfg := strings.Join([]string{
		"nodes:",
		"  - hostname: cp1",
		"    ipAddress: 10.0.0.10",
		"    controlPlane: true",
		"  - hostname: wk1",
		"    ipAddress: 10.0.0.11",
		"    controlPlane: false",
	}, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(td, "config.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatalf("write config.yaml failed: %v", err)
	}

	helper.AllIPs = nil
	helper.ControlPlaneIPs = nil
	helper.WorkerIPs = nil
	ExtractIPs()

	if len(helper.AllIPs) != 2 || len(helper.ControlPlaneIPs) != 1 || len(helper.WorkerIPs) != 1 {
		t.Fatalf("unexpected extracted IP sets: all=%v cp=%v worker=%v", helper.AllIPs, helper.ControlPlaneIPs, helper.WorkerIPs)
	}
}

func TestLoadTalConfig_ErrorBranches(t *testing.T) {
	resetTalhelperutilHooks(t)
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	talhelperutilFatalFn = func(error, string) {}
	talhelperutilExitFn = func(int) { panic(exitPanic{}) }
	expectExitPanic(t, func() {
		_ = loadTalConfig()
	})

	resetTalhelperutilHooks(t)
	talhelperutilFatalFn = func(error, string) {}
	talhelperutilExitFn = func(int) { panic(exitPanic{}) }
	if err := os.WriteFile(filepath.Join(td, "config.yaml"), []byte("nodes: [bad"), 0644); err != nil {
		t.Fatalf("write bad config failed: %v", err)
	}
	expectExitPanic(t, func() {
		_ = loadTalConfig()
	})
}

func TestDefaultTalhelperutilFatal(t *testing.T) {
	defaultTalhelperutilFatal(errors.New("boom"), "fatal hook test")
}

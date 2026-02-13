package talassist

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	talhelperCfg "github.com/budimanjojo/talhelper/v3/pkg/config"
	"github.com/budimanjojo/talhelper/v3/pkg/generate"
	talhelperTalos "github.com/budimanjojo/talhelper/v3/pkg/talos"
	sideroConfig "github.com/siderolabs/talos/pkg/machinery/config"
	"github.com/siderolabs/talos/pkg/machinery/config/generate/secrets"
	"github.com/trueforge-org/forgetool/pkg/helper"
)

type exitPanic struct{}

func resetTalassistHooks(t *testing.T) {
	t.Helper()
	loadAndValidateFromFileFn = talhelperCfg.LoadAndValidateFromFile
	parseContractFromVersionFn = sideroConfig.ParseContractFromVersion
	newSecretBundleFn = talhelperTalos.NewSecretBundle
	generateConfigFn = generate.GenerateConfig
	talConfigGenerateGitignoreFn = defaultTalConfigGenerateGitignore
	mkdirAllFn = os.MkdirAll
	writeFileFn = os.WriteFile
	talassistFatalFn = defaultTalassistFatal
	talassistExitFn = func(int) {}
}

func expectExitPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected exit panic")
		}
	}()
	fn()
}

func TestLoadTalConfig_SuccessAndError(t *testing.T) {
	resetTalassistHooks(t)
	cfg := &talhelperCfg.TalhelperConfig{ClusterName: "demo"}
	loadAndValidateFromFileFn = func(string, []string, bool) (*talhelperCfg.TalhelperConfig, error) {
		return cfg, nil
	}
	LoadTalConfig()
	if TalConfig != cfg {
		t.Fatalf("expected TalConfig pointer to be assigned")
	}

	resetTalassistHooks(t)
	talassistFatalFn = func(error, string) {}
	talassistExitFn = func(int) { panic(exitPanic{}) }
	loadAndValidateFromFileFn = func(string, []string, bool) (*talhelperCfg.TalhelperConfig, error) {
		return nil, errors.New("bad config")
	}
	expectExitPanic(t, LoadTalConfig)
}

func TestGenSchema_WriteErrorAndSuccess(t *testing.T) {
	resetTalassistHooks(t)
	oldClusterPath := helper.ClusterPath
	helper.ClusterPath = t.TempDir()
	t.Cleanup(func() { helper.ClusterPath = oldClusterPath })

	writeFileFn = func(string, []byte, os.FileMode) error { return errors.New("write") }
	talassistFatalFn = func(error, string) {}
	talassistExitFn = func(int) { panic(exitPanic{}) }
	expectExitPanic(t, func() { _ = GenSchema() })

	resetTalassistHooks(t)
	helper.ClusterPath = t.TempDir()
	var wrotePath string
	writeFileFn = func(p string, b []byte, m os.FileMode) error {
		wrotePath = p
		return os.WriteFile(p, b, m)
	}
	if err := GenSchema(); err != nil {
		t.Fatalf("GenSchema success path failed: %v", err)
	}
	if wrotePath == "" {
		t.Fatal("expected schema path to be written")
	}
}

func TestNewSecretBundle_ErrorAndSuccess(t *testing.T) {
	resetTalassistHooks(t)
	LatestTalosVersion = "v1.7.0"
	parseContractFromVersionFn = func(string) (*sideroConfig.VersionContract, error) {
		v, _ := sideroConfig.ParseContractFromVersion("v1.7.0")
		return v, nil
	}
	newSecretBundleFn = func(secrets.Clock, sideroConfig.VersionContract) (*secrets.Bundle, error) {
		return nil, errors.New("bundle")
	}
	if got := NewSecretBundle(); got != nil {
		t.Fatal("expected nil bundle when bundle creation fails")
	}

	resetTalassistHooks(t)
	LatestTalosVersion = "v1.7.0"
	if got := NewSecretBundle(); got == nil {
		t.Fatal("expected non-nil bundle")
	}
}

func TestTalhelperGenConfig_ErrorAndSuccess(t *testing.T) {
	resetTalassistHooks(t)
	TalConfig = &talhelperCfg.TalhelperConfig{ClusterName: "demo"}
	generateConfigFn = func(*talhelperCfg.TalhelperConfig, bool, string, string, string, bool, bool) error {
		return errors.New("gen")
	}
	talassistFatalFn = func(error, string) {}
	talassistExitFn = func(int) { panic(exitPanic{}) }
	expectExitPanic(t, func() { _ = TalhelperGenConfig() })

	resetTalassistHooks(t)
	TalConfig = &talhelperCfg.TalhelperConfig{ClusterName: "demo"}
	generateConfigFn = func(*talhelperCfg.TalhelperConfig, bool, string, string, string, bool, bool) error { return nil }
	talConfigGenerateGitignoreFn = func(*talhelperCfg.TalhelperConfig, string) error { return errors.New("gitignore") }
	talassistFatalFn = func(error, string) {}
	talassistExitFn = func(int) { panic(exitPanic{}) }
	expectExitPanic(t, func() { _ = TalhelperGenConfig() })

	resetTalassistHooks(t)
	TalConfig = &talhelperCfg.TalhelperConfig{ClusterName: "demo"}
	oldTalosGenerated := helper.TalosGenerated
	oldTalSecretFile := helper.TalSecretFile
	helper.TalosGenerated = filepath.Join(t.TempDir(), "generated")
	helper.TalSecretFile = filepath.Join(t.TempDir(), "tal.secret")
	t.Cleanup(func() {
		helper.TalosGenerated = oldTalosGenerated
		helper.TalSecretFile = oldTalSecretFile
	})
	generateConfigFn = func(*talhelperCfg.TalhelperConfig, bool, string, string, string, bool, bool) error { return nil }
	talConfigGenerateGitignoreFn = func(*talhelperCfg.TalhelperConfig, string) error { return nil }
	if err := TalhelperGenConfig(); err != nil {
		t.Fatalf("TalhelperGenConfig success path failed: %v", err)
	}
}

func TestTalassistDefaultHelpers(t *testing.T) {
	if err := defaultTalConfigGenerateGitignore(nil, "out"); err == nil {
		t.Fatal("expected error for nil talconfig")
	}

	_ = defaultTalConfigGenerateGitignore(&talhelperCfg.TalhelperConfig{}, t.TempDir())

	defaultTalassistFatal(errors.New("fatal"), "test fatal hook")
}

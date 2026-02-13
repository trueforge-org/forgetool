package gencmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	talhelperCfg "github.com/budimanjojo/talhelper/v3/pkg/config"
	"github.com/trueforge-org/forgetool/pkg/helper"
	"github.com/trueforge-org/forgetool/pkg/talassist"
)

func TestRunBootstrap_ApplyErrorBranches(t *testing.T) {
	resetGencmdHooks(t)
	oldCfg := talassist.TalConfig
	t.Cleanup(func() { talassist.TalConfig = oldCfg })
	talassist.TalConfig = &talhelperCfg.TalhelperConfig{Nodes: []talhelperCfg.Node{{IPAddress: "10.0.0.1"}}}

	parseBootstrapExtraArgsFn = func([]string) []string { return nil }
	decryptBootstrapFilesFn = func() {}
	runBootstrapNodeLifecycleFn = func(string, []string) {}
	setupBootstrapClusterFn = func() (context.Context, chan struct{}, []string, []string, error) {
		return context.Background(), make(chan struct{}), []string{"ns"}, []string{"vsc"}, nil
	}

	calls := 0
	applyManifestFilesFn = func(context.Context, []string, string) error {
		calls++
		if calls == 1 {
			return errors.New("first apply fail")
		}
		return nil
	}
	RunBootstrap([]string{"bootstrap"})
	if calls != 1 {
		t.Fatalf("expected early return after first apply failure, calls=%d", calls)
	}

	calls = 0
	applyManifestFilesFn = func(context.Context, []string, string) error {
		calls++
		if calls == 2 {
			return errors.New("second apply fail")
		}
		return nil
	}
	RunBootstrap([]string{"bootstrap"})
	if calls != 2 {
		t.Fatalf("expected early return after second apply failure, calls=%d", calls)
	}
}

func TestExecCmdTimeoutAndInsecureRetryError(t *testing.T) {
	resetGencmdHooks(t)
	sleepFn = func(time.Duration) {}
	bootstrapRetryTimeout = 0
	nowFn = func() time.Time { return time.Unix(0, 0) }
	sinceFn = func(time.Time) time.Duration { return time.Hour }
	call := 0
	runTalosctlCommandFn = func([]string, bool) (string, error) {
		call++
		if call <= 2 {
			return "bootstrap is not available yet", errors.New("fail")
		}
		return "bootstrap is not available yet", nil
	}
	ExecCmd("talosctl bootstrap")

	resetGencmdHooks(t)
	call = 0
	runTalosctlCommandFn = func([]string, bool) (string, error) {
		call++
		if call == 1 {
			return "certificate signed by unknown authority", errors.New("cert")
		}
		return "retry fail", errors.New("retry")
	}
	runNodeCommand("talosctl get", "n1")
}

func TestGenConfigRemainingBranches(t *testing.T) {
	resetGencmdHooks(t)
	processCalls := 0
	processDirectoryFn = func(string) error {
		processCalls++
		if processCalls == 2 {
			return errors.New("second err")
		}
		return nil
	}
	if err := GenConfig(nil); err != nil {
		t.Fatalf("unexpected genconfig error: %v", err)
	}

	resetGencmdHooks(t)
	td := t.TempDir()
	helper.TalosGenerated = td
	helper.TalSecretFile = filepath.Join(td, "secret.yaml")
	createTalSecretFileFn = func(string) (*os.File, error) { return nil, errors.New("create fail") }
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on create error")
		}
	}()
	_ = genTalSecret()
}

func TestGenTalSecretEncodeWriteAndStatElse(t *testing.T) {
	resetGencmdHooks(t)
	oldVersion := talassist.LatestTalosVersion
	talassist.LatestTalosVersion = "v1.7.0"
	t.Cleanup(func() { talassist.LatestTalosVersion = oldVersion })
	td := t.TempDir()
	helper.TalosGenerated = td
	helper.TalSecretFile = filepath.Join(td, "secret.yaml")

	encodeSecretBundleFn = func(any) ([]byte, error) { return nil, errors.New("encode") }
	if err := genTalSecret(); err == nil {
		t.Fatal("expected encode error")
	}

	resetGencmdHooks(t)
	helper.TalosGenerated = td
	helper.TalSecretFile = filepath.Join(td, "secret2.yaml")
	writeTalSecretBytesFn = func(*os.File, []byte) (int, error) { return 0, errors.New("write") }
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on write error")
		}
	}()
	_ = genTalSecret()
}

func TestGenTalSecretStatElseBranchAndPlainApplyBranches(t *testing.T) {
	resetGencmdHooks(t)
	oldVersion := talassist.LatestTalosVersion
	talassist.LatestTalosVersion = "v1.7.0"
	t.Cleanup(func() { talassist.LatestTalosVersion = oldVersion })
	helper.TalSecretFile = string([]byte{'a', 0, 'b'})
	_ = genTalSecret()

	oldCfg := talassist.TalConfig
	t.Cleanup(func() { talassist.TalConfig = oldCfg })
	talassist.TalConfig = &talhelperCfg.TalhelperConfig{ClusterName: "demo", Nodes: []talhelperCfg.Node{{Hostname: "n1", IPAddress: "10.0.0.1"}}}
	oldTalCfg := helper.TalosConfigFile
	helper.TalosConfigFile = "/tmp/talosconfig"
	t.Cleanup(func() { helper.TalosConfigFile = oldTalCfg })

	allWithExtra := GenPlain("health", "", []string{"--x", "1"})
	if len(allWithExtra) != 1 {
		t.Fatalf("expected one command, got %v", allWithExtra)
	}
	singleNoExtra := GenPlain("health", "10.0.0.1", nil)
	if len(singleNoExtra) != 1 {
		t.Fatalf("expected one command, got %v", singleNoExtra)
	}

	resetGencmdHooks(t)
	oldTalCfg = helper.TalosConfigFile
	oldGen := helper.TalosGenerated
	helper.TalosConfigFile = "/tmp/talosconfig"
	helper.TalosGenerated = "/tmp/generated"
	t.Cleanup(func() {
		helper.TalosConfigFile = oldTalCfg
		helper.TalosGenerated = oldGen
	})
	talassist.TalConfig = &talhelperCfg.TalhelperConfig{ClusterName: "demo", Nodes: []talhelperCfg.Node{{Hostname: "n1", IPAddress: "10.0.0.1"}}}
	osExitFn = func(int) { panic(exitPanic{}) }
	expectExitPanic(t, func() { _ = GenApply("10.0.0.2", nil) })
}

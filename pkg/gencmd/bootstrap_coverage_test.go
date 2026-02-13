package gencmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	talhelperCfg "github.com/budimanjojo/talhelper/v3/pkg/config"
	"github.com/trueforge-org/forgetool/pkg/fluxhandler"
	"github.com/trueforge-org/forgetool/pkg/helper"
	"github.com/trueforge-org/forgetool/pkg/talassist"
	"k8s.io/client-go/kubernetes"
)

func TestParseBootstrapExtraArgs(t *testing.T) {
	if got := parseBootstrapExtraArgs([]string{"bootstrap"}); got != nil {
		t.Fatalf("expected nil extra args, got %v", got)
	}
	if got := parseBootstrapExtraArgs([]string{"bootstrap", "--a", "b"}); len(got) != 2 {
		t.Fatalf("expected 2 extra args, got %v", got)
	}
}

func TestDecryptBootstrapFiles(t *testing.T) {
	resetGencmdHooks(t)
	sopsDecryptFilesFn = func() error { return errors.New("boom") }
	decryptBootstrapFiles()
	sopsDecryptFilesFn = func() error { return nil }
	decryptBootstrapFiles()
}

func TestRunBootstrap_OrchestrationPaths(t *testing.T) {
	resetGencmdHooks(t)
	oldCfg := talassist.TalConfig
	t.Cleanup(func() { talassist.TalConfig = oldCfg })
	talassist.TalConfig = &talhelperCfg.TalhelperConfig{Nodes: []talhelperCfg.Node{{IPAddress: "10.0.0.1"}}}

	calls := make([]string, 0)
	parseBootstrapExtraArgsFn = func(args []string) []string { markCalledWithIndex(&calls, "parse"); return []string{"--x"} }
	decryptBootstrapFilesFn = func() { markCalledWithIndex(&calls, "decrypt") }
	runBootstrapNodeLifecycleFn = func(string, []string) { markCalledWithIndex(&calls, "lifecycle") }
	setupBootstrapClusterFn = func() (context.Context, chan struct{}, []string, []string, error) {
		markCalledWithIndex(&calls, "setup")
		return context.Background(), make(chan struct{}), []string{"a"}, []string{"b"}, nil
	}
	applyManifestFilesFn = func(context.Context, []string, string) error {
		markCalledWithIndex(&calls, "apply")
		return nil
	}
	finalizeBaseClusterFn = func(chan struct{}) { markCalledWithIndex(&calls, "finalize") }
	installBootstrapChartPhasesFn = func(context.Context, []string) { markCalledWithIndex(&calls, "charts") }
	fluxBootstrapFn = func(context.Context) { markCalledWithIndex(&calls, "flux") }

	RunBootstrap([]string{"bootstrap", "--x"})
	if len(calls) == 0 {
		t.Fatal("expected orchestration calls")
	}

	calls = calls[:0]
	setupBootstrapClusterFn = func() (context.Context, chan struct{}, []string, []string, error) {
		markCalledWithIndex(&calls, "setup")
		return nil, nil, nil, nil, errors.New("fail")
	}
	RunBootstrap([]string{"bootstrap"})
	if len(calls) != 4 {
		t.Fatalf("expected early return after setup error, got %v", calls)
	}
}

func TestRunBootstrapNodeLifecycleAndExit(t *testing.T) {
	resetGencmdHooks(t)
	oldVIP := helper.TalEnv["VIP_IP"]
	helper.TalEnv["VIP_IP"] = "1.2.3.4"
	t.Cleanup(func() { helper.TalEnv["VIP_IP"] = oldVIP })

	calls := 0
	waitForHealthFn = func(string, []string) (string, error) { calls++; return "", nil }
	genApplyFn = func(string, []string) []string { return []string{"apply"} }
	execCmdsFn = func([]string, bool) error { return nil }
	genPlainFn = func(command, node string, extra []string) []string { return []string{command + ":" + node} }
	execCmdFn = func(string) { calls++ }
	checkStatusFn = func([]string, []string, time.Duration) error { return nil }
	runBootstrapNodeLifecycle("10.0.0.1", []string{"--insecure"})
	if calls == 0 {
		t.Fatal("expected lifecycle hooks to be called")
	}

	osExitFn = func(int) { panic(exitPanic{}) }
	checkStatusFn = func([]string, []string, time.Duration) error { return errors.New("pods not ready") }
	expectExitPanic(t, func() {
		runBootstrapNodeLifecycle("10.0.0.1", nil)
	})
}

func TestSetupBootstrapClusterVariants(t *testing.T) {
	writeBootstrapChartsConfig(t)

	resetGencmdHooks(t)
	getClientsetFn = func() (*kubernetes.Clientset, error) { return nil, errors.New("no client") }
	if _, _, _, _, err := setupBootstrapCluster(); err == nil {
		t.Fatal("expected clientset error")
	}

	resetGencmdHooks(t)
	getClientsetFn = kubectlClientsetNoop
	loadBootstrapHelmReposFn = func() error { return errors.New("repos fail") }
	if _, _, _, _, err := setupBootstrapCluster(); err == nil {
		t.Fatal("expected repo load error")
	}

	resetGencmdHooks(t)
	getClientsetFn = kubectlClientsetNoop
	loadBootstrapHelmReposFn = func() error { return nil }
	collectBootstrapFilePathsFn = func() ([]string, []string, error) { return nil, nil, errors.New("walk fail") }
	if _, _, _, _, err := setupBootstrapCluster(); err == nil {
		t.Fatal("expected collect paths error")
	}

	resetGencmdHooks(t)
	getClientsetFn = kubectlClientsetNoop
	loadBootstrapHelmReposFn = func() error { return nil }
	collectBootstrapFilePathsFn = func() ([]string, []string, error) { return []string{"n"}, []string{"v"}, nil }
	ctx, stopCh, ns, vsc, err := setupBootstrapCluster()
	if err != nil || ctx == nil || stopCh == nil || len(ns) != 1 || len(vsc) != 1 {
		t.Fatalf("unexpected setup success values: %v %v %v %v %v", ctx, stopCh, ns, vsc, err)
	}
}

func TestLoadBootstrapHelmRepos(t *testing.T) {
	resetGencmdHooks(t)
	loadAllHelmReposFn = func(string) (map[string]*fluxhandler.HelmRepo, error) { return nil, errors.New("x") }
	if err := loadBootstrapHelmRepos(); err == nil {
		t.Fatal("expected error")
	}
	loadAllHelmReposFn = func(string) (map[string]*fluxhandler.HelmRepo, error) {
		return map[string]*fluxhandler.HelmRepo{"a": {}}, nil
	}
	if err := loadBootstrapHelmRepos(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCollectBootstrapFilePaths(t *testing.T) {
	resetGencmdHooks(t)
	oldCluster := helper.ClusterPath
	t.Cleanup(func() { helper.ClusterPath = oldCluster })

	td := t.TempDir()
	helper.ClusterPath = td
	if err := os.MkdirAll(filepath.Join(td, "a"), 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(td, "a", "namespace.yaml"), []byte("x"), 0o644)
	_ = os.WriteFile(filepath.Join(td, "a", "volumeSnapshotClass.yaml"), []byte("x"), 0o644)
	ns, vsc, err := collectBootstrapFilePaths()
	if err != nil || len(ns) != 1 || len(vsc) != 1 {
		t.Fatalf("unexpected collect result: %v %v %v", ns, vsc, err)
	}

	helper.ClusterPath = filepath.Join(td, "missing")
	if _, _, err := collectBootstrapFilePaths(); err == nil {
		t.Fatal("expected walk error")
	}
}

func TestApplyManifestFilesAndFinalize(t *testing.T) {
	resetGencmdHooks(t)
	kubectlApplyFn = func(context.Context, string) error { return nil }
	if err := applyManifestFiles(context.Background(), []string{"a", "b"}, "Manifest"); err != nil {
		t.Fatalf("unexpected apply error: %v", err)
	}

	osExitFn = func(int) { panic(exitPanic{}) }
	kubectlApplyFn = func(context.Context, string) error { return errors.New("fail") }
	expectExitPanic(t, func() {
		_ = applyManifestFiles(context.Background(), []string{"a"}, "Manifest")
	})

	resetGencmdHooks(t)
	called := false
	genPlainFn = func(command, node string, extra []string) []string { return []string{"healthcmd"} }
	execCmdFn = func(string) { called = true }
	stop := make(chan struct{})
	finalizeBaseCluster(stop)
	if !called {
		t.Fatal("expected health command execution")
	}
	select {
	case <-stop:
	default:
		t.Fatal("expected stop channel to be closed")
	}
}

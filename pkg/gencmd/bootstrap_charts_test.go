package gencmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/trueforge-org/forgetool/pkg/fluxhandler"
	"github.com/trueforge-org/forgetool/pkg/helper"
)

type bootstrapChartsFatalPanic struct{}

func expectBootstrapChartsFatalPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		r := recover()
		if _, ok := r.(bootstrapChartsFatalPanic); !ok {
			t.Fatalf("expected bootstrap charts fatal panic, got %v", r)
		}
	}()
	fn()
}

func TestBootstrapChartBuilders(t *testing.T) {
	writeBootstrapChartsConfig(t)

	oldCluster := helper.ClusterPath
	helper.ClusterPath = "/tmp/cluster"
	t.Cleanup(func() { helper.ClusterPath = oldCluster })

	c := newBootstrapHelmChart("/x", true, false)
	if c.ChartPath != "/x" || !c.Retry || c.Wait {
		t.Fatalf("unexpected chart config: %+v", c)
	}

	charts := baseBootstrapCharts()
	if len(charts) != 3 {
		t.Fatalf("expected 3 base charts, got %d", len(charts))
	}
	if charts[0].ChartPath != filepath.Join(helper.ClusterPath, "kubernetes/kube-system/cilium/app") || !charts[0].Wait {
		t.Fatalf("unexpected first base chart config: %+v", charts[0])
	}
}

func TestInstallBootstrapChartPhases(t *testing.T) {
	writeBootstrapChartsConfig(t)

	resetGencmdHooks(t)
	oldCluster := helper.ClusterPath
	oldVIP := helper.TalEnv["VIP_IP"]
	helper.ClusterPath = "/tmp/cluster"
	helper.TalEnv["VIP_IP"] = "10.0.0.2"
	t.Cleanup(func() {
		helper.ClusterPath = oldCluster
		helper.TalEnv["VIP_IP"] = oldVIP
	})

	phaseSizes := []int{}
	installChartsFn = func(charts []fluxhandler.HelmChart, _ map[string]*fluxhandler.HelmRepo, _ bool) {
		phaseSizes = append(phaseSizes, len(charts))
	}
	checkStatusFn = func([]string, []string, time.Duration) error { return nil }
	applyManifestFilesFn = func(context.Context, []string, string) error { return nil }

	installBootstrapChartPhases(context.Background(), []string{"vsc"})
	if len(phaseSizes) != 4 {
		t.Fatalf("expected 4 install phases, got %d", len(phaseSizes))
	}
	expected := []int{3, 10, 1, 4}
	for i := range expected {
		if phaseSizes[i] != expected[i] {
			t.Fatalf("phase %d expected %d charts, got %d", i, expected[i], phaseSizes[i])
		}
	}

	resetGencmdHooks(t)
	osExitFn = func(int) { panic(exitPanic{}) }
	checkStatusFn = func([]string, []string, time.Duration) error { return errors.New("wait fail") }
	expectExitPanic(t, func() {
		installBootstrapChartPhases(context.Background(), nil)
	})
}

func writeBootstrapChartsConfig(t *testing.T) {
	t.Helper()

	oldCacheDir := helper.CacheDir
	helper.CacheDir = t.TempDir()
	t.Cleanup(func() { helper.CacheDir = oldCacheDir })

	if err := os.MkdirAll(filepath.Join(helper.CacheDir, "kubernetes"), 0o755); err != nil {
		t.Fatalf("mkdir kubernetes dir failed: %v", err)
	}

	config, err := os.ReadFile(filepath.Join("testdata", "bootstrap", "charts.json"))
	if err != nil {
		t.Fatalf("read charts fixture failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(helper.CacheDir, "kubernetes", "charts.json"), config, 0o644); err != nil {
		t.Fatalf("write charts config failed: %v", err)
	}
}

func TestValidateBootstrapChartConfig(t *testing.T) {
	if err := validateBootstrapChartConfig(bootstrapChartConfig{}); err == nil {
		t.Fatal("expected error for empty chart config")
	}
	if err := validateBootstrapChartConfig(bootstrapChartConfig{
		Charts: []bootstrapChart{{Path: "", Stage: 1}},
	}); err == nil {
		t.Fatal("expected error for missing chart path")
	}
	if err := validateBootstrapChartConfig(bootstrapChartConfig{
		Charts: []bootstrapChart{{Path: "kubernetes/x", Stage: 9}},
	}); err == nil {
		t.Fatal("expected error for invalid chart stage")
	}
	if err := validateBootstrapChartConfig(bootstrapChartConfig{
		Charts: []bootstrapChart{{Path: "kubernetes/x", Stage: 1}},
	}); err != nil {
		t.Fatalf("expected valid chart config, got %v", err)
	}
}

func TestLoadBootstrapChartConfig_FatalBranches(t *testing.T) {
	resetGencmdHooks(t)
	oldCacheDir := helper.CacheDir
	helper.CacheDir = t.TempDir()
	t.Cleanup(func() {
		helper.CacheDir = oldCacheDir
		resetGencmdHooks(t)
	})

	bootstrapChartsFatalFn = func(error, string) { panic(bootstrapChartsFatalPanic{}) }

	if err := os.MkdirAll(filepath.Join(helper.CacheDir, "kubernetes"), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(helper.CacheDir, "kubernetes", "charts.json"), []byte("{"), 0o644); err != nil {
		t.Fatalf("write invalid json failed: %v", err)
	}
	expectBootstrapChartsFatalPanic(t, func() {
		_ = loadBootstrapChartConfig()
	})

	if err := os.WriteFile(filepath.Join(helper.CacheDir, "kubernetes", "charts.json"), []byte(`{"charts":[{"path":"","stage":1}]}`), 0o644); err != nil {
		t.Fatalf("write invalid schema failed: %v", err)
	}
	expectBootstrapChartsFatalPanic(t, func() {
		_ = loadBootstrapChartConfig()
	})

	if err := os.Remove(filepath.Join(helper.CacheDir, "kubernetes", "charts.json")); err != nil {
		t.Fatalf("remove charts.json failed: %v", err)
	}
	if err := os.Mkdir(filepath.Join(helper.CacheDir, "kubernetes", "charts.json"), 0o755); err != nil {
		t.Fatalf("create charts.json directory failed: %v", err)
	}
	expectBootstrapChartsFatalPanic(t, func() {
		_ = loadBootstrapChartConfig()
	})

	if err := os.RemoveAll(filepath.Join(helper.CacheDir, "kubernetes")); err != nil {
		t.Fatalf("remove kubernetes dir failed: %v", err)
	}
	expectBootstrapChartsFatalPanic(t, func() {
		_ = loadBootstrapChartConfig()
	})
}

func TestBootstrapChartsFatal_DefaultFunction(t *testing.T) {
	resetGencmdHooks(t)
	bootstrapChartsExitFn = func(int) { panic(bootstrapChartsFatalPanic{}) }
	t.Cleanup(func() {
		resetGencmdHooks(t)
	})

	expectBootstrapChartsFatalPanic(t, func() {
		bootstrapChartsFatalFn(errors.New("x"), "x")
	})
}

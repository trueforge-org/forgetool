package gencmd

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/trueforge-org/forgetool/pkg/fluxhandler"
	"github.com/trueforge-org/forgetool/pkg/helper"
)

func TestBootstrapChartBuilders(t *testing.T) {
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

func TestValidateBootstrapChartConfig(t *testing.T) {
	if err := validateBootstrapChartConfig(bootstrapChartConfig{}); err == nil {
		t.Fatal("expected error for empty chart config")
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

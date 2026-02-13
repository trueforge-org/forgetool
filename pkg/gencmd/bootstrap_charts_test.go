package gencmd

import (
	"context"
	"errors"
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

	installCalls := 0
	installChartsFn = func(_ []fluxhandler.HelmChart, _ map[string]*fluxhandler.HelmRepo, _ bool) {
		installCalls++
	}
	checkStatusFn = func([]string, []string, time.Duration) error { return nil }
	applyManifestFilesFn = func(context.Context, []string, string) error { return nil }

	installBootstrapChartPhases(context.Background(), []string{"vsc"})
	if installCalls != 4 {
		t.Fatalf("expected 4 install phases, got %d", installCalls)
	}

	resetGencmdHooks(t)
	osExitFn = func(int) { panic(exitPanic{}) }
	checkStatusFn = func([]string, []string, time.Duration) error { return errors.New("wait fail") }
	expectExitPanic(t, func() {
		installBootstrapChartPhases(context.Background(), nil)
	})
}

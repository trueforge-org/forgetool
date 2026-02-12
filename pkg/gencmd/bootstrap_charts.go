package gencmd

import (
	"context"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/trueforge-org/forgetool/pkg/fluxhandler"
	"github.com/trueforge-org/forgetool/pkg/helper"
	"github.com/trueforge-org/forgetool/pkg/kubectlcmds"
)

func baseBootstrapCharts() []fluxhandler.HelmChart {
	return []fluxhandler.HelmChart{
		newBootstrapHelmChart(filepath.Join(helper.ClusterPath, "/kubernetes/kube-system/cilium/app"), false, true),
		newBootstrapHelmChart(filepath.Join(helper.ClusterPath, "/kubernetes/kube-system/kubelet-csr-approver/app"), false, true),
		newBootstrapHelmChart(filepath.Join(helper.ClusterPath, "/kubernetes/system/kube-prometheus-stack/app"), false, false),
	}
}

func newBootstrapHelmChart(chartPath string, retry bool, wait bool) fluxhandler.HelmChart {
	return fluxhandler.HelmChart{
		ChartPath: chartPath,
		Retry:     retry,
		Wait:      wait,
	}
}

func installBootstrapChartPhases(ctx context.Context, vscFilePaths []string) {
	prioCharts := []fluxhandler.HelmChart{
		newBootstrapHelmChart(filepath.Join(helper.ClusterPath, "/kubernetes/system/spegel/app"), false, true),
		newBootstrapHelmChart(filepath.Join(helper.ClusterPath, "/kubernetes/system/cert-manager/app"), false, false),
		newBootstrapHelmChart(filepath.Join(helper.ClusterPath, "/kubernetes/system/kubernetes-reflector/app"), false, false),
	}
	fluxhandler.InstallCharts(prioCharts, HelmRepos, false)

	intermediateCharts := []fluxhandler.HelmChart{
		newBootstrapHelmChart(filepath.Join(helper.ClusterPath, "/kubernetes/system/metallb/app"), false, false),
		newBootstrapHelmChart(filepath.Join(helper.ClusterPath, "/kubernetes/core/clusterissuer/app"), false, false),
		newBootstrapHelmChart(filepath.Join(helper.ClusterPath, "/kubernetes/system/cloudnative-pg/app"), false, false),
		newBootstrapHelmChart(filepath.Join(helper.ClusterPath, "/kubernetes/kube-system/node-feature-discovery/app"), false, false),
		newBootstrapHelmChart(filepath.Join(helper.ClusterPath, "/kubernetes/kube-system/metrics-server/app"), false, false),
		newBootstrapHelmChart(filepath.Join(helper.ClusterPath, "/kubernetes/kube-system/descheduler/app"), false, false),
		newBootstrapHelmChart(filepath.Join(helper.ClusterPath, "/kubernetes/system/volsync/app"), false, true),
		newBootstrapHelmChart(filepath.Join(helper.ClusterPath, "/kubernetes/system/snapshot-controller/app"), false, true),
		newBootstrapHelmChart(filepath.Join(helper.ClusterPath, "/kubernetes/system/openebs/app"), false, true),
		newBootstrapHelmChart(filepath.Join(helper.ClusterPath, "/kubernetes/system/longhorn/app"), false, true),
	}
	fluxhandler.InstallCharts(intermediateCharts, HelmRepos, true)

	requiredMLBPods := []string{"metallb-controller", "metallb-speaker"}
	log.Info().Msgf("Bootstrap: Waiting for MetalLB Pods to be running for: %v", helper.TalEnv["VIP_IP"])
	if err := kubectlcmds.CheckStatus(requiredMLBPods, []string{}, 600); err != nil {
		log.Error().Err(err).Msgf("Error: %v\n", err)
		os.Exit(1)
	}

	lateCharts := []fluxhandler.HelmChart{newBootstrapHelmChart(filepath.Join(helper.ClusterPath, "/kubernetes/core/metallb-config/app"), false, false)}

	log.Info().Msgf("Bootstrap: Loading VolumeSnapshotClasses")
	_ = applyManifestFiles(ctx, vscFilePaths, "VolumeSnapshotClass")
	fluxhandler.InstallCharts(lateCharts, HelmRepos, true)

	log.Info().Msg("Bootstrap: Installing included applications")
	postCharts := []fluxhandler.HelmChart{
		newBootstrapHelmChart(filepath.Join(helper.ClusterPath, "/kubernetes/networking/nginx-internal/app"), false, true),
		newBootstrapHelmChart(filepath.Join(helper.ClusterPath, "/kubernetes/networking/nginx-external/app"), false, true),
		newBootstrapHelmChart(filepath.Join(helper.ClusterPath, "/kubernetes/core/blocky/app"), false, true),
		newBootstrapHelmChart(filepath.Join(helper.ClusterPath, "/kubernetes/apps/kubernetes-dashboard/app"), false, true),
	}
	fluxhandler.InstallCharts(postCharts, HelmRepos, true)
}

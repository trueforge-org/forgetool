package gencmd

import (
	"context"
	"path/filepath"

	"github.com/rs/zerolog/log"
	forgetoolembed "github.com/trueforge-org/forgetool/embed"
	"github.com/trueforge-org/forgetool/pkg/fluxhandler"
	"github.com/trueforge-org/forgetool/pkg/helper"
	"gopkg.in/yaml.v3"
)

func baseBootstrapCharts() []fluxhandler.HelmChart {
	return bootstrapPhaseCharts(loadBootstrapChartConfig().Base)
}

func newBootstrapHelmChart(chartPath string, retry bool, wait bool) fluxhandler.HelmChart {
	return fluxhandler.HelmChart{
		ChartPath: chartPath,
		Retry:     retry,
		Wait:      wait,
	}
}

func installBootstrapChartPhases(ctx context.Context, vscFilePaths []string) {
	config := loadBootstrapChartConfig()
	prioCharts := bootstrapPhaseCharts(config.Prio)
	installChartsFn(prioCharts, HelmRepos, false)

	intermediateCharts := bootstrapPhaseCharts(config.Intermediate)
	installChartsFn(intermediateCharts, HelmRepos, true)

	requiredMLBPods := []string{"metallb-controller", "metallb-speaker"}
	log.Info().Msgf("Bootstrap: Waiting for MetalLB Pods to be running for: %v", helper.TalEnv["VIP_IP"])
	if err := checkStatusFn(requiredMLBPods, []string{}, 600); err != nil {
		log.Error().Err(err).Msgf("Error: %v\n", err)
		osExitFn(1)
	}

	lateCharts := bootstrapPhaseCharts(config.Late)

	log.Info().Msgf("Bootstrap: Loading VolumeSnapshotClasses")
	_ = applyManifestFilesFn(ctx, vscFilePaths, "VolumeSnapshotClass")
	installChartsFn(lateCharts, HelmRepos, true)

	log.Info().Msg("Bootstrap: Installing included applications")
	postCharts := bootstrapPhaseCharts(config.Post)
	installChartsFn(postCharts, HelmRepos, true)
}

type bootstrapChartConfig struct {
	Base         []bootstrapChart `yaml:"base"`
	Prio         []bootstrapChart `yaml:"prio"`
	Intermediate []bootstrapChart `yaml:"intermediate"`
	Late         []bootstrapChart `yaml:"late"`
	Post         []bootstrapChart `yaml:"post"`
}

type bootstrapChart struct {
	Path  string `yaml:"path"`
	Retry bool   `yaml:"retry"`
	Wait  bool   `yaml:"wait"`
}

func loadBootstrapChartConfig() bootstrapChartConfig {
	data, err := forgetoolembed.GenericFiles.ReadFile("generic/kubernetes/bootstrap-charts.yaml")
	if err != nil {
		log.Fatal().Err(err).Msg("Bootstrap: failed to read embedded bootstrap chart config")
	}

	var config bootstrapChartConfig
	if err = yaml.Unmarshal(data, &config); err != nil {
		log.Fatal().Err(err).Msg("Bootstrap: failed to parse embedded bootstrap chart config")
	}
	return config
}

func bootstrapPhaseCharts(charts []bootstrapChart) []fluxhandler.HelmChart {
	phaseCharts := make([]fluxhandler.HelmChart, 0, len(charts))
	for _, chart := range charts {
		phaseCharts = append(phaseCharts, newBootstrapHelmChart(filepath.Join(helper.ClusterPath, chart.Path), chart.Retry, chart.Wait))
	}
	return phaseCharts
}

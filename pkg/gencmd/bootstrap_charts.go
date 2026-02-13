package gencmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/rs/zerolog/log"
	"github.com/trueforge-org/forgetool/pkg/fluxhandler"
	"github.com/trueforge-org/forgetool/pkg/helper"
)

func baseBootstrapCharts() []fluxhandler.HelmChart {
	return bootstrapPhaseCharts(loadBootstrapChartConfig().ChartsByStage[0])
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
	prioCharts := bootstrapPhaseCharts(config.ChartsByStage[1])
	installChartsFn(prioCharts, HelmRepos, false)

	intermediateCharts := bootstrapPhaseCharts(config.ChartsByStage[2])
	installChartsFn(intermediateCharts, HelmRepos, true)

	requiredMLBPods := []string{"metallb-controller", "metallb-speaker"}
	log.Info().Msgf("Bootstrap: Waiting for MetalLB Pods to be running for: %v", helper.TalEnv["VIP_IP"])
	if err := checkStatusFn(requiredMLBPods, []string{}, 600); err != nil {
		log.Error().Err(err).Msgf("Error: %v\n", err)
		osExitFn(1)
	}

	lateCharts := bootstrapPhaseCharts(config.ChartsByStage[3])

	log.Info().Msgf("Bootstrap: Loading VolumeSnapshotClasses")
	_ = applyManifestFilesFn(ctx, vscFilePaths, "VolumeSnapshotClass")
	installChartsFn(lateCharts, HelmRepos, true)

	log.Info().Msg("Bootstrap: Installing included applications")
	postCharts := bootstrapPhaseCharts(config.ChartsByStage[4])
	installChartsFn(postCharts, HelmRepos, true)
}

type bootstrapChartConfig struct {
	Charts        []bootstrapChart         `json:"charts"`
	ChartsByStage map[int][]bootstrapChart `json:"-"`
}

type bootstrapChart struct {
	Path  string `json:"path"`
	Retry bool   `json:"retry"`
	Wait  bool   `json:"wait"`
	Stage int    `json:"stage"`
}

func loadBootstrapChartConfig() bootstrapChartConfig {
	paths := []string{
		filepath.Join(helper.CacheDir, "kubernetes", "charts.json"),
		filepath.Join(helper.CacheDir, "kubernetes", "bootstrap-charts.json"),
		filepath.Join(helper.CacheDir, "charts.json"),
	}
	var (
		data []byte
		err  error
	)
	for _, path := range paths {
		data, err = os.ReadFile(path)
		if err == nil {
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			log.Fatal().Err(err).Msgf("Bootstrap: failed to read bootstrap chart config from %s", path)
		}
	}
	if err != nil {
		log.Fatal().Err(err).Msg("Bootstrap: failed to read bootstrap chart config from cluster-template cache")
	}

	var config bootstrapChartConfig
	if err = json.Unmarshal(data, &config); err != nil {
		log.Fatal().Err(err).Msg("Bootstrap: failed to parse embedded bootstrap chart config")
	}
	if err = validateBootstrapChartConfig(config); err != nil {
		log.Fatal().Err(err).Msg("Bootstrap: invalid bootstrap chart config schema")
	}
	config.ChartsByStage = make(map[int][]bootstrapChart)
	for _, chart := range config.Charts {
		config.ChartsByStage[chart.Stage] = append(config.ChartsByStage[chart.Stage], chart)
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

func validateBootstrapChartConfig(config bootstrapChartConfig) error {
	if len(config.Charts) == 0 {
		return fmt.Errorf("charts is required")
	}
	allowedStages := []int{0, 1, 2, 3, 4}
	for i, chart := range config.Charts {
		if chart.Path == "" {
			return fmt.Errorf("charts[%d].path is required", i)
		}
		if !slices.Contains(allowedStages, chart.Stage) {
			return fmt.Errorf("charts[%d].stage must be one of %v", i, allowedStages)
		}
	}
	return nil
}

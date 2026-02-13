package fluxhandler

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/rs/zerolog/log"
)

var (
	helmupgradeExitFn            = os.Exit
	helmupgradeLoadHelmReleaseFn = LoadHelmRelease
	helmupgradeHelmUpgradeFn     = HelmUpgrade
)

// UpgradeCharts upgrades Helm releases with provided Helm charts and repositories
func UpgradeCharts(charts []HelmChart, HelmRepos map[string]*HelmRepo, async bool) {
	var wg sync.WaitGroup
	for _, chart := range charts {
		wg.Add(1)
		go func(chart HelmChart) {
			defer wg.Done()
			processUpgradeChart(chart, HelmRepos, async)
		}(chart)

		if !async {
			wg.Wait()
		}
	}

	if async {
		wg.Wait()
	}
}

func processUpgradeChart(chart HelmChart, HelmRepos map[string]*HelmRepo, async bool) {
	valuesFile := filepath.Join(chart.ChartPath, "values.yaml")
	helmreleaseFile := filepath.Join(chart.ChartPath, "helm-release.yaml")

	helmRelease, err := helmupgradeLoadHelmReleaseFn(helmreleaseFile)
	if err != nil {
		log.Error().Err(err).Msgf("Error loading Helm release for chart at %s", chart.ChartPath)
		if !async {
			helmupgradeExitFn(1)
		}
		return
	}
	if helmRelease == nil {
		log.Error().Msgf("Empty Helm release for chart at %s\n", chart.ChartPath)
		if !async {
			helmupgradeExitFn(1)
		}
		return
	}

	releaseName := helmRelease.Metadata.Name
	if helmRelease.Spec.ReleaseName != "" {
		releaseName = helmRelease.Spec.ReleaseName
	}

	repoName := helmRelease.Spec.Chart.Spec.SourceRef.Name
	repo, ok := HelmRepos[repoName]
	if !ok || repo.Spec.URL == "" {
		log.Error().Msgf("Empty or invalid Helm repository for %s\n", repoName)
		if !async {
			helmupgradeExitFn(1)
		}
		return
	}

	log.Info().Msgf("Upgrading %s\n", helmRelease.Metadata.Name)
	err = helmupgradeHelmUpgradeFn(repo.Spec.URL, helmRelease.Spec.Chart.Spec.Chart, releaseName, helmRelease.Metadata.Namespace, valuesFile, helmRelease.Spec.Chart.Spec.Version, chart.Wait, true)
	if err != nil {
		log.Error().Err(err).Msgf("Error upgrading %s\n", helmRelease.Metadata.Name)
		if !async {
			helmupgradeExitFn(1)
		}
	}
}

package fluxhandler

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

var (
	helmreleaseExitFn            = os.Exit
	helmreleaseLoadHelmReleaseFn = LoadHelmRelease
	helmreleaseHelmInstallFn     = HelmInstall
)

type HelmChart struct {
	ChartPath string
	Retry     bool
	Wait      bool
}

type SourceRef struct {
	Kind      string `yaml:"kind,omitempty"`
	Name      string `yaml:"name,omitempty"`
	Namespace string `yaml:"namespace,omitempty"`
}

type ChartSpec struct {
	Chart     string    `yaml:"chart,omitempty"`
	Version   string    `yaml:"version,omitempty"`
	SourceRef SourceRef `yaml:"sourceRef,omitempty"`
}

type Chart struct {
	Spec ChartSpec `yaml:"spec,omitempty"`
}

type Spec struct {
	Interval    string                 `yaml:"interval,omitempty"`
	Chart       Chart                  `yaml:"chart,omitempty"`
	ReleaseName string                 `yaml:"releaseName,omitempty"`
	Values      map[string]interface{} `yaml:"values,omitempty"`
}

type Metadata struct {
	Name      string `yaml:"name,omitempty"`
	Namespace string `yaml:"namespace,omitempty"`
}

type HelmRelease struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata,omitempty"`
	Spec       Spec     `yaml:"spec,omitempty"`
}

func LoadHelmRelease(filename string) (*HelmRelease, error) {
	// Read YAML file
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Initialize HelmRelease struct
	config := &HelmRelease{}

	// Unmarshal YAML into struct
	err = yaml.Unmarshal(yamlFile, config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	// Ensure Values is not nil
	if config.Spec.Values == nil {
		config.Spec.Values = make(map[string]interface{})
	}
	return config, nil
}

func InstallCharts(charts []HelmChart, HelmRepos map[string]*HelmRepo, async bool) {
	var wg sync.WaitGroup
	for _, chart := range charts {
		wg.Add(1)
		go func(chart HelmChart) {
			defer wg.Done()
			processInstallChart(chart, HelmRepos, async)
		}(chart)
		if !async {
			wg.Wait()
		}
	}
	if async {
		wg.Wait()
	}
}

func processInstallChart(chart HelmChart, HelmRepos map[string]*HelmRepo, async bool) {
	valuesFile := filepath.Join(chart.ChartPath, "values.yaml")
	helmreleaseFile := filepath.Join(chart.ChartPath, "helm-release.yaml")
	releaseName, helmRelease, repoURL := resolveInstallChartContext(chart, helmreleaseFile, HelmRepos)

	log.Info().Msgf("Bootstrap: Installing %s\n", helmRelease.Metadata.Name)
	if err := helmreleaseHelmInstallFn(repoURL, helmRelease.Spec.Chart.Spec.Chart, releaseName, helmRelease.Metadata.Namespace, valuesFile, helmRelease.Spec.Chart.Spec.Version, chart.Retry, chart.Wait, true); err != nil {
		if strings.Contains(err.Error(), "webhook") {
			return
		}

		log.Error().Err(err).Msgf("Error: %v\n", err)
		if !async {
			helmreleaseExitFn(1)
		}
	}
}

func resolveInstallChartContext(chart HelmChart, helmreleaseFile string, helmRepos map[string]*HelmRepo) (string, *HelmRelease, string) {
	helmRelease, err := helmreleaseLoadHelmReleaseFn(helmreleaseFile)
	if err != nil {
		log.Info().Msgf("ERROR LOADING helmRelease for:  %v", chart)
		helmreleaseExitFn(1)
	}
	if helmRelease == nil {
		log.Info().Msgf("ERROR Empty helmRelease for:  %v", chart)
		helmreleaseExitFn(1)
	}

	releaseName := helmRelease.Metadata.Name
	if helmRelease.Spec.ReleaseName != "" {
		releaseName = helmRelease.Spec.ReleaseName
	}

	repoName := helmRelease.Spec.Chart.Spec.SourceRef.Name
	if helmRepos[repoName] == nil {
		log.Info().Msgf("ERROR Empty helmRepo for: %s", repoName)
		helmreleaseExitFn(1)
	}
	if helmRepos[repoName].Spec.URL == "" {
		log.Info().Msgf("ERROR Empty repoURL for: %s", repoName)
		helmreleaseExitFn(1)
	}

	return releaseName, helmRelease, helmRepos[repoName].Spec.URL
}

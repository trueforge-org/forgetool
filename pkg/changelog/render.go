package changelog

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/trueforge-org/forgetool/pkg/helper"
)

var loadChangedDataFileFunc = func(cd *ChangedData, path string) error { return cd.LoadFromFile(path) }
var walkChartsRenderFunc = helper.WalkCharts2
var renderChartChangelogFunc = func(o *ChangelogOptions, data *ChangedData, chartName, train string) error {
	return o.renderChartChangelog(data, chartName, train)
}

func (o *ChangelogOptions) Render() error {
	start := time.Now()
	log.Info().Msgf("Starting changelog render at %s", start)

	changelogData := ChangedData{mu: &sync.RWMutex{}, Charts: make(map[string]*Chart)}
	activeCharts := ActiveCharts{items: make(map[string]ActiveChart), mu: &sync.RWMutex{}}
	if err := loadChangedDataFileFunc(&changelogData, o.JSONOutputPath); err != nil {
		return fmt.Errorf("failed to load %s: %w", o.JSONOutputPath, err)
	}
	if err := walkChartsRenderFunc([]string{o.RepoPath}, activeCharts.getActiveChartsWalker, helper.AsyncMode); err != nil {
		return fmt.Errorf("failed to walk charts: %w", err)
	}

	for _, chart := range activeCharts.items {
		if err := renderChartChangelogFunc(o, &changelogData, chart.Name, chart.Train); err != nil {
			return fmt.Errorf("failed to render changelog for chart [%s]: %w", chart.Name, err)
		}
	}

	log.Info().Msgf("Finished in %s", time.Since(start))
	return nil
}

func (o *ChangelogOptions) renderChartChangelog(changelogData *ChangedData, chartName, train string) error {
	chartData := changelogData.Charts[chartName]
	if !o.hasRenderableChartData(chartData, chartName) {
		return nil
	}

	tmpl, err := template.ParseFiles(o.TemplatePath)
	if err != nil {
		return err
	}

	if err := o.prepareChartVersions(chartData); err != nil {
		return err
	}

	chartData.Name = chartName
	chartData.Train = train

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, chartData); err != nil {
		return err
	}

	output := filepath.Join(o.ChartsDir, train, chartName)
	if err := os.MkdirAll(output, os.ModePerm); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(output, o.ChangelogFileName), buf.Bytes(), 0644); err != nil {
		return err
	}

	return nil
}

func (o *ChangelogOptions) hasRenderableChartData(chartData *Chart, chartName string) bool {
	if chartData == nil {
		log.Error().Msgf("chart [%s] not found in %s", chartName, o.JSONOutputPath)
		return false
	}
	if chartData.Versions == nil {
		log.Error().Msgf("chart [%s] has no versions in %s", chartName, o.JSONOutputPath)
		return false
	}

	return true
}

func (o *ChangelogOptions) prepareChartVersions(chartData *Chart) error {
	if _, err := chartData.SortVersions(true); err != nil {
		return err
	}

	for _, version := range chartData.Versions {
		sortedCommits, err := version.SortCommits(true)
		if err != nil {
			return err
		}
		version.SortedCommits = sortedCommits
	}

	return nil
}

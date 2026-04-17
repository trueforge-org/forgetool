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
var walkAppsRenderFunc = helper.WalkCharts2
var renderAppChangelogFunc = func(o *ChangelogOptions, data *ChangedData, appName, train string) error {
	return o.renderAppChangelog(data, appName, train)
}

func (o *ChangelogOptions) Render() error {
	start := time.Now()
	if err := configureForAppType(o.AppType); err != nil {
		return err
	}
	log.Info().Msgf("Starting changelog render at %s", start)

	changelogData := ChangedData{mu: &sync.RWMutex{}, Apps: make(map[string]*App)}
	activeApps := ActiveApps{items: make(map[string]ActiveApp), mu: &sync.RWMutex{}}
	if err := loadChangedDataFileFunc(&changelogData, o.JSONOutputPath); err != nil {
		return fmt.Errorf("failed to load %s: %w", o.JSONOutputPath, err)
	}
	if err := walkAppsRenderFunc([]string{o.RepoPath}, activeApps.getActiveAppsWalker, helper.AsyncMode); err != nil {
		return fmt.Errorf("failed to walk apps: %w", err)
	}

	for _, app := range activeApps.items {
		if err := renderAppChangelogFunc(o, &changelogData, app.Name, app.Train); err != nil {
			return fmt.Errorf("failed to render changelog for app [%s]: %w", app.Name, err)
		}
	}

	log.Info().Msgf("Finished in %s", time.Since(start))
	return nil
}

func (o *ChangelogOptions) renderAppChangelog(changelogData *ChangedData, appName, train string) error {
	appData := changelogData.Apps[appName]
	if !o.hasRenderableAppData(appData, appName) {
		return nil
	}

	tmpl, err := template.ParseFiles(o.TemplatePath)
	if err != nil {
		return err
	}

	if err := o.prepareAppVersions(appData); err != nil {
		return err
	}

	appData.Name = appName
	appData.Train = train

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, appData); err != nil {
		return err
	}

	output := renderOutputPathFunc(o.AppsDir, train, appName)
	if err := os.MkdirAll(output, os.ModePerm); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(output, o.ChangelogFileName), buf.Bytes(), 0644); err != nil {
		return err
	}

	return nil
}

func (o *ChangelogOptions) hasRenderableAppData(appData *App, appName string) bool {
	if appData == nil {
		log.Error().Msgf("app [%s] not found in %s", appName, o.JSONOutputPath)
		return false
	}
	if appData.Versions == nil {
		log.Error().Msgf("app [%s] has no versions in %s", appName, o.JSONOutputPath)
		return false
	}

	return true
}

func (o *ChangelogOptions) prepareAppVersions(appData *App) error {
	if _, err := appData.SortVersions(true); err != nil {
		return err
	}

	for _, version := range appData.Versions {
		sortedCommits, err := version.SortCommits(true)
		if err != nil {
			return err
		}
		version.SortedCommits = sortedCommits
	}

	return nil
}

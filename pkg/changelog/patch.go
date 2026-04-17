package changelog

import (
	"errors"
	"fmt"

	"github.com/Masterminds/semver/v3"
	"github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rs/zerolog/log"
)

var errSkipPatch = errors.New("skip patch")
var getAppPathFunc = chartGetAppPath
var getAppVersionFunc = getAppVersion
var getFilePatchesFunc = func(p *object.Patch) []diff.FilePatch { return p.FilePatches() }
var getChangedFilePairFunc = getChangedFilePair
var getOldAndNewVersionFunc = getOldAndNewVersion

func getChangedFilePair(p diff.FilePatch) (string, oldNewPaths, error) {
	old, new := p.Files()
	if new == nil { // No new file, nothing to do
		log.Debug().Msgf("Skipping file patch. Reason: New file is empty")
		return "", oldNewPaths{}, errSkipPatch
	}

	// Get app name and check if its an active app
	// if the new.Path() is a path outside of the apps folder,
	// it will not be an active app anyway and so we skip the diff
	appName := getAppNameFunc(new.Path())
	if appName == invalidName || !activeApps.isActiveApp(appName) {
		log.Debug().Msgf("Skipping file patch. Reason: [%s] is not an active app", new.Path())
		return "", oldNewPaths{}, errSkipPatch
	}
	if _, err := getAppPathFunc(new.Path()); err != nil {
		log.Debug().Msgf("Skipping file patch. Reason: [%s] is not a valid app path", new.Path())
		return "", oldNewPaths{}, errSkipPatch
	}
	if old != nil { // If an old file exists in the patch
		if _, err := getAppPathFunc(old.Path()); err != nil {
			log.Debug().Msgf("Skipping file patch. Reason: [%s] is not a valid app path", old.Path())
			return "", oldNewPaths{}, errSkipPatch
		}
	}

	return appName, oldNewPaths{new: new, old: old}, nil
}

func getOldAndNewVersion(c *object.Commit, par *object.Commit, paths oldNewPaths) (string, string, error) {
	newAppPath, err := getAppPathFunc(paths.new.Path())
	if err != nil {
		return "", "", fmt.Errorf("failed to get app path from file path [%s]: %w", paths.new.Path(), err)
	}
	newAppVer, err := getAppVersionFunc(c, newAppPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to get app data from path [%s]: %w", newAppPath, err)
	}

	oldAppVer := ""
	if paths.old != nil { // If an old file exists in the patch
		oldAppPath, err := getAppPathFunc(paths.old.Path())
		if err != nil {
			return "", "", fmt.Errorf("failed to get app path from file path [%s]: %w", paths.old.Path(), err)
		}
		// Note here we pass the parent commit, not the current commit
		oldAppVer, err = getAppVersionFunc(par, oldAppPath)
		if err != nil {
			return "", "", fmt.Errorf("failed to get app data from path [%s]: %w", oldAppPath, err)
		}
	}

	return oldAppVer, newAppVer, nil
}

func getAppsWithMultipleChangedFiles(p *object.Patch) (appsWithChangedFiles, error) {
	appsWithMultipleFiles := make(appsWithChangedFiles)
	for _, p := range getFilePatchesFunc(p) {
		// Get app name and the "new" file path
		appName, paths, err := getChangedFilePairFunc(p)
		if err != nil {
			if errors.Is(err, errSkipPatch) {
				continue
			}
			return appsWithChangedFiles{}, fmt.Errorf("failed to get changed files: %w", err)
		}
		// if there is no new file, skip the filePatch
		if paths.new.Path() == "" {
			continue
		}

		// Add the file to the apps changed files
		appsWithMultipleFiles[appName] = append(appsWithMultipleFiles[appName], paths)
	}

	return appsWithMultipleFiles, nil
}

func getAppsWithSingleChangedFile(c appsWithChangedFiles) appsWithChangedFile {
	appsWithSingleFile := make(appsWithChangedFile)
	for appName, filePaths := range c {
		for _, paths := range filePaths {
			_, ok := appsWithSingleFile[appName]
			// If the app hasn't been seen before,
			// or the filePath is a Chart.yaml file
			// we add the pair to the map
			if !ok || isPreferredFileFunc(paths.new.Path()) {
				appsWithSingleFile[appName] = paths
				continue
			}
		}
	}
	return appsWithSingleFile
}

func processAppsWithSingleChangedFile(c *object.Commit, par *object.Commit, appsWithSingleFile appsWithChangedFile) error {
	// For each app, get the old and new versions
	for appName, paths := range appsWithSingleFile {
		oldVer, newVer, err := getOldAndNewVersionFunc(c, par, paths)
		if err != nil {
			return fmt.Errorf("failed to get old and new versions: %w", err)
		}
		// If the old version is empty, (app addition)
		// we add the new version to the changedData
		if oldVer == "" {
			addAppToChangedData(appName, newVer, paths.new.Path(), c)
			continue
		}

		oldSemVer, err := semver.NewVersion(oldVer)
		if err != nil {
			return fmt.Errorf("failed to parse old version ([%s]) for file [%s] in commit [%s]: %w", oldVer, paths.new.Path(), c.Hash.String(), err)
		}
		newSemVer, err := semver.NewVersion(newVer)
		if err != nil {
			return fmt.Errorf("failed to parse new version ([%s]) for file [%s] in commit [%s]: %w", newVer, paths.new.Path(), c.Hash.String(), err)
		}

		// if new version is greater than the old version, we add the new version to the changedData
		if newSemVer.GreaterThan(oldSemVer) {
			addAppToChangedData(appName, newVer, paths.new.Path(), c)
			continue
		}

		// Otherwise, we add the new version to the stagingData
		// It is probably less or equal to the old version,
		// in either case the app changes is unreleased.
		// so it should go to the "next" version, we do that at the end
		// although if its less, it will be hard to actually get which is the "next" version
		// but we can't really do anything about it, so just put it on the immediate next version
		addAppToStagingData(appName, newVer, paths.new.Path(), c)
	}
	return nil
}

func addAppToChangedData(appName, version, appPath string, c *object.Commit) {
	changedData.mu.Lock()
	changedData.AddOrUpdateApp(appName, version, getAppTrainFunc(appPath), c)
	changedData.mu.Unlock()
}

func addAppToStagingData(appName, version, appPath string, c *object.Commit) {
	stagingData.mu.Lock()
	stagingData.AddOrUpdateApp(appName, version, getAppTrainFunc(appPath), c)
	stagingData.mu.Unlock()
}

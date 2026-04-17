package changelog

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
)

const chartManifestFile = "Chart.yaml"

var chartAppFilePathRegex = regexp.MustCompile(`^charts/([\w-_]+)/([\w-_]+)/Chart.yaml$`)

var chartCharsToRemove = []string{"-"}

func chartGetAppName(path string) string {
	// path = charts/<train>/<app>/...
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		log.Debug().Msgf("failed to get app name from path [%s]", path)
		return invalidName
	}
	return parts[2]
}

func chartGetAppTrain(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		log.Error().Msgf("Could not get app train from path [%s]", path)
		return ""
	}
	return parts[1]
}

func chartGetAppPath(path string) (string, error) {
	original := path
	for {
		if path == "." {
			return "", fmt.Errorf("path too short [%s], or could not construct app path", original)
		}
		if chartAppFilePathRegex.MatchString(filepath.Join(path, chartManifestFile)) {
			return filepath.Join(path, chartManifestFile), nil
		}
		path = filepath.Dir(path)
	}
}

// chartGetVersion parses the version from a Chart.yaml content string.
// We use this instead of NewHelmChart.Load(), because
// this will work even if the Chart.yaml is malformed
func chartGetVersion(strData string) (string, error) {
	lines := strings.Split(strData, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "version:") {
			ver := strings.TrimSpace(strings.Split(line, ":")[1])
			for _, c := range chartCharsToRemove {
				ver = strings.ReplaceAll(ver, c, "")
			}
			return ver, nil
		}
	}
	return "", fmt.Errorf("could not find version in file")
}

func chartIsPreferredFile(path string) bool {
	return strings.HasSuffix(path, chartManifestFile)
}

func chartRenderOutputPath(appsDir, train, appName string) string {
	return filepath.Join(appsDir, train, appName)
}

func chartParseActiveApp(a *ActiveApps, path string) error {
	// path = <root>/<train>/<app>/Chart.yaml
	segLen := len(strings.Split(path, "/"))
	if segLen < 3 {
		return fmt.Errorf("path (%s) is not valid. expected at least <root>/<train>/<app>/Chart.yaml", path)
	}
	appDir, _ := filepath.Split(path)
	appDir = strings.TrimSuffix(appDir, "/")
	train := filepath.Dir(appDir)
	train = filepath.Base(train)
	appName := filepath.Base(appDir)
	a.mu.Lock()
	if _, ok := a.items[appName]; !ok {
		a.items[appName] = ActiveApp{Name: appName, Train: train}
	} else {
		log.Error().Msgf("app [%s] already exists in activeApps", appName)
	}
	a.mu.Unlock()
	return nil
}

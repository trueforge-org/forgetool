package changelog

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/trueforge-org/forgetool/pkg/version"
)

const containerManifestFile = "docker-bake.hcl"

var containerAppFilePathRegex = regexp.MustCompile(`^apps/([\w-_]+)/docker-bake\.hcl$`)

var containerVersionVarRe = regexp.MustCompile(`^\s*variable\s+"VERSION"\s*\{`)
var containerVersionDefaultRe = regexp.MustCompile(`^\s*default\s*=\s*"(.+)"`)

func containerGetAppName(path string) string {
	// path = apps/<container>/...
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		log.Debug().Msgf("failed to get container name from path [%s]", path)
		return invalidName
	}
	return parts[1]
}

func containerGetAppTrain(_ string) string {
	return ""
}

func containerGetAppPath(path string) (string, error) {
	original := path
	for {
		if path == "." {
			return "", fmt.Errorf("path too short [%s], or could not construct app path", original)
		}
		if containerAppFilePathRegex.MatchString(filepath.Join(path, containerManifestFile)) {
			return filepath.Join(path, containerManifestFile), nil
		}
		path = filepath.Dir(path)
	}
}

// containerGetVersion parses the VERSION variable from a docker-bake.hcl content string
// and returns a sanitised semantic version (major.minor.patch).
func containerGetVersion(strData string) (string, error) {
	lines := strings.Split(strData, "\n")
	inVersionBlock := false
	for _, line := range lines {
		if containerVersionVarRe.MatchString(line) {
			inVersionBlock = true
			continue
		}
		if inVersionBlock {
			if m := containerVersionDefaultRe.FindStringSubmatch(line); m != nil {
				info := version.Sanitize(m[1])
				log.Debug().Msgf("sanitised container version: upstream=%q semantic=%q raw=%q valid=%v",
					info.Upstream, info.Semantic, info.Raw, info.IsValidSemver)
				return info.Semantic, nil
			}
			if strings.Contains(line, "}") {
				inVersionBlock = false
			}
		}
	}
	return "", fmt.Errorf("could not find VERSION variable in file")
}

func containerIsPreferredFile(path string) bool {
	return strings.HasSuffix(path, containerManifestFile)
}

func containerRenderOutputPath(appsDir, _ string, appName string) string {
	return filepath.Join(appsDir, appName)
}

func containerParseActiveApp(a *ActiveApps, path string) error {
	// path = <root>/<container>/docker-bake.hcl
	segLen := len(strings.Split(path, "/"))
	if segLen < 2 {
		return fmt.Errorf("path (%s) is not valid. expected at least <root>/<app>/docker-bake.hcl", path)
	}
	appDir, _ := filepath.Split(path)
	appDir = strings.TrimSuffix(appDir, "/")
	appName := filepath.Base(appDir)
	a.mu.Lock()
	if _, ok := a.items[appName]; !ok {
		a.items[appName] = ActiveApp{Name: appName, Train: ""}
	} else {
		log.Error().Msgf("app [%s] already exists in activeApps", appName)
	}
	a.mu.Unlock()
	return nil
}

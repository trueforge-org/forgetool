package changelog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
)

type ActiveApp struct {
	Name  string
	Train string
}

type ActiveApps struct {
	items map[string]ActiveApp
	mu    *sync.RWMutex
}

func (a *ActiveApps) isActiveApp(appName string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	_, ok := a.items[appName]
	return ok
}

func (a *ActiveApps) getActiveAppsWalker(path string, entry os.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if entry.Name() != "Chart.yaml" {
		return nil
	}
	// path = apps/<train>/<app>/Chart.yaml or apps/<app>/Chart.yaml
	segLen := len(strings.Split(path, "/"))
	if segLen < 3 {
		return fmt.Errorf("path (%s) is not valid. expected at least <root>/<app>/Chart.yaml", path)
	}
	// appDir = apps/<train>/<app>/
	appDir, _ := filepath.Split(path)
	// appDir = apps/<train>/<app>
	appDir = strings.TrimSuffix(appDir, "/")
	// train = apps/<train>
	train := filepath.Dir(appDir)
	// train = <train>
	train = filepath.Base(train)
	// appName = <app>
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

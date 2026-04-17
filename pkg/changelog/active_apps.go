package changelog

import (
	"os"
	"sync"
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
	if entry.Name() != activeAppsManifestFile {
		return nil
	}
	return parseActiveAppFunc(a, path)
}

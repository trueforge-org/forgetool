package changelog

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestIsActiveAppMissing(t *testing.T) {
	ac := ActiveApps{items: make(map[string]ActiveApp), mu: &sync.RWMutex{}}
	if ac.isActiveApp("nonexistent") {
		t.Fatalf("expected nonexistent app to not be active")
	}
}

func TestIsActiveAppPresent(t *testing.T) {
	ac := ActiveApps{items: map[string]ActiveApp{
		"myapp": {Name: "myapp", Train: "stable"},
	}, mu: &sync.RWMutex{}}
	if !ac.isActiveApp("myapp") {
		t.Fatalf("expected myapp to be active")
	}
}

func TestGetActiveAppsWalkerSkipsNonChartYaml(t *testing.T) {
	ac := ActiveApps{items: make(map[string]ActiveApp), mu: &sync.RWMutex{}}

	td := t.TempDir()
	valuesPath := filepath.Join(td, "charts", "stable", "app", "values.yaml")
	writeFile(t, valuesPath, "foo: bar\n")

	entries, err := os.ReadDir(filepath.Dir(valuesPath))
	if err != nil {
		t.Fatalf("readdir failed: %v", err)
	}
	var valuesEntry os.DirEntry
	for _, e := range entries {
		if e.Name() == "values.yaml" {
			valuesEntry = e
			break
		}
	}

	err = ac.getActiveAppsWalker(valuesPath, valuesEntry, nil)
	if err != nil {
		t.Fatalf("walker should not fail for non-Chart.yaml: %v", err)
	}
	if len(ac.items) != 0 {
		t.Fatalf("expected no active apps for non-Chart.yaml file")
	}
}

func TestGetActiveAppsWalkerErrorPropagation(t *testing.T) {
	ac := ActiveApps{items: make(map[string]ActiveApp), mu: &sync.RWMutex{}}

	td := t.TempDir()
	appPath := filepath.Join(td, "charts", "stable", "app", "Chart.yaml")
	writeFile(t, appPath, "name: app\nversion: 1.0.0\n")

	entries, err := os.ReadDir(filepath.Dir(appPath))
	if err != nil {
		t.Fatalf("readdir failed: %v", err)
	}
	var appEntry os.DirEntry
	for _, e := range entries {
		if e.Name() == "Chart.yaml" {
			appEntry = e
			break
		}
	}

	testErr := fmt.Errorf("test error")
	err = ac.getActiveAppsWalker(appPath, appEntry, testErr)
	if err == nil {
		t.Fatalf("expected error to be propagated")
	}
	if err != testErr {
		t.Fatalf("expected test error, got %v", err)
	}
}

func TestGetActiveAppsWalkerDuplicateApp(t *testing.T) {
	ac := ActiveApps{items: make(map[string]ActiveApp), mu: &sync.RWMutex{}}

	td := t.TempDir()
	appPath := filepath.Join(td, "charts", "stable", "app", "Chart.yaml")
	writeFile(t, appPath, "name: app\nversion: 1.0.0\n")

	entries, err := os.ReadDir(filepath.Dir(appPath))
	if err != nil {
		t.Fatalf("readdir failed: %v", err)
	}
	var appEntry os.DirEntry
	for _, e := range entries {
		if e.Name() == "Chart.yaml" {
			appEntry = e
			break
		}
	}

	// First call should succeed
	if err := ac.getActiveAppsWalker(filepath.ToSlash(appPath), appEntry, nil); err != nil {
		t.Fatalf("first walker call failed: %v", err)
	}
	// Second call with same app name should not error but should log
	if err := ac.getActiveAppsWalker(filepath.ToSlash(appPath), appEntry, nil); err != nil {
		t.Fatalf("second walker call failed: %v", err)
	}
	if len(ac.items) != 1 {
		t.Fatalf("expected 1 active app, got %d", len(ac.items))
	}
}

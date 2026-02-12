package changelog

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestIsActiveChartMissing(t *testing.T) {
	ac := ActiveCharts{items: make(map[string]ActiveChart), mu: &sync.RWMutex{}}
	if ac.isActiveChart("nonexistent") {
		t.Fatalf("expected nonexistent chart to not be active")
	}
}

func TestIsActiveChartPresent(t *testing.T) {
	ac := ActiveCharts{items: map[string]ActiveChart{
		"mychart": {Name: "mychart", Train: "stable"},
	}, mu: &sync.RWMutex{}}
	if !ac.isActiveChart("mychart") {
		t.Fatalf("expected mychart to be active")
	}
}

func TestGetActiveChartsWalkerSkipsNonChartYaml(t *testing.T) {
	ac := ActiveCharts{items: make(map[string]ActiveChart), mu: &sync.RWMutex{}}

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

	err = ac.getActiveChartsWalker(valuesPath, valuesEntry, nil)
	if err != nil {
		t.Fatalf("walker should not fail for non-Chart.yaml: %v", err)
	}
	if len(ac.items) != 0 {
		t.Fatalf("expected no active charts for non-Chart.yaml file")
	}
}

func TestGetActiveChartsWalkerErrorPropagation(t *testing.T) {
	ac := ActiveCharts{items: make(map[string]ActiveChart), mu: &sync.RWMutex{}}

	td := t.TempDir()
	chartPath := filepath.Join(td, "charts", "stable", "app", "Chart.yaml")
	writeFile(t, chartPath, "name: app\nversion: 1.0.0\n")

	entries, err := os.ReadDir(filepath.Dir(chartPath))
	if err != nil {
		t.Fatalf("readdir failed: %v", err)
	}
	var chartEntry os.DirEntry
	for _, e := range entries {
		if e.Name() == "Chart.yaml" {
			chartEntry = e
			break
		}
	}

	testErr := fmt.Errorf("test error")
	err = ac.getActiveChartsWalker(chartPath, chartEntry, testErr)
	if err == nil {
		t.Fatalf("expected error to be propagated")
	}
	if err != testErr {
		t.Fatalf("expected test error, got %v", err)
	}
}

func TestGetActiveChartsWalkerDuplicateChart(t *testing.T) {
	ac := ActiveCharts{items: make(map[string]ActiveChart), mu: &sync.RWMutex{}}

	td := t.TempDir()
	chartPath := filepath.Join(td, "charts", "stable", "app", "Chart.yaml")
	writeFile(t, chartPath, "name: app\nversion: 1.0.0\n")

	entries, err := os.ReadDir(filepath.Dir(chartPath))
	if err != nil {
		t.Fatalf("readdir failed: %v", err)
	}
	var chartEntry os.DirEntry
	for _, e := range entries {
		if e.Name() == "Chart.yaml" {
			chartEntry = e
			break
		}
	}

	// First call should succeed
	if err := ac.getActiveChartsWalker(filepath.ToSlash(chartPath), chartEntry, nil); err != nil {
		t.Fatalf("first walker call failed: %v", err)
	}
	// Second call with same chart name should not error but should log
	if err := ac.getActiveChartsWalker(filepath.ToSlash(chartPath), chartEntry, nil); err != nil {
		t.Fatalf("second walker call failed: %v", err)
	}
	if len(ac.items) != 1 {
		t.Fatalf("expected 1 active chart, got %d", len(ac.items))
	}
}

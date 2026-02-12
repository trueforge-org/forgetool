package website

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

func TestGetChartData_AddsChart(t *testing.T) {
	// Create temp charts structure: <tmp>/trainA/mychart/Chart.yaml
	td := t.TempDir()
	chartDir := filepath.Join(td, "trainA", "mychart")
	if err := os.MkdirAll(chartDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	chartPath := filepath.Join(chartDir, "Chart.yaml")
	yaml := "name: mychart\nversion: 0.1.0\n"
	if err := os.WriteFile(chartPath, []byte(yaml), 0644); err != nil {
		t.Fatalf("write chart: %v", err)
	}

	// Obtain DirEntry for Chart.yaml
	entries, err := os.ReadDir(chartDir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}

	var entry os.DirEntry
	for _, e := range entries {
		if e.Name() == "Chart.yaml" {
			entry = e
			break
		}
	}
	if entry == nil {
		t.Fatal("chart entry not found")
	}

	opts := &ChartListOptions{}
	if err := opts.GetChartData(chartPath, entry, nil); err != nil {
		t.Fatalf("GetChartData failed: %v", err)
	}

	if opts.list == nil {
		t.Fatalf("list nil")
	}
	if opts.list.TotalCount != 1 {
		t.Fatalf("expected totalcount 1 got %d", opts.list.TotalCount)
	}
	if len(opts.list.Trains) != 1 {
		t.Fatalf("expected 1 train got %d", len(opts.list.Trains))
	}
	tr := opts.list.Trains[0]
	if tr.Name != "trainA" {
		t.Fatalf("expected train name trainA got %s", tr.Name)
	}
	if len(tr.Charts) != 1 || tr.Charts[0].Name != "mychart" {
		t.Fatalf("chart not recorded correctly")
	}
}

func TestGetChartData_TrainFilter(t *testing.T) {
	td := t.TempDir()
	chartDir := filepath.Join(td, "trainB", "chartb")
	if err := os.MkdirAll(chartDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	chartPath := filepath.Join(chartDir, "Chart.yaml")
	yaml := "name: chartb\nversion: 0.2.0\n"
	if err := os.WriteFile(chartPath, []byte(yaml), 0644); err != nil {
		t.Fatalf("write chart: %v", err)
	}
	entries, err := os.ReadDir(chartDir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	var entry os.DirEntry
	for _, e := range entries {
		if e.Name() == "Chart.yaml" {
			entry = e
			break
		}
	}
	if entry == nil {
		t.Fatal("chart entry not found")
	}

	opts := &ChartListOptions{TrainFilter: []string{"other"}}
	if err := opts.GetChartData(chartPath, entry, nil); err != nil {
		t.Fatalf("GetChartData failed: %v", err)
	}
	if opts.list == nil {
		// list remains nil because filtered out
		return
	}
	if opts.list.TotalCount != 0 || len(opts.list.Trains) != 0 {
		t.Fatalf("expected no charts added when filtered")
	}
}

func TestGetChartData_SkipExcludedDir(t *testing.T) {
	td := t.TempDir()
	// create an excluded directory name from helper.ExcludedDirs
	var exName string
	if len(helper.ExcludedDirs) > 0 {
		exName = helper.ExcludedDirs[0]
	} else {
		exName = "templates"
	}
	exPath := filepath.Join(td, exName)
	if err := os.MkdirAll(exPath, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	entries, err := os.ReadDir(td)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	var entry os.DirEntry
	for _, e := range entries {
		if e.Name() == exName {
			entry = e
			break
		}
	}
	if entry == nil {
		t.Fatal("excluded entry not found")
	}

	opts := &ChartListOptions{}
	err = opts.GetChartData(exPath, entry, nil)
	if err == nil {
		t.Fatalf("expected filepath.SkipDir error for excluded dir")
	}
}

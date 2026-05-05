package website

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/trueforge-org/forgetool/v4/pkg/helper"
)

type fakeDirEntry struct{ name string }

func (f fakeDirEntry) Name() string               { return f.name }
func (f fakeDirEntry) IsDir() bool                { return false }
func (f fakeDirEntry) Type() os.FileMode          { return 0 }
func (f fakeDirEntry) Info() (os.FileInfo, error) { return nil, nil }

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

func TestGetChartData_ReturnsInputError(t *testing.T) {
	td := t.TempDir()
	entries, err := os.ReadDir(td)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty temp dir")
	}

	chartPath := filepath.Join(td, "Chart.yaml")
	if err := os.WriteFile(chartPath, []byte("name: x\nversion: 0.1.0\n"), 0644); err != nil {
		t.Fatalf("write chart: %v", err)
	}
	fileEntries, err := os.ReadDir(td)
	if err != nil {
		t.Fatalf("readdir with file: %v", err)
	}
	entry := fileEntries[0]

	opts := &ChartListOptions{}
	wantErr := errors.New("walker error")
	if err := opts.GetChartData(chartPath, entry, wantErr); !errors.Is(err, wantErr) {
		t.Fatalf("expected input error to be returned, got: %v", err)
	}
}

func TestGetChartData_IgnoresNonChartYAML(t *testing.T) {
	td := t.TempDir()
	otherPath := filepath.Join(td, "values.yaml")
	if err := os.WriteFile(otherPath, []byte("foo: bar\n"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	entries, err := os.ReadDir(td)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one entry")
	}

	opts := &ChartListOptions{}
	if err := opts.GetChartData(otherPath, entries[0], nil); err != nil {
		t.Fatalf("GetChartData should ignore non-Chart.yaml files: %v", err)
	}
	if opts.list == nil {
		t.Fatalf("expected list to be initialized")
	}
	if opts.list.TotalCount != 0 {
		t.Fatalf("expected total count unchanged for non-Chart.yaml file")
	}
}

func TestGetChartData_AppendsToExistingTrain(t *testing.T) {
	td := t.TempDir()
	chartDirA := filepath.Join(td, "stable", "charta")
	chartDirB := filepath.Join(td, "stable", "chartb")
	if err := os.MkdirAll(chartDirA, 0755); err != nil {
		t.Fatalf("mkdir charta: %v", err)
	}
	if err := os.MkdirAll(chartDirB, 0755); err != nil {
		t.Fatalf("mkdir chartb: %v", err)
	}
	pathA := filepath.Join(chartDirA, "Chart.yaml")
	pathB := filepath.Join(chartDirB, "Chart.yaml")
	if err := os.WriteFile(pathA, []byte("name: charta\nversion: 0.1.0\n"), 0644); err != nil {
		t.Fatalf("write charta: %v", err)
	}
	if err := os.WriteFile(pathB, []byte("name: chartb\nversion: 0.1.1\n"), 0644); err != nil {
		t.Fatalf("write chartb: %v", err)
	}

	entriesA, _ := os.ReadDir(chartDirA)
	entriesB, _ := os.ReadDir(chartDirB)
	opts := &ChartListOptions{}
	if err := opts.GetChartData(pathA, entriesA[0], nil); err != nil {
		t.Fatalf("GetChartData charta failed: %v", err)
	}
	if err := opts.GetChartData(pathB, entriesB[0], nil); err != nil {
		t.Fatalf("GetChartData chartb failed: %v", err)
	}

	if opts.list.TotalCount != 2 {
		t.Fatalf("expected total count 2, got %d", opts.list.TotalCount)
	}
	if len(opts.list.Trains) != 1 {
		t.Fatalf("expected one train, got %d", len(opts.list.Trains))
	}
	if opts.list.Trains[0].Count != 2 || len(opts.list.Trains[0].Charts) != 2 {
		t.Fatalf("expected stable train to contain both charts, got %+v", opts.list.Trains[0])
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

func TestGetChartData_InvalidChartYAML(t *testing.T) {
	td := t.TempDir()
	chartDir := filepath.Join(td, "stable", "badchart")
	if err := os.MkdirAll(chartDir, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	chartPath := filepath.Join(chartDir, "Chart.yaml")
	if err := os.WriteFile(chartPath, []byte("name: [broken\n"), 0644); err != nil {
		t.Fatalf("write invalid chart failed: %v", err)
	}
	entries, err := os.ReadDir(chartDir)
	if err != nil {
		t.Fatalf("readdir failed: %v", err)
	}
	if err := os.Remove(chartPath); err != nil {
		t.Fatalf("remove chart file failed: %v", err)
	}

	opts := &ChartListOptions{}
	if err := opts.GetChartData(chartPath, entries[0], nil); err == nil {
		t.Fatalf("expected error when chart yaml cannot be loaded")
	}
}

func TestAddChartToTrain_AppendsExistingTrain(t *testing.T) {
	opts := &ChartListOptions{list: &ChartList{Trains: []Train{{Name: "stable", Count: 1, Charts: []Chart{{Name: "first", Train: "stable"}}}}}}
	opts.addChartToTrain(Chart{Name: "second", Train: "stable"})

	if opts.list.Trains[0].Count != 2 {
		t.Fatalf("expected train count to increment")
	}
	if len(opts.list.Trains[0].Charts) != 2 {
		t.Fatalf("expected chart to be appended")
	}
}

func TestAddChartToTrain_AppendsNewTrainAfterContinue(t *testing.T) {
	opts := &ChartListOptions{list: &ChartList{Trains: []Train{{Name: "incubator", Count: 1, Charts: []Chart{{Name: "x", Train: "incubator"}}}}}}
	opts.addChartToTrain(Chart{Name: "app", Train: "stable"})

	if len(opts.list.Trains) != 2 {
		t.Fatalf("expected second train to be appended")
	}
}

func TestGetChartData_LoadErrorWithFakeEntry(t *testing.T) {
	opts := &ChartListOptions{}
	err := opts.GetChartData(filepath.Join(t.TempDir(), "missing", "Chart.yaml"), fakeDirEntry{name: "Chart.yaml"}, nil)
	if err == nil {
		t.Fatalf("expected chart load error for missing Chart.yaml path")
	}
}

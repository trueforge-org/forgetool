package website

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/trueforge-org/forgetool/pkg/charts/chartFile"
	"github.com/trueforge-org/forgetool/pkg/helper"
)

var marshalChartList = json.Marshal

type ChartList struct {
	TotalCount int64   `json:"totalCount"`
	Trains     []Train `json:"trains"`
}

type Train struct {
	Name   string  `json:"name"`
	Count  int64   `json:"count"`
	Charts []Chart `json:"charts"`
}

type Chart struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Train       string `json:"train"`
	Link        string `json:"link"`
	Icon        string `json:"icon"`
	Version     string `json:"version"`
}

type ChartListOptions struct {
	OutputPath  string   // Path to put the chart list json file
	TrainFilter []string // Empty means all trains
	sync.Mutex
	list *ChartList
}

func (o *ChartListOptions) WriteChartList() error {
	if o.list == nil {
		return fmt.Errorf("chart list is nil")
	}

	data, err := marshalChartList(o.list)
	if err != nil {
		return err
	}

	return os.WriteFile(o.OutputPath, data, 0644)
}
func (o *ChartListOptions) GetChartData(path string, entry os.DirEntry, err error) error {
	if o.list == nil {
		o.list = &ChartList{}
	}

	if walkErr := o.validateChartEntry(entry, err); walkErr != nil {
		return walkErr
	}
	if entry.Name() != "Chart.yaml" {
		return nil
	}

	chart := chartFile.NewHelmChart()
	if err := chart.LoadFromFile(path); err != nil {
		return err
	}

	train := chartFile.GetTrain(path, chart)
	if len(o.TrainFilter) > 0 {
		if !slices.Contains(o.TrainFilter, train) {
			return nil
		}
	}

	o.Lock()
	defer o.Unlock()
	// Increment the total count
	o.list.TotalCount++
	webChart := buildWebChart(path, chart)
	o.addChartToTrain(webChart)

	return nil
}

func (o *ChartListOptions) validateChartEntry(entry os.DirEntry, err error) error {
	if err != nil {
		return err
	}

	if entry.IsDir() && slices.Contains(helper.ExcludedDirs, entry.Name()) {
		return filepath.SkipDir
	}

	return nil
}

func buildWebChart(path string, chart *chartFile.HelmChart) Chart {
	return Chart{
		Name:        chart.Metadata.Name,
		Description: chart.Metadata.Description,
		Icon:        chart.Metadata.Icon,
		Link:        chart.Metadata.Home,
		Version:     chart.Metadata.Version,
		Train:       chartFile.GetTrain(path, chart),
	}
}

func (o *ChartListOptions) addChartToTrain(webChart Chart) {
	for idx, train := range o.list.Trains {
		if train.Name != webChart.Train {
			continue
		}

		o.list.Trains[idx].Count++
		o.list.Trains[idx].Charts = append(o.list.Trains[idx].Charts, webChart)
		return
	}

	o.list.Trains = append(o.list.Trains, Train{
		Name:   webChart.Train,
		Count:  1,
		Charts: []Chart{webChart},
	})
}

package cmd

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/spf13/cobra"
	"github.com/trueforge-org/forgetool/pkg/charts/website"
	"github.com/trueforge-org/forgetool/pkg/helper"
)

var chartsGenChartsListLongHelp = strings.TrimSpace(`

`)

func defaultChartsGenChartListOptionsFactory() *website.ChartListOptions {
	return &website.ChartListOptions{
		OutputPath:  "./charts.json",
		TrainFilter: []string{},
	}
}

func defaultChartsGenChartListGetChartData(opts *website.ChartListOptions) fs.WalkDirFunc {
	return opts.GetChartData
}

func defaultChartsGenChartListWrite(opts *website.ChartListOptions) error {
	return opts.WriteChartList()
}

var (
	chartsGenChartListWalkCharts2    = helper.WalkCharts2
	chartsGenChartListOptionsFactory = defaultChartsGenChartListOptionsFactory
	chartsGenChartListGetChartData   = defaultChartsGenChartListGetChartData
	chartsGenChartListWrite          = defaultChartsGenChartListWrite
	chartsGenChartListRunner         = runChartsGenChartList
	chartsGenChartListOnError        = func(err error) { log.Fatal().Err(err).Msg("chart list generation failed") }
)

func runChartsGenChartList(args []string) error {
	opts := chartsGenChartListOptionsFactory()
	if err := chartsGenChartListWalkCharts2(args, chartsGenChartListGetChartData(opts), helper.AsyncMode); err != nil {
		return fmt.Errorf("failed to generate chart list json file: %w", err)
	}

	if err := chartsGenChartListWrite(opts); err != nil {
		return fmt.Errorf("failed to write chart list json file: %w", err)
	}

	return nil
}

var genChartListCmd = &cobra.Command{
	Use:     "genchartlist",
	Short:   "Generate chart list json file",
	Long:    chartsGenChartsListLongHelp,
	Example: "forgetool charts genchartlist <path to charts folder>",
	Run: func(cmd *cobra.Command, args []string) {
		if err := chartsGenChartListRunner(args); err != nil {
			chartsGenChartListOnError(err)
		}

	},
}

func init() {
	charts.AddCommand(genChartListCmd)
}

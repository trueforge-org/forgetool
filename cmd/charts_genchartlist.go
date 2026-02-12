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

var (
	chartsGenChartListWalkCharts2 = helper.WalkCharts2
	chartsGenChartListOptionsFactory = func() *website.ChartListOptions {
		return &website.ChartListOptions{
			OutputPath:  "./charts.json",
			TrainFilter: []string{},
		}
	}
	chartsGenChartListGetChartData = func(opts *website.ChartListOptions) fs.WalkDirFunc { return opts.GetChartData }
	chartsGenChartListWrite        = func(opts *website.ChartListOptions) error { return opts.WriteChartList() }
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
		if err := runChartsGenChartList(args); err != nil {
			log.Fatal().Err(err).Msg("chart list generation failed")
		}

	},
}

func init() {
	charts.AddCommand(genChartListCmd)
}

package cmd

import (
	"strings"

	"github.com/rs/zerolog/log"

	"slices"

	"github.com/spf13/cobra"
	"github.com/trueforge-org/forgetool/v4/pkg/charts/chartFile"
	"github.com/trueforge-org/forgetool/v4/pkg/helper"
)

var chartsGenMetaLongHelp = strings.TrimSpace(`

`)

var chartsGenMetaWalkCharts = helper.WalkCharts
var chartsGenMetaRunner = runChartsGenMeta
var chartsGenMetaOnError = func(err error) { log.Fatal().Err(err).Msg("failed to update Chart.yaml:") }

func parseChartsGenMetaArgs(args []string) (string, []string) {
	bump := ""
	if len(args) > 0 && slices.Contains([]string{"patch", "minor", "major"}, args[0]) {
		bump = args[0]
		args = args[1:]
	}

	return bump, args
}

func runChartsGenMeta(args []string) error {
	bump, paths := parseChartsGenMetaArgs(args)
	mode := helper.SyncMode

	return chartsGenMetaWalkCharts(paths, chartFile.UpdateChartFile, bump, mode)
}

var genMetaCmd = &cobra.Command{
	Use:     "genmeta",
	Short:   "Generate and update Chart.yaml metadata",
	Example: "forgetool charts genmeta",
	Long:    chartsGenMetaLongHelp,
	Run: func(cmd *cobra.Command, args []string) {
		err := chartsGenMetaRunner(args)
		if err != nil {
			chartsGenMetaOnError(err)
		}
	},
}

func init() {
	charts.AddCommand(genMetaCmd)
}

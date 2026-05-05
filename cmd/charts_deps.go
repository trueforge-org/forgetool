package cmd

import (
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/trueforge-org/forgetool/v4/pkg/charts/deps"
	"github.com/trueforge-org/forgetool/v4/pkg/helper"
)

var chartsDepsLongHelp = strings.TrimSpace(`

`)

var (
	chartsDepsLoadGPGKey = deps.LoadGPGKey
	chartsDepsWalkCharts = helper.WalkCharts
	chartsDepsRunner     = runChartsDeps
	chartsDepsOnError    = func(err error) { log.Fatal().Err(err).Msg("failed to update Chart.yaml") }
)

func runChartsDeps(args []string) error {
	if err := chartsDepsLoadGPGKey(); err != nil {
		return err
	}

	mode := helper.SyncMode
	if err := chartsDepsWalkCharts(args, deps.DownloadDeps, "", mode); err != nil {
		return err
	}

	return nil
}

var depsCmd = &cobra.Command{
	Use:     "deps",
	Short:   "Download, Update and Verify Helm dependencies",
	Long:    chartsDepsLongHelp,
	Example: "forgetool charts deps <chart> <chart> <chart>",
	Run: func(cmd *cobra.Command, args []string) {
		if err := chartsDepsRunner(args); err != nil {
			chartsDepsOnError(err)
		}
	},
}

func init() {
	charts.AddCommand(depsCmd)
}

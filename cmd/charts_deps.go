package cmd

import (
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/trueforge-org/forgetool/pkg/charts/deps"
	"github.com/trueforge-org/forgetool/pkg/helper"
)

var chartsDepsLongHelp = strings.TrimSpace(`

`)

var (
	chartsDepsLoadGPGKey = deps.LoadGPGKey
	chartsDepsWalkCharts = helper.WalkCharts
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
		if err := chartsDepsLoadGPGKey(); err != nil {
			log.Fatal().Err(err).Msg("failed to load gpg key")
			return
		}

		// Specify the mode (SyncMode or AsyncMode)
		mode := helper.SyncMode // Change to helper.SyncMode for synchronous processing
		if err := chartsDepsWalkCharts(args, deps.DownloadDeps, "", mode); err != nil {
			log.Fatal().Err(err).Msg("failed to update Chart.yaml")
		}
	},
}

func init() {
	charts.AddCommand(depsCmd)
}

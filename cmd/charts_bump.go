package cmd

import (
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/trueforge-org/forgetool/pkg/charts/version"
)

var chartsBumpLongHelp = strings.TrimSpace(`

`)

var chartsBumpVersion = version.Bump
var chartsBumpRunner = runChartsBump
var chartsBumpOnError = func(err error) { log.Fatal().Err(err).Msg("failed to bump version") }

func runChartsBump(args []string) error {
	return chartsBumpVersion(args[0], args[1])
}

var bumper = &cobra.Command{
	Use:     "bump",
	Short:   "generate a bumped image version",
	Long:    chartsBumpLongHelp,
	Example: "forgetool charts bump <version> <kind>",
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := chartsBumpRunner(args); err != nil {
			chartsBumpOnError(err)
		}
	},
}

func init() {
	charts.AddCommand(bumper)
}

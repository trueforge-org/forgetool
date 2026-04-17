package cmd

import (
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/trueforge-org/forgetool/pkg/version"
)

var chartsBumpLongHelp = strings.TrimSpace(`
Bump a semantic version.

Kinds:
	- major
	- minor
	- patch
	- pinDigest (alias for patch)
	- digest (alias for patch)
	- pin (alias for patch)
	- lockfile (alias for patch)
`)

var chartsBumpVersion = version.Bump
var chartsBumpRunner = runChartsBump
var chartsBumpOnError = func(err error) { log.Fatal().Err(err).Msg("failed to bump version") }

func runChartsBump(args []string) error {
	return chartsBumpVersion(args[0], args[1])
}

var bumper = &cobra.Command{
	Use:     "bump",
	Short:   "generate a bumped semantic version",
	Long:    chartsBumpLongHelp,
	Example: "forgetool charts bump 1.2.3 patch\nforgetool charts bump 1.2.3 pinDigest",
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

package cmd

import (
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/spf13/cobra"
	"github.com/trueforge-org/forgetool/v4/pkg/charts/image"
)

var chartsTagCleanLongHelp = strings.TrimSpace(`

`)

var chartsTagClean = image.Clean
var chartsTagCleanRunner = runChartsTagClean
var chartsTagCleanOnError = func(err error) { log.Fatal().Err(err).Msg("failed to clean tag") }

func runChartsTagClean(args []string) error {
	return chartsTagClean(args[0])
}

var tagCleaner = &cobra.Command{
	Use:     "tagcleaner",
	Short:   "Creates a clean version tag from a container digest",
	Long:    chartsTagCleanLongHelp,
	Example: "forgetool charts tagclean <tag>",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := chartsTagCleanRunner(args)
		if err != nil {
			chartsTagCleanOnError(err)
		}
	},
}

func init() {
	charts.AddCommand(tagCleaner)
}

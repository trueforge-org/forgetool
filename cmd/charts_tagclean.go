package cmd

import (
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/spf13/cobra"
	"github.com/trueforge-org/forgetool/pkg/charts/image"
)

var chartsTagCleanLongHelp = strings.TrimSpace(`

`)

var chartsTagClean = image.Clean

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
		err := runChartsTagClean(args)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to clean tag")
		}
	},
}

func init() {
	charts.AddCommand(tagCleaner)
}

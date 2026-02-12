package cmd

import (
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/trueforge-org/forgetool/pkg/sops"
)

var advPrecommitLongHelp = strings.TrimSpace(`

`)

var (
	precommitCheckFilesAndReportEncryption = sops.CheckFilesAndReportEncryption
	precommitExit                          = os.Exit
)

func runPrecommit() error {
	return precommitCheckFilesAndReportEncryption(true, true)
}

var precommit = &cobra.Command{
	Use:     "precommit",
	Short:   "Runs the PreCommit encryption check",
	Example: "forgetool adv precommit",
	Long:    advPrecommitLongHelp,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runPrecommit(); err != nil {
			log.Info().Msgf("Error checking files: %v\n", err)
			precommitExit(1)
		}
	},
}

func init() {
	adv.AddCommand(precommit)
}

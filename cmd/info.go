package cmd

import (
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/trueforge-org/forgetool/pkg/info"
)

var description = strings.TrimSpace(`Forgetool is a toolkit to help maintain trueforge projects.
`)

var infoLongHelp = strings.TrimSpace(description + `
Workflow:
  Create talconfig.yaml file defining your nodes information like so:

 Available commands
  > forgetool chart
  > forgetool container

`)

var infoCmd = &cobra.Command{
	Use:     "info",
	Short:   "Prints information about the forgetool binary",
	Long:    infoLongHelp,
	Example: "forgetool info",
	Run: func(cmd *cobra.Command, args []string) {
		log.Info().Msg(description)
		info.NewInfo().Print()
	},
}

func init() {
	RootCmd.AddCommand(infoCmd)
}

package cmd

import (
	"strings"

	"github.com/spf13/cobra"
)

var containerLongHelp = strings.TrimSpace(`
These are all commands that can be used to test container images

`)

var containerCmd = &cobra.Command{
	Use:           "container",
	Short:         "Commands for container validation",
	Example:       "forgetool container test",
	Long:          containerLongHelp,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	RootCmd.AddCommand(containerCmd)
}

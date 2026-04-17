package cmd

import (
	"strings"

	"github.com/spf13/cobra"
)

var containersLongHelp = strings.TrimSpace(`
These are all commands that can be used to test container images
`)

var containersCmd = &cobra.Command{
	Use:           "containers",
	Short:         "Commands for container validation",
	Example:       "forgetool containers test",
	Long:          containersLongHelp,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	RootCmd.AddCommand(containersCmd)
}

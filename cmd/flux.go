package cmd

import (
	"strings"

	"github.com/spf13/cobra"
)

//nolint:unused
var fluxLongHelp = strings.TrimSpace(`
These are all commands that can be used to maintain FluxCD

`)

var fluxCmd = &cobra.Command{
	Use:           "flux",
	Short:         "Commands for handling FluxCD",
	Example:       "forgetool flux bootstrap",
	Long:          advLongHelp,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	RootCmd.AddCommand(fluxCmd)
}

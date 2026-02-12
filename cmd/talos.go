package cmd

import (
	"strings"

	"github.com/spf13/cobra"
)

//nolint:unused
var talosLongHelp = strings.TrimSpace(`
These are all commands that can be used to maintain Talos OS

`)

var talosCmd = &cobra.Command{
	Use:           "talos",
	Short:         "Commands for handling Talos OS",
	Example:       "forgetool talos apply",
	Long:          advLongHelp,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	RootCmd.AddCommand(talosCmd)
}

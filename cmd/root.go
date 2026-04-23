package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var thisversion string

var RootCmd = &cobra.Command{
	Use:           "forgetool",
	Short:         "A tool to help maintaining charts and containers",
	Long:          infoLongHelp,
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       thisversion,
}

func Execute() error {
	cmd, err := RootCmd.ExecuteContextC(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())

		errorString := err.Error()
		if strings.Contains(errorString, "arg(s)") || strings.Contains(errorString, "flag") || strings.Contains(errorString, "command") {
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, cmd.UsageString())
		}
	}

	return err
}

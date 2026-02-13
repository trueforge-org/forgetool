package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/siderolabs/talos/cmd/talosctl/cmd/mgmt"
	_ "github.com/siderolabs/talos/cmd/talosctl/cmd/mgmt/cluster/create" // import to get the command registered via the init() function.
	"github.com/siderolabs/talos/cmd/talosctl/cmd/talos"
	talosctlpkg "github.com/trueforge-org/forgetool/pkg/talosctl"
)

var talosctlLongHelp = strings.TrimSpace(`
These are all talosctlanced commands that should generally not be needed

`)

var internalTalosctlMu sync.Mutex
var internalTalosctl = buildInternalTalosctlCommand()

var talosctl = &cobra.Command{
	Use:           "talosctl",
	Short:         "A CLI for out-of-band management of Kubernetes nodes created by Talos",
	Example:       "forgetool talosctl <bootstrap/health/precommit>",
	Long:          talosctlLongHelp,
	SilenceUsage:  true,
	SilenceErrors: true,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		out, err := runTalosctlArgs(args, false)
		if out != "" {
			fmt.Fprint(cmd.OutOrStdout(), out)
		}
		return err
	},
}

func buildInternalTalosctlCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "talosctl",
		Short:         "A CLI for out-of-band management of Kubernetes nodes created by Talos",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	for _, command := range mgmt.Commands {
		root.AddCommand(command)
	}

	for _, command := range talos.Commands {
		root.AddCommand(command)
	}

	return root
}

func runTalosctlArgs(args []string, silent bool) (string, error) {
	internalTalosctlMu.Lock()
	defer internalTalosctlMu.Unlock()

	var stdoutBuf, stderrBuf bytes.Buffer
	stdoutWriter := io.Writer(&stdoutBuf)
	stderrWriter := io.Writer(&stderrBuf)

	if !silent {
		stdoutWriter = io.MultiWriter(&stdoutBuf, os.Stdout)
		stderrWriter = io.MultiWriter(&stderrBuf, os.Stderr)
	}

	internalTalosctl.SetOut(stdoutWriter)
	internalTalosctl.SetErr(stderrWriter)
	internalTalosctl.SetArgs(args)
	err := internalTalosctl.Execute()
	internalTalosctl.SetArgs(nil)

	return stdoutBuf.String() + stderrBuf.String(), err
}

func init() {
	talosctlpkg.SetExecutor(runTalosctlArgs)

	RootCmd.AddCommand(talosctl)
}

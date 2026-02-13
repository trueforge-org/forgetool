package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/siderolabs/talos/cmd/talosctl/cmd/mgmt"
	"github.com/siderolabs/talos/cmd/talosctl/cmd/mgmt/cluster"
	_ "github.com/siderolabs/talos/cmd/talosctl/cmd/mgmt/cluster/create" // import to get the command registered via the init() function.
	"github.com/siderolabs/talos/cmd/talosctl/cmd/talos"
	talosctlpkg "github.com/trueforge-org/forgetool/pkg/talosctl"
)

var talosctlLongHelp = strings.TrimSpace(`
These are all talosctlanced commands that should generally not be needed

`)

var talosctl = &cobra.Command{
	Use:           "talosctl",
	Short:         "A CLI for out-of-band management of Kubernetes nodes created by Talos",
	Example:       "forgetool talosctl <bootstrap/health/precommit>",
	Long:          talosctlLongHelp,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func runTalosctlArgs(args []string, silent bool) (string, error) {
	var stdoutBuf, stderrBuf bytes.Buffer
	stdoutWriter := io.Writer(&stdoutBuf)
	stderrWriter := io.Writer(&stderrBuf)

	if !silent {
		stdoutWriter = io.MultiWriter(&stdoutBuf, os.Stdout)
		stderrWriter = io.MultiWriter(&stderrBuf, os.Stderr)
	}

	talosctl.SetOut(stdoutWriter)
	talosctl.SetErr(stderrWriter)
	talosctl.SetArgs(args)
	err := talosctl.Execute()
	talosctl.SetArgs(nil)

	return stdoutBuf.String() + stderrBuf.String(), err
}

func init() {
	talosctlpkg.SetExecutor(runTalosctlArgs)

	RootCmd.AddCommand(talosctl)
	const (
		talosGroup   = "talos"
		mgmtGroup    = "mgmt"
		clusterGroup = "cluster"
	)

	talosctl.AddGroup(&cobra.Group{ID: talosGroup, Title: "Manage running Talos clusters:"})
	talosctl.AddGroup(&cobra.Group{ID: mgmtGroup, Title: "Commands to generate and manage machine configuration offline:"})
	talosctl.AddGroup(&cobra.Group{ID: clusterGroup, Title: "Local Talos cluster commands:"})

	for _, cmd := range mgmt.Commands {
		cmd.GroupID = mgmtGroup
		if cmd == cluster.Cmd {
			cmd.GroupID = clusterGroup
		}

		talosctl.AddCommand(cmd)
	}

	for _, cmd := range talos.Commands {
		cmd.GroupID = talosGroup
		talosctl.AddCommand(cmd)
	}
}

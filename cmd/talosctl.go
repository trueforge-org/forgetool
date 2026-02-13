package cmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/siderolabs/talos/cmd/talosctl/cmd/mgmt"
	"github.com/siderolabs/talos/cmd/talosctl/cmd/mgmt/cluster"
	_ "github.com/siderolabs/talos/cmd/talosctl/cmd/mgmt/cluster/create" // import to get the command registered via the init() function.
	"github.com/siderolabs/talos/cmd/talosctl/cmd/talos"
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

func init() {
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

package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/trueforge-org/forgetool/pkg/gencmd"
	"github.com/trueforge-org/forgetool/pkg/helper"
	"github.com/trueforge-org/forgetool/pkg/initfiles"
	"github.com/trueforge-org/forgetool/pkg/talassist"
)

var (
	talosBootstrapGetYesOrNo    = helper.GetYesOrNo
	talosBootstrapLoadTalEnv    = initfiles.LoadTalEnv
	talosBootstrapLoadTalConfig = talassist.LoadTalConfig
	talosBootstrapRunBootstrap  = gencmd.RunBootstrap
	talosBootstrapGenPlain      = gencmd.GenPlain
	talosBootstrapExecCmd       = gencmd.ExecCmd
)

var advBootstrapLongHelp = strings.TrimSpace(`

`)

var bootstrap = &cobra.Command{
	Use:     "bootstrap",
	Short:   "bootstrap first Talos Node",
	Example: "forgetool talos bootstrap",
	Long:    advBootstrapLongHelp,
	Run:     bootstrapfunc,
}

func bootstrapfunc(cmd *cobra.Command, args []string) {
	if talosBootstrapGetYesOrNo("Do you want to also run the complete ForgeTool Bootstrap, besides just talos? (yes/no) [y/n]: ", false) {
		talosBootstrapLoadTalEnv(false)
		talosBootstrapLoadTalConfig()
		talosBootstrapRunBootstrap(args)
	} else {
		bootstrapcmds := talosBootstrapGenPlain("bootstrap", talassist.TalConfig.Nodes[0].IPAddress, []string{})
		talosBootstrapExecCmd(bootstrapcmds[0])
	}
}

func init() {
	talosCmd.AddCommand(bootstrap)
}

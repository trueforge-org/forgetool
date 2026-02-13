package cmd

import (
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/trueforge-org/forgetool/pkg/gencmd"
	"github.com/trueforge-org/forgetool/pkg/initfiles"
	"github.com/trueforge-org/forgetool/pkg/sops"
	"github.com/trueforge-org/forgetool/pkg/talassist"
)

var (
	talosResetDecryptFiles  = sops.DecryptFiles
	talosResetLoadTalEnv    = initfiles.LoadTalEnv
	talosResetLoadTalConfig = talassist.LoadTalConfig
	talosResetGenPlain      = gencmd.GenPlain
	talosResetExecCmds      = gencmd.ExecCmds
)

func runTalosReset(args []string) {
	node, extraArgs := parseTalosApplyArgs(args)

	if err := talosResetDecryptFiles(); err != nil {
		log.Info().Msgf("Error decrypting files: %v\n", err)
	}
	_ = talosResetLoadTalEnv(false)
	talosResetLoadTalConfig()

	log.Info().Msg("Running Cluster node Reset")

	taloscmds := talosResetGenPlain("reset", node, extraArgs)
	_ = talosResetExecCmds(taloscmds, true)
}

var advResetLongHelp = strings.TrimSpace(`

`)

var reset = &cobra.Command{
	Use:     "reset",
	Short:   "Reset Talos Nodes and Kubernetes",
	Example: "forgetool talos reset <NodeIP>",
	Long:    advResetLongHelp,
	Run: func(cmd *cobra.Command, args []string) {
		runTalosReset(args)
	},
}

func init() {
	talosCmd.AddCommand(reset)
}

package cmd

import (
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/trueforge-org/forgetool/pkg/gencmd"
	"github.com/trueforge-org/forgetool/pkg/helper"
	"github.com/trueforge-org/forgetool/pkg/initfiles"
	"github.com/trueforge-org/forgetool/pkg/sops"
	"github.com/trueforge-org/forgetool/pkg/talassist"
)

var (
	talosUpgradeDecryptFiles   = sops.DecryptFiles
	talosUpgradeLoadTalEnv     = initfiles.LoadTalEnv
	talosUpgradeLoadTalConfig  = talassist.LoadTalConfig
	talosUpgradeGenUpgrade     = gencmd.GenUpgrade
	talosUpgradeExecCmds       = gencmd.ExecCmds
	talosUpgradeGenKubeUpgrade = gencmd.GenKubeUpgrade
	talosUpgradeExecCmd        = gencmd.ExecCmd
	talosUpgradeGenPlain       = gencmd.GenPlain
)

func runTalosUpgrade(args []string) {
	node, extraArgs := parseTalosApplyArgs(args)

	if err := talosUpgradeDecryptFiles(); err != nil {
		log.Info().Msgf("Error decrypting files: %v\n", err)
	}
	_ = talosUpgradeLoadTalEnv(false)
	talosUpgradeLoadTalConfig()

	log.Info().Msg("Running Cluster Upgrade")

	taloscmds := talosUpgradeGenUpgrade(node, extraArgs)
	_ = talosUpgradeExecCmds(taloscmds, true)

	log.Info().Msg("Running Kubernetes Upgrade")
	kubeUpgradeCmd := talosUpgradeGenKubeUpgrade(helper.TalEnv["VIP_IP"])
	talosUpgradeExecCmd(kubeUpgradeCmd)

	log.Info().Msg("(re)Loading KubeConfig)")
	kubeconfigcmds := talosUpgradeGenPlain("health", helper.TalEnv["VIP_IP"], []string{"-f"})
	talosUpgradeExecCmd(kubeconfigcmds[0])
}

var upgradeLongHelp = strings.TrimSpace(`
The "upgrade" command updates Talos to the latest version specified in talconfig.yaml for all nodes.
It also applies any changed "extentions" and/or "overlays" specified there.

On top of this, after upgrading Talos on all nodes, it also executes kubernetes-upgrades for the whole cluster as well.

`)

var upgrade = &cobra.Command{
	Use:     "upgrade",
	Short:   "Upgrade Talos Nodes and Kubernetes",
	Example: "forgetool talos upgrade <NodeIP>",
	Long:    upgradeLongHelp,
	Run: func(cmd *cobra.Command, args []string) {
		runTalosUpgrade(args)
	},
}

func init() {
	talosCmd.AddCommand(upgrade)
}

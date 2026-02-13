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
	talosKubeconfigDecryptFiles  = sops.DecryptFiles
	talosKubeconfigLoadTalEnv    = initfiles.LoadTalEnv
	talosKubeconfigLoadTalConfig = talassist.LoadTalConfig
	talosKubeconfigGenPlain      = gencmd.GenPlain
	talosKubeconfigExecCmds      = gencmd.ExecCmds
)

func runTalosKubeconfig(args []string) {
	node, extraArgs := parseTalosApplyArgs(args)

	if err := talosKubeconfigDecryptFiles(); err != nil {
		log.Info().Msgf("Error decrypting files: %v\n", err)
	}
	_ = talosKubeconfigLoadTalEnv(false)
	talosKubeconfigLoadTalConfig()
	log.Info().Msg("Running Cluster kubeconfig")

	taloscmds := talosKubeconfigGenPlain("kubeconfig", node, extraArgs)
	_ = talosKubeconfigExecCmds(taloscmds, true)
}

//nolint:unused
var advKubeconfigLongHelp = strings.TrimSpace(`

`)

var kubeconfig = &cobra.Command{
	Use:     "kubeconfig",
	Short:   "kubeconfig for Talos Cluster",
	Example: "forgetool talos kubeconfig <NodeIP>",
	Long:    advResetLongHelp,
	Run: func(cmd *cobra.Command, args []string) {
		runTalosKubeconfig(args)
	},
}

func init() {
	talosCmd.AddCommand(kubeconfig)
}

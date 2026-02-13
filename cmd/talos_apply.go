package cmd

import (
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/trueforge-org/forgetool/pkg/gencmd"
	"github.com/trueforge-org/forgetool/pkg/helper"
	"github.com/trueforge-org/forgetool/pkg/initfiles"
	"github.com/trueforge-org/forgetool/pkg/nodestatus"
	"github.com/trueforge-org/forgetool/pkg/sops"
	"github.com/trueforge-org/forgetool/pkg/talassist"
)

var (
	talosApplyGenApply = gencmd.GenApply
	talosApplyExecCmds = gencmd.ExecCmds
	talosApplyGenPlain = gencmd.GenPlain
	talosApplyExecCmd  = gencmd.ExecCmd

	talosApplyDecryptFiles       = sops.DecryptFiles
	talosApplyLoadTalEnv         = initfiles.LoadTalEnv
	talosApplyLoadTalConfig      = talassist.LoadTalConfig
	talosApplyWaitForHealth      = nodestatus.WaitForHealth
	talosApplyCheckNeedBootstrap = nodestatus.CheckNeedBootstrap
	talosApplyGetYesOrNo         = helper.GetYesOrNo
	talosApplyRunBootstrap       = gencmd.RunBootstrap
	talosApplyRunApply           = RunApply
)

func parseTalosApplyArgs(args []string) (string, []string) {
	var extraArgs []string
	node := ""

	if len(args) > 1 {
		extraArgs = args[1:]
	}
	if len(args) >= 1 {
		node = args[0]
		if args[0] == "all" {
			node = ""
		}
	}

	return node, extraArgs
}

func runTalosApply(args []string) {
	node, extraArgs := parseTalosApplyArgs(args)

	if err := talosApplyDecryptFiles(); err != nil {
		log.Info().Msgf("Error decrypting files: %v\n", err)
	}

	_ = talosApplyLoadTalEnv(false)
	talosApplyLoadTalConfig()
	bootstrapNode := talassist.TalConfig.Nodes[0].IPAddress

	log.Info().Msgf("Checking if first node   is ready to recieve anything... %s", bootstrapNode)
	status, err := talosApplyWaitForHealth(bootstrapNode, []string{"running", "maintenance"})
	if err != nil {
		return
	}

	if status == "maintenance" {
		log.Info().Msg("Detected maintenance mode...")
		bootstrapNeeded, bootstrapErr := talosApplyCheckNeedBootstrap(bootstrapNode)
		if bootstrapErr != nil {
			return
		}

		if bootstrapNeeded {
			log.Info().Msg("First node requires bootstrapping before it can be used.")
			shouldBootstrap := true
			if !helper.NonInteractive {
				shouldBootstrap = talosApplyGetYesOrNo("Do you want to bootstrap now? (yes/no) [y/n]: ", true)
			} else {
				log.Info().Msg("Non-interactive mode detected, proceeding with bootstrap...")
			}

			if shouldBootstrap {
				talosApplyRunBootstrap(extraArgs)
				if talosApplyGetYesOrNo("Do you want to apply config to all remaining clusternodes as well? (yes/no) [y/n]: ", true) {
					talosApplyRunApply(false, "", extraArgs)
				}
			} else {
				log.Info().Msg("Exiting bootstrap, as apply is not possible...")
			}

			return
		}

		log.Info().Msg("First node does not require bootstrapping.")
		log.Info().Msg("Assuming apply is requested... continuing with Apply...")
		talosApplyRunApply(true, node, extraArgs)
		return
	}

	if status == "running" {
		log.Info().Msg("Apply: running first controlnode detected, continuing...")
		talosApplyRunApply(true, node, extraArgs)
	}
}

var applyLongHelp = strings.TrimSpace(`
The "apply" command applies your Talos System configuration to each node in the cluster, existing or new It also runs automated checking of your config file and health checks between each node it has processed, to ensure you don't accidentally take down your whole cluster.

## Bootstrapping
If the cluster has not been bootstrapped yet, Apply will automatically detect this and ask if you want to bootstrap the cluster

Bootstrapping will apply your config to the first (top) controlplane node in your "talconfig.yaml", it then "bootstraps" hence creating a new cluster with said node.

After this is done, we apply a number of helm-charts and manifests by default such as:

- Metallb
- Metallb-Config
- Cilium (CNI)
- Certificate-Approver
- Spegel
- Kubernetes-Dashboard

### Bootstrapping FluxCD

During Bootstrapping, if a "GITHUB_REPOSITORY" is set in "clusterenv.yaml", you will be asked if you also want to bootstrap FluxCD, checkout the getting-started guide for more info

## About Bootstrapping

While we load a lot of helm-charts during bootstrap, we will *never* manage them for you.
You're responsible for maintaining and configuring your cluster after bootstrapping.

Apply and *all other* commands, are just for maintaining Talos itself.
Not any contained helm-charts

`)

var apply = &cobra.Command{
	Use:     "apply",
	Short:   "apply",
	Aliases: []string{"apply-config"},
	Example: "forgetool apply <NodeIP>",
	Long:    applyLongHelp,
	Run: func(cmd *cobra.Command, args []string) {
		runTalosApply(args)
	},
}

func RunApply(kubeconfig bool, node string, extraArgs []string) {
	taloscmds := talosApplyGenApply(node, extraArgs)
	talosApplyExecCmds(taloscmds, true)

	if kubeconfig {
		kubeconfigcmds := talosApplyGenPlain("kubeconfig", helper.TalEnv["VIP_IP"], []string{"-f"})
		talosApplyExecCmd(kubeconfigcmds[0])
	}

	//if helper.GetYesOrNo("Do you want to (re)load ssh, Sops and ClusterEnv onto the cluster? (yes/no) [y/n]: ") {
	//
	//}
}

func init() {
	talosCmd.AddCommand(apply)
}

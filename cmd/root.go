package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/siderolabs/talos/cmd/talosctl/cmd/common"
	"github.com/spf13/cobra"
	"github.com/trueforge-org/forgetool/pkg/helper"
)

var thisversion string

var RootCmd = &cobra.Command{
	Use:           "forgetool",
	Short:         "A tool to help with creating Talos cluster",
	Long:          infoLongHelp,
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       thisversion,
}

func init() {
	// Define the --cluster flag
	RootCmd.PersistentFlags().StringVar(&helper.ClusterName, "cluster", "main", "Cluster name")
}

func Execute() error {

	// Execute adds all child commands to the root command and sets flags appropriately.
	// This is called by main.main(). It only needs to happen once to the RootCmd.
	cmd, err := RootCmd.ExecuteContextC(context.Background())
	if err != nil && !common.SuppressErrors {
		fmt.Fprintln(os.Stderr, err.Error())

		errorString := err.Error()
		// TODO: this is a nightmare, but arg-flag related validation returns simple `fmt.Errorf`, no way to distinguish
		//       these errors
		if strings.Contains(errorString, "arg(s)") || strings.Contains(errorString, "flag") || strings.Contains(errorString, "command") {
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, cmd.UsageString())
		}
	}

	// Parse only the persistent flags (like --cluster) before executing any command
	RootCmd.PersistentFlags().Parse(os.Args[1:])

	// You can now access the helper.ClusterName variable
	if helper.ClusterName != "" {
		log.Info().Msgf("Cluster name: %s\n", helper.ClusterName)
		helper.ClusterPath = filepath.Join("./clusters", helper.ClusterName)
		helper.ClusterEnvFile = filepath.Join(helper.ClusterPath, "/clusterenv.yaml")
		helper.TalConfigFile = filepath.Join(helper.ClusterPath, "/talos", "talconfig.yaml")
		helper.TalosPath = filepath.Join(helper.ClusterPath, "/talos")
		helper.KubernetesPath = filepath.Join(helper.ClusterPath, "/kubernetes")
		helper.TalosGenerated = filepath.Join(helper.TalosPath, "/generated")
		helper.TalosConfigFile = filepath.Join(helper.TalosGenerated, "talosconfig")
		helper.TalSecretFile = filepath.Join(helper.TalosGenerated, "talsecret.yaml")
	}

	// Execute the root command and all subcommands
	return RootCmd.Execute()
}

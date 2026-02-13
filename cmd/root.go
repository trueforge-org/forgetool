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

func applyClusterContext(clusterName string) {
	if clusterName == "" {
		return
	}

	helper.ClusterName = clusterName
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

func clusterNameFromArgs(args []string, fallback string) string {
	clusterName := fallback
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--cluster=") {
			clusterName = strings.TrimPrefix(arg, "--cluster=")
			continue
		}

		if arg == "--cluster" {
			if i+1 < len(args) {
				clusterName = args[i+1]
				i++
			}
		}
	}

	return clusterName
}

func Execute() error {
	applyClusterContext(clusterNameFromArgs(os.Args[1:], helper.ClusterName))

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

	return err
}

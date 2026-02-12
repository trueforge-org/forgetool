package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/trueforge-org/forgetool/pkg/helper"
	"github.com/trueforge-org/forgetool/pkg/initfiles"
	"github.com/trueforge-org/forgetool/pkg/kubectlcmds"
	"github.com/trueforge-org/forgetool/pkg/talassist"
)

var advTestCmdlongHelp = strings.TrimSpace(`
This command is mostly just for development usage and should NEVER be used by end-users.
`)

var (
	advTestLoadTalEnv    = func() { initfiles.LoadTalEnv(false) }
	advTestLoadTalConfig = talassist.LoadTalConfig
	advTestKubectlApply  = kubectlcmds.KubectlApply
	advTestExit          = os.Exit
)

func advTestManifestPaths() []string {
	return []string{
		filepath.Join(helper.KubernetesPath, "flux-system", "flux", "sopssecret.secret.yaml"),
		filepath.Join(helper.KubernetesPath, "flux-system", "flux", "deploykey.secret.yaml"),
		filepath.Join(helper.KubernetesPath, "flux-system", "flux", "clustersettings.secret.yaml"),
	}
}

func runAdvTestCommand(ctx context.Context, manifestPaths []string) error {
	for _, filePath := range manifestPaths {
		log.Info().Msgf("Bootstrap: Loading Manifest: %v", filePath)
		if err := advTestKubectlApply(ctx, filePath); err != nil {
			return fmt.Errorf("error applying manifest for %s: %w", filepath.Base(filePath), err)
		}
	}

	return nil
}

var testcmd = &cobra.Command{
	Use:   "test",
	Short: "tests specific code for developer usages",
	Long:  advTestCmdlongHelp,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		advTestLoadTalEnv()
		advTestLoadTalConfig()
		// err := fluxhandler.ProcessJSONFiles("./testdata/truenas_exports")
		// if err != nil {
		//  log.Info().Msg("Error:", err)
		// }
		if err := runAdvTestCommand(ctx, advTestManifestPaths()); err != nil {
			log.Info().Msgf("%v\n", err)
			advTestExit(1)
		}
	},
}

func init() {
	adv.AddCommand(testcmd)
}

package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/trueforge-org/forgetool/pkg/initfiles"
	"github.com/trueforge-org/forgetool/pkg/sops"
)

var (
	clusterInitDecryptFiles = sops.DecryptFiles
	clusterInitInitFiles    = initfiles.InitFiles
)

func runClusterInit() {
	_ = clusterInitDecryptFiles()
	_ = clusterInitInitFiles()
}

var initLongHelp = strings.TrimSpace(`
ForgeTool requires a specific directory layout to ensure smooth operators and standardised environments.

To ensure smooth deployment, the init function can pre-generate all required files in the right places.
Afterwards, you can edit talconfig.yaml and clusterenv.yaml to reflect your personal settings.

When done, please run forgetool cluster genconfig to generate all configurations based on your personal settings.
`)

var initFiles = &cobra.Command{
	Use:     "init",
	Short:   "generate Basic cluster file-and-folder structure in current folder",
	Long:    initLongHelp,
	Example: "forgetool cluster init",
	Run: func(cmd *cobra.Command, args []string) {
		runClusterInit()
	},
}

func init() {
	clusterCmd.AddCommand(initFiles)
}

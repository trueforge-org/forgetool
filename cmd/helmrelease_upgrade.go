package cmd

import (
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/trueforge-org/forgetool/pkg/fluxhandler"
	"github.com/trueforge-org/forgetool/pkg/initfiles"
)

var (
	helmreleaseUpgradeLoadTalEnv = initfiles.LoadTalEnv
	helmreleaseUpgradeLoadRepos  = fluxhandler.LoadAllHelmRepos
	helmreleaseUpgradeCharts     = fluxhandler.UpgradeCharts
)

func runHelmreleaseUpgrade(argPath string) {
	helmreleaseUpgradeLoadTalEnv(false)

	dir := resolveHelmReleaseDir(argPath)
	helmRepoPath := filepath.Join("./repositories", "helm")
	helmRepos, _ := helmreleaseUpgradeLoadRepos(helmRepoPath)
	intermediateCharts := []fluxhandler.HelmChart{{ChartPath: dir, Retry: false, Wait: true}}

	helmreleaseUpgradeCharts(intermediateCharts, helmRepos, false)
}

var hrUpgradeLongHelp = strings.TrimSpace(`

`)

var hrupgrade = &cobra.Command{
	Use:     "upgrade",
	Short:   "run helm-upgrade using a helm-release file without flux",
	Aliases: []string{"update", "edit"},
	Example: "forgetool helmrelease upgrade",
	Long:    hrUpgradeLongHelp,
	Run: func(cmd *cobra.Command, args []string) {
		runHelmreleaseUpgrade(args[0])
	},
}

func init() {
	helmrelease.AddCommand(hrupgrade)
}

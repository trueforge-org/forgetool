package cmd

import (
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/trueforge-org/forgetool/pkg/fluxhandler"
	"github.com/trueforge-org/forgetool/pkg/initfiles"
)

var (
	helmreleaseInstallLoadTalEnv = initfiles.LoadTalEnv
	helmreleaseInstallLoadRepos  = fluxhandler.LoadAllHelmRepos
	helmreleaseInstallCharts     = fluxhandler.InstallCharts
)

func resolveHelmReleaseDir(inputPath string) string {
	if filename := filepath.Base(inputPath); filename != "" && filename != "." && filename != "/" {
		return filepath.Dir(inputPath)
	}
	return inputPath
}

func runHelmreleaseInstall(argPath string) {
	helmreleaseInstallLoadTalEnv(false)

	dir := resolveHelmReleaseDir(argPath)
	helmRepoPath := filepath.Join("./repositories", "helm")
	helmRepos, _ := helmreleaseInstallLoadRepos(helmRepoPath)
	intermediateCharts := []fluxhandler.HelmChart{{ChartPath: dir, Retry: false, Wait: true}}

	helmreleaseInstallCharts(intermediateCharts, helmRepos, false)
}

var hrInstalLongHelp = strings.TrimSpace(`

`)

var hrinstall = &cobra.Command{
	Use:     "install",
	Short:   "install a helm-release file without flux, helm-release file needs to be called helm-release.yaml",
	Example: "forgetool helmrelease install",
	Long:    hrInstalLongHelp,
	Run: func(cmd *cobra.Command, args []string) {
		runHelmreleaseInstall(args[0])
	},
}

func init() {
	helmrelease.AddCommand(hrinstall)
}

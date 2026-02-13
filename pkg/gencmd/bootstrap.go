package gencmd

import (
	"context"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"

	"github.com/trueforge-org/forgetool/pkg/fluxhandler"
	"github.com/trueforge-org/forgetool/pkg/helper"
	"github.com/trueforge-org/forgetool/pkg/kubectlcmds"
	"github.com/trueforge-org/forgetool/pkg/nodestatus"
	"github.com/trueforge-org/forgetool/pkg/sops"
	"github.com/trueforge-org/forgetool/pkg/talassist"
)

var HelmRepos map[string]*fluxhandler.HelmRepo

var (
	parseBootstrapExtraArgsFn     = parseBootstrapExtraArgs
	decryptBootstrapFilesFn       = decryptBootstrapFiles
	runBootstrapNodeLifecycleFn   = runBootstrapNodeLifecycle
	setupBootstrapClusterFn       = setupBootstrapCluster
	applyManifestFilesFn          = applyManifestFiles
	finalizeBaseClusterFn         = finalizeBaseCluster
	installBootstrapChartPhasesFn = installBootstrapChartPhases
	fluxBootstrapFn               = fluxhandler.FluxBootstrap

	waitForHealthFn             = nodestatus.WaitForHealth
	genApplyFn                  = GenApply
	execCmdsFn                  = ExecCmds
	genPlainFn                  = GenPlain
	execCmdFn                   = ExecCmd
	checkStatusFn               = kubectlcmds.CheckStatus
	getClientsetFn              = kubectlcmds.GetClientset
	loadBootstrapHelmReposFn    = loadBootstrapHelmRepos
	approvePendingCertsFn       = kubectlcmds.ApprovePendingCertificates
	installChartsFn             = fluxhandler.InstallCharts
	baseBootstrapChartsFn       = baseBootstrapCharts
	loadAllHelmReposFn          = fluxhandler.LoadAllHelmRepos
	collectBootstrapFilePathsFn = collectBootstrapFilePaths
	kubectlApplyFn              = kubectlcmds.KubectlApply
	sopsDecryptFilesFn          = sops.DecryptFiles
	osExitFn                    = os.Exit
)

var manifestPaths = []string{
	filepath.Join(helper.KubernetesPath, "flux-system", "flux", "sopssecret.secret.yaml"),
	filepath.Join(helper.KubernetesPath, "flux-system", "flux", "deploykey.secret.yaml"),
	filepath.Join(helper.KubernetesPath, "flux-system", "flux", "clustersettings.secret.yaml"),
}

func RunBootstrap(args []string) {
	extraArgs := parseBootstrapExtraArgsFn(args)
	decryptBootstrapFilesFn()

	bootstrapNode := talassist.TalConfig.Nodes[0].IPAddress
	runBootstrapNodeLifecycleFn(bootstrapNode, extraArgs)

	ctx, stopCh, namespaceFilePaths, vscFilePaths, err := setupBootstrapClusterFn()
	if err != nil {
		return
	}

	if err = applyManifestFilesFn(ctx, namespaceFilePaths, "namespace"); err != nil {
		return
	}
	if err = applyManifestFilesFn(ctx, manifestPaths, "Manifest"); err != nil {
		return
	}

	finalizeBaseClusterFn(stopCh)
	installBootstrapChartPhasesFn(ctx, vscFilePaths)

	log.Info().Msg("------")
	fluxBootstrapFn(ctx)
	log.Info().Msg("Bootstrap: Completed Successfully!")
}

func parseBootstrapExtraArgs(args []string) []string {
	if len(args) > 1 {
		return args[1:]
	}
	return nil
}

func decryptBootstrapFiles() {
	if err := sopsDecryptFilesFn(); err != nil {
		log.Info().Msgf("Error decrypting files: %v\n", err)
	}
}

func runBootstrapNodeLifecycle(bootstrapNode string, extraArgs []string) {
	waitForHealthFn(bootstrapNode, []string{"maintenance"})

	taloscmds := genApplyFn(bootstrapNode, extraArgs)
	execCmdsFn(taloscmds, false)

	waitForHealthFn(bootstrapNode, []string{"booting"})
	log.Info().Msgf("Bootstrap: At this point your system is installed to disk, please make sure not to reboot into the installer ISO/USB  %s", bootstrapNode)

	log.Info().Msgf("Bootstrap: running bootstrap on node:  %s", bootstrapNode)
	bootstrapcmds := genPlainFn("bootstrap", bootstrapNode, extraArgs)
	execCmdFn(bootstrapcmds[0])

	log.Info().Msgf("Bootstrap: waiting for VIP %v to come online...", helper.TalEnv["VIP_IP"])
	waitForHealthFn(helper.TalEnv["VIP_IP"], []string{"running"})

	log.Info().Msgf("Bootstrap: Configuring kubeconfig/kubectl for VIP: %v", helper.TalEnv["VIP_IP"])
	kubeconfigcmds := genPlainFn("kubeconfig", helper.TalEnv["VIP_IP"], []string{"-f"})
	execCmdFn(kubeconfigcmds[0])

	requiredPods := []string{"kube-controller-manager", "kube-scheduler", "kube-apiserver"}
	log.Info().Msgf("Bootstrap: Waiting for system Pods to be running for: %v", helper.TalEnv["VIP_IP"])
	if err := checkStatusFn(requiredPods, []string{}, 600); err != nil {
		log.Error().Err(err).Msgf("Error: %v\n", err)
		osExitFn(1)
	}
}

func setupBootstrapCluster() (context.Context, chan struct{}, []string, []string, error) {
	log.Info().Msg("Bootstrap: Starting Cluster configuration...")
	stopCh := make(chan struct{})

	clientset, err := getClientsetFn()
	if err != nil {
		log.Info().Msgf("Error getting Kubernetes clientset: %v", err)
		return nil, nil, nil, nil, err
	}
	ctx := context.Background()

	if err := loadBootstrapHelmReposFn(); err != nil {
		return nil, nil, nil, nil, err
	}

	go approvePendingCertsFn(clientset, stopCh)
	installChartsFn(baseBootstrapChartsFn(), HelmRepos, true)

	namespaceFilePaths, vscFilePaths, err := collectBootstrapFilePathsFn()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return ctx, stopCh, namespaceFilePaths, vscFilePaths, nil
}

func loadBootstrapHelmRepos() error {
	helmRepoPath := filepath.Join("./repositories", "helm")
	loadedRepos, err := loadAllHelmReposFn(helmRepoPath)
	if err != nil {
		log.Error().Err(err).Msg("Failed to load Helm repositories")
		return err
	}

	HelmRepos = loadedRepos
	return nil
}

func collectBootstrapFilePaths() ([]string, []string, error) {
	var namespaceFilePaths []string
	var vscFilePaths []string

	err := filepath.WalkDir(helper.ClusterPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Base(path) == "namespace.yaml" {
			namespaceFilePaths = append(namespaceFilePaths, path)
		}
		if filepath.Base(path) == "volumeSnapshotClass.yaml" {
			vscFilePaths = append(vscFilePaths, path)
		}
		return nil
	})
	if err != nil {
		log.Info().Msgf("Error walking the path: %v\n", err)
		return nil, nil, err
	}

	return namespaceFilePaths, vscFilePaths, nil
}

func applyManifestFiles(ctx context.Context, files []string, label string) error {
	for _, filePath := range files {
		log.Info().Msgf("Bootstrap: Loading %s: %v", label, filePath)
		if err := kubectlApplyFn(ctx, filePath); err != nil {
			log.Info().Msgf("Error applying manifest for %s: %v\n", filepath.Base(filePath), err)
			osExitFn(1)
		}
	}

	return nil
}

func finalizeBaseCluster(stopCh chan struct{}) {
	log.Info().Msg("Bootstrap: Base Cluster Configuration Completed, continuing setup...")
	log.Info().Msg("Bootstrap: Confirming cluster health...")
	healthcmd := genPlainFn("health", helper.TalEnv["VIP_IP"], []string{})
	execCmdFn(healthcmd[0])
	close(stopCh)
}

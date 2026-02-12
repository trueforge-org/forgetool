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

var manifestPaths = []string{
	filepath.Join(helper.KubernetesPath, "flux-system", "flux", "sopssecret.secret.yaml"),
	filepath.Join(helper.KubernetesPath, "flux-system", "flux", "deploykey.secret.yaml"),
	filepath.Join(helper.KubernetesPath, "flux-system", "flux", "clustersettings.secret.yaml"),
}

func RunBootstrap(args []string) {
	extraArgs := parseBootstrapExtraArgs(args)
	decryptBootstrapFiles()

	bootstrapNode := talassist.TalConfig.Nodes[0].IPAddress
	runBootstrapNodeLifecycle(bootstrapNode, extraArgs)

	ctx, stopCh, namespaceFilePaths, vscFilePaths, err := setupBootstrapCluster()
	if err != nil {
		return
	}

	if err = applyManifestFiles(ctx, namespaceFilePaths, "namespace"); err != nil {
		return
	}
	if err = applyManifestFiles(ctx, manifestPaths, "Manifest"); err != nil {
		return
	}

	finalizeBaseCluster(stopCh)
	installBootstrapChartPhases(ctx, vscFilePaths)

	log.Info().Msg("------")
	fluxhandler.FluxBootstrap(ctx)
	log.Info().Msg("Bootstrap: Completed Successfully!")
}

func parseBootstrapExtraArgs(args []string) []string {
	if len(args) > 1 {
		return args[1:]
	}
	return nil
}

func decryptBootstrapFiles() {
	if err := sops.DecryptFiles(); err != nil {
		log.Info().Msgf("Error decrypting files: %v\n", err)
	}
}

func runBootstrapNodeLifecycle(bootstrapNode string, extraArgs []string) {
	nodestatus.WaitForHealth(bootstrapNode, []string{"maintenance"})

	taloscmds := GenApply(bootstrapNode, extraArgs)
	ExecCmds(taloscmds, false)

	nodestatus.WaitForHealth(bootstrapNode, []string{"booting"})
	log.Info().Msgf("Bootstrap: At this point your system is installed to disk, please make sure not to reboot into the installer ISO/USB  %s", bootstrapNode)

	log.Info().Msgf("Bootstrap: running bootstrap on node:  %s", bootstrapNode)
	bootstrapcmds := GenPlain("bootstrap", bootstrapNode, extraArgs)
	ExecCmd(bootstrapcmds[0])

	log.Info().Msgf("Bootstrap: waiting for VIP %v to come online...", helper.TalEnv["VIP_IP"])
	nodestatus.WaitForHealth(helper.TalEnv["VIP_IP"], []string{"running"})

	log.Info().Msgf("Bootstrap: Configuring kubeconfig/kubectl for VIP: %v", helper.TalEnv["VIP_IP"])
	kubeconfigcmds := GenPlain("kubeconfig", helper.TalEnv["VIP_IP"], []string{"-f"})
	ExecCmd(kubeconfigcmds[0])

	requiredPods := []string{"kube-controller-manager", "kube-scheduler", "kube-apiserver"}
	log.Info().Msgf("Bootstrap: Waiting for system Pods to be running for: %v", helper.TalEnv["VIP_IP"])
	if err := kubectlcmds.CheckStatus(requiredPods, []string{}, 600); err != nil {
		log.Error().Err(err).Msgf("Error: %v\n", err)
		os.Exit(1)
	}
}

func setupBootstrapCluster() (context.Context, chan struct{}, []string, []string, error) {
	log.Info().Msg("Bootstrap: Starting Cluster configuration...")
	stopCh := make(chan struct{})

	clientset, err := kubectlcmds.GetClientset()
	if err != nil {
		log.Info().Msgf("Error getting Kubernetes clientset: %v", err)
		return nil, nil, nil, nil, err
	}
	ctx := context.Background()

	helmRepoPath := filepath.Join("./repositories", "helm")
	HelmRepos, err = fluxhandler.LoadAllHelmRepos(helmRepoPath)
	if err != nil {
		log.Error().Err(err).Msg("Failed to load Helm repositories")
		return nil, nil, nil, nil, err
	}

	go kubectlcmds.ApprovePendingCertificates(clientset, stopCh)

	baseCharts := []fluxhandler.HelmChart{
		{filepath.Join(helper.ClusterPath, "/kubernetes/kube-system/cilium/app"), false, true},
		{filepath.Join(helper.ClusterPath, "/kubernetes/kube-system/kubelet-csr-approver/app"), false, true},
		{filepath.Join(helper.ClusterPath, "/kubernetes/system/kube-prometheus-stack/app"), false, false},
	}
	fluxhandler.InstallCharts(baseCharts, HelmRepos, true)

	namespaceFilePaths, vscFilePaths, err := collectBootstrapFilePaths()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return ctx, stopCh, namespaceFilePaths, vscFilePaths, nil
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
		if err := kubectlcmds.KubectlApply(ctx, filePath); err != nil {
			log.Info().Msgf("Error applying manifest for %s: %v\n", filepath.Base(filePath), err)
			os.Exit(1)
		}
	}

	return nil
}

func finalizeBaseCluster(stopCh chan struct{}) {
	log.Info().Msg("Bootstrap: Base Cluster Configuration Completed, continuing setup...")
	log.Info().Msg("Bootstrap: Confirming cluster health...")
	healthcmd := GenPlain("health", helper.TalEnv["VIP_IP"], []string{})
	ExecCmd(healthcmd[0])
	close(stopCh)
}

func installBootstrapChartPhases(ctx context.Context, vscFilePaths []string) {
	prioCharts := []fluxhandler.HelmChart{
		{filepath.Join(helper.ClusterPath, "/kubernetes/system/spegel/app"), false, true},
		{filepath.Join(helper.ClusterPath, "/kubernetes/system/cert-manager/app"), false, false},
		{filepath.Join(helper.ClusterPath, "/kubernetes/system/kubernetes-reflector/app"), false, false},
	}
	fluxhandler.InstallCharts(prioCharts, HelmRepos, false)

	intermediateCharts := []fluxhandler.HelmChart{
		{filepath.Join(helper.ClusterPath, "/kubernetes/system/metallb/app"), false, false},
		{filepath.Join(helper.ClusterPath, "/kubernetes/core/clusterissuer/app"), false, false},
		{filepath.Join(helper.ClusterPath, "/kubernetes/system/cloudnative-pg/app"), false, false},
		{filepath.Join(helper.ClusterPath, "/kubernetes/kube-system/node-feature-discovery/app"), false, false},
		{filepath.Join(helper.ClusterPath, "/kubernetes/kube-system/metrics-server/app"), false, false},
		{filepath.Join(helper.ClusterPath, "/kubernetes/kube-system/descheduler/app"), false, false},
		{filepath.Join(helper.ClusterPath, "/kubernetes/system/volsync/app"), false, true},
		{filepath.Join(helper.ClusterPath, "/kubernetes/system/snapshot-controller/app"), false, true},
		{filepath.Join(helper.ClusterPath, "/kubernetes/system/openebs/app"), false, true},
		{filepath.Join(helper.ClusterPath, "/kubernetes/system/longhorn/app"), false, true},
	}
	fluxhandler.InstallCharts(intermediateCharts, HelmRepos, true)

	requiredMLBPods := []string{"metallb-controller", "metallb-speaker"}
	log.Info().Msgf("Bootstrap: Waiting for MetalLB Pods to be running for: %v", helper.TalEnv["VIP_IP"])
	if err := kubectlcmds.CheckStatus(requiredMLBPods, []string{}, 600); err != nil {
		log.Error().Err(err).Msgf("Error: %v\n", err)
		os.Exit(1)
	}

	lateCharts := []fluxhandler.HelmChart{{filepath.Join(helper.ClusterPath, "/kubernetes/core/metallb-config/app"), false, false}}

	log.Info().Msgf("Bootstrap: Loading VolumeSnapshotClasses")
	_ = applyManifestFiles(ctx, vscFilePaths, "VolumeSnapshotClass")
	fluxhandler.InstallCharts(lateCharts, HelmRepos, true)

	log.Info().Msg("Bootstrap: Installing included applications")
	postCharts := []fluxhandler.HelmChart{
		{filepath.Join(helper.ClusterPath, "/kubernetes/networking/nginx-internal/app"), false, true},
		{filepath.Join(helper.ClusterPath, "/kubernetes/networking/nginx-external/app"), false, true},
		{filepath.Join(helper.ClusterPath, "/kubernetes/core/blocky/app"), false, true},
		{filepath.Join(helper.ClusterPath, "/kubernetes/apps/kubernetes-dashboard/app"), false, true},
	}
	fluxhandler.InstallCharts(postCharts, HelmRepos, true)
}

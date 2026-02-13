package gencmd

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	talhelperCfg "github.com/budimanjojo/talhelper/v3/pkg/config"
	"github.com/trueforge-org/forgetool/pkg/fluxhandler"
	"k8s.io/client-go/kubernetes"
)

func resetGencmdHooks(t *testing.T) {
	t.Helper()

	parseBootstrapExtraArgsFn = parseBootstrapExtraArgs
	decryptBootstrapFilesFn = decryptBootstrapFiles
	runBootstrapNodeLifecycleFn = runBootstrapNodeLifecycle
	setupBootstrapClusterFn = setupBootstrapCluster
	applyManifestFilesFn = applyManifestFiles
	finalizeBaseClusterFn = finalizeBaseCluster
	installBootstrapChartPhasesFn = installBootstrapChartPhases
	fluxBootstrapFn = fluxhandler.FluxBootstrap

	waitForHealthFn = func(string, []string) (string, error) { return "", nil }
	genApplyFn = GenApply
	execCmdsFn = ExecCmds
	genPlainFn = GenPlain
	execCmdFn = ExecCmd
	checkStatusFn = func([]string, []string, time.Duration) error { return nil }
	getClientsetFn = kubectlClientsetNoop
	loadBootstrapHelmReposFn = loadBootstrapHelmRepos
	approvePendingCertsFn = func(*kubernetes.Clientset, <-chan struct{}) {}
	installChartsFn = func([]fluxhandler.HelmChart, map[string]*fluxhandler.HelmRepo, bool) {}
	baseBootstrapChartsFn = baseBootstrapCharts
	loadAllHelmReposFn = func(string) (map[string]*fluxhandler.HelmRepo, error) { return map[string]*fluxhandler.HelmRepo{}, nil }
	collectBootstrapFilePathsFn = collectBootstrapFilePaths
	kubectlApplyFn = func(context.Context, string) error { return nil }
	sopsDecryptFilesFn = func() error { return nil }
	osExitFn = func(int) {}

	runTalosctlCommandFn = func([]string, bool) (string, error) { return "", nil }
	sleepFn = func(time.Duration) {}
	nowFn = time.Now
	sinceFn = time.Since
	bootstrapRetryTimeout = 2 * time.Minute
	genConfigFn = GenConfig
	extractNodeFn = func(string) string { return "node" }
	checkNodeHealthFn = func(string, string, bool) error { return nil }
	getYesOrNoFn = func(string) bool { return true }

	checkRunAgainFileExistsFn = func() bool { return false }
	loadTalConfigFn = func() {}
	genSchemaFn = func() error { return nil }
	genTalEnvConfigMapFn = func() error { return nil }
	checkEnvVariablesFn = func() {}
	genTalSecretFn = func() error { return nil }
	talhelperGenConfigFn = func() error { return nil }
	updateGitRepoFn = func() {}
	processDirectoryFn = func(string) error { return nil }
	createEncrPreCommitHookFn = func() error { return nil }
	genConfigFatalExitFn = defaultGenConfigFatalExit
	createTalSecretFileFn = os.Create
	encodeSecretBundleFn = defaultEncodeSecretBundle
	writeTalSecretBytesFn = defaultWriteTalSecretBytes

	generateUpgradeCommandFn = func(*talhelperCfg.TalhelperConfig, string, string, []string, bool) error { return nil }
	upgradeFatalFn = defaultUpgradeFatal
	upgradePipeFn = os.Pipe
	upgradeReadAllFn = io.ReadAll
	upgradeCloseReaderFn = defaultUpgradeCloseReader
	upgradeCloseWriterFn = defaultUpgradeCloseWriter
}

func kubectlClientsetNoop() (*kubernetes.Clientset, error) {
	return &kubernetes.Clientset{}, nil
}

type exitPanic struct{}

func expectExitPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		r := recover()
		if _, ok := r.(exitPanic); !ok {
			t.Fatalf("expected exit panic, got %v", r)
		}
	}()
	fn()
}

func markCalledWithIndex(calls *[]string, name string) {
	*calls = append(*calls, name)
}

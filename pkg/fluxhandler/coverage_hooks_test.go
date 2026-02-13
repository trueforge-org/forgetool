package fluxhandler

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"os"
	"path"
	"time"

	"github.com/trueforge-org/forgetool/pkg/helper"
	"github.com/trueforge-org/forgetool/pkg/kubectlcmds"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
)

var (
	defaultFluxLogFatalErrMsgFn             = fluxLogFatalErrMsgFn
	defaultFluxFatalErrMsgFn                = fluxFatalErrMsgFn
	defaultHelmUpgradeRunFn                 = helmUpgradeRunFn
	defaultHelmActionConfigInitFn           = helmActionConfigInitFn
	defaultHelmValuesMergeFn                = helmValuesMergeFn
	defaultHelmGetKubernetesClientSetFn     = helmGetKubernetesClientSetFn
	defaultHelmNamespaceGetFn               = helmNamespaceGetFn
	defaultHelmNamespaceCreateFn            = helmNamespaceCreateFn
	defaultHelmYAMLMarshalFn                = helmYAMLMarshalFn
	defaultHelmWriteFileFn                  = helmWriteFileFn
	defaultHelmNewStatusRunFn               = helmNewStatusRunFn
	defaultHelmInstallRunFn                 = helmInstallRunFn
	defaultHelmPullActionConfigInitFn       = helmPullActionConfigInitFn
	defaultHelmPullClientRunFn              = helmPullClientRunFn
	defaultHelmPullRepoNewChartRepositoryFn = helmPullRepoNewChartRepositoryFn
	defaultHelmPullDownloadIndexFileFn      = helmPullDownloadIndexFileFn
	defaultHelmPullRepoNewFileFn            = helmPullRepoNewFileFn
	defaultHelmPullRepoWriteFileFn          = helmPullRepoWriteFileFn
	defaultHelmPullRegistryNewClientFn      = helmPullRegistryNewClientFn
	defaultSSHGenerateKeyFn                 = sshGenerateKeyFn
	defaultSSHPemBlockForKeyFn              = sshPemBlockForKeyFn
	defaultSSHPublicKeyToOpenSSHFn          = sshPublicKeyToOpenSSHFn
	defaultSSHBuildGitSecretYAMLFn          = sshBuildGitSecretYAMLFn
	defaultSSHYAMLMarshalFn                 = sshYAMLMarshalFn
	defaultSSHYAMLUnmarshalFn               = sshYAMLUnmarshalFn
)

func resetFluxhandlerCoverageHooks() {
	fluxGetYesOrNoFn = helper.GetYesOrNo
	fluxBootstrapFluxCDFn = bootstrapFluxCD
	fluxLogFatalErrMsgFn = defaultFluxLogFatalErrMsgFn
	fluxFatalErrMsgFn = defaultFluxFatalErrMsgFn
	fluxCheckGitRepoFn = checkGitRepo
	fluxSetupFluxCDFn = setupFluxCD
	fluxSetupRepositoriesFn = setupRepositories
	fluxKubectlApplyFn = kubectlcmds.KubectlApply
	fluxIsCurrentDirGitRepoFn = helper.IsCurrentDirGitRepo
	fluxKubectlApplyKustomizeFn = kubectlcmds.KubectlApplyKustomize
	fluxRenameFluxBootstrapFilesFn = renameFluxBootstrapFiles
	fluxRevertFluxBootstrapFilesFn = revertFluxBootstrapFiles
	fluxOSRenameFn = os.Rename
	fluxExitFn = os.Exit

	helmreleaseExitFn = os.Exit
	helmreleaseLoadHelmReleaseFn = LoadHelmRelease
	helmreleaseHelmInstallFn = HelmInstall

	helmupgradeExitFn = os.Exit
	helmupgradeLoadHelmReleaseFn = LoadHelmRelease
	helmupgradeHelmUpgradeFn = HelmUpgrade

	helmInitHelmActionConfigFn = initHelmActionConfig
	helmEnsureNamespaceFn = ensureNamespace
	helmResolveChartPathFn = resolveChartPath
	helmLoaderLoadFn = loader.Load
	helmBuildReleaseValueFilesFn = buildReleaseValueFiles
	helmMergeValueFilesFn = mergeValueFiles
	helmRunInstallWithTimeoutRetryFn = runInstallWithTimeoutRetry
	helmWaitForReleaseFn = waitForRelease
	helmNewInstallFn = action.NewInstall
	helmNewUpgradeFn = action.NewUpgrade
	helmUpgradeRunFn = defaultHelmUpgradeRunFn
	helmActionConfigInitFn = defaultHelmActionConfigInitFn
	helmHelmPullFn = HelmPull
	helmCreateValuesYAMLFn = createValuesYAML
	helmLoadHelmReleaseFn = LoadHelmRelease
	helmEnvSubstFn = helper.EnvSubst
	helmOSStatFn = os.Stat
	helmValuesMergeFn = defaultHelmValuesMergeFn
	helmGetKubernetesClientSetFn = defaultHelmGetKubernetesClientSetFn
	helmNamespaceGetFn = defaultHelmNamespaceGetFn
	helmNamespaceCreateFn = defaultHelmNamespaceCreateFn
	helmYAMLMarshalFn = defaultHelmYAMLMarshalFn
	helmWriteFileFn = defaultHelmWriteFileFn
	helmNewStatusRunFn = defaultHelmNewStatusRunFn
	helmSleepFn = time.Sleep
	helmInstallRunFn = defaultHelmInstallRunFn

	helmPullNewEnvSettingsFn = cli.New
	helmPullActionConfigInitFn = defaultHelmPullActionConfigInitFn
	helmPullNewDefaultRegistryClientFn = newDefaultRegistryClient
	helmPullNewPullWithOptsFn = action.NewPullWithOpts
	helmPullMkdirAllFn = os.MkdirAll
	helmPullConfigureVerificationFn = configureHelmPullVerification
	helmPullResolveLinkFn = resolveHelmPullLink
	helmPullUpdateHelmRepoFn = updateHelmRepo
	helmPullClientRunFn = defaultHelmPullClientRunFn
	helmPullRemoveFn = os.Remove
	helmPullPathJoinFn = path.Join
	helmPullRepoNewChartRepositoryFn = defaultHelmPullRepoNewChartRepositoryFn
	helmPullDownloadIndexFileFn = defaultHelmPullDownloadIndexFileFn
	helmPullRepoNewFileFn = defaultHelmPullRepoNewFileFn
	helmPullRepoLoadFileFn = repo.LoadFile
	helmPullRepoWriteFileFn = defaultHelmPullRepoWriteFileFn
	helmPullRegistryNewClientFn = defaultHelmPullRegistryNewClientFn
	helmPullStatFn = os.Stat
	sshStatFn = os.Stat
	sshCreateNewGitSecretFilesFn = createNewGitSecretFiles
	sshWritePublicKeyFromExistingFn = writePublicKeyFromExistingSecret
	sshGenerateKeyFn = defaultSSHGenerateKeyFn
	sshPemBlockForKeyFn = defaultSSHPemBlockForKeyFn
	sshPublicKeyToOpenSSHFn = defaultSSHPublicKeyToOpenSSHFn
	sshWriteFileFn = os.WriteFile
	sshBuildGitSecretYAMLFn = defaultSSHBuildGitSecretYAMLFn
	sshMkdirAllFn = os.MkdirAll
	sshGetKnownHostsEntryFn = getKnownHostsEntry
	sshReadFileFn = os.ReadFile
	sshYAMLMarshalFn = defaultSSHYAMLMarshalFn
	sshYAMLUnmarshalFn = defaultSSHYAMLUnmarshalFn

	_ = func() (*ecdsa.PrivateKey, error) { return ecdsa.GenerateKey(elliptic.P384(), rand.Reader) }
}

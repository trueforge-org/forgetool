package fluxhandler

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/trueforge-org/forgetool/pkg/helper"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/client-go/kubernetes"
)

func mustPanic(t *testing.T, fn func()) {
	didPanic := false
	defer func() {
		if recover() != nil {
			didPanic = true
		}
		if !didPanic {
			t.Fatalf("expected panic")
		}
	}()
	fn()
}

func setupFluxhandlerTest(t *testing.T) string {
	t.Helper()
	resetFluxhandlerCoverageHooks()
	td := t.TempDir()
	oldCluster := helper.ClusterPath
	oldHelmCache := helper.HelmCache
	oldTalEnv := helper.TalEnv
	helper.ClusterPath = td
	helper.HelmCache = filepath.Join(td, "helmcache")
	helper.TalEnv = map[string]string{}
	t.Cleanup(func() {
		helper.ClusterPath = oldCluster
		helper.HelmCache = oldHelmCache
		helper.TalEnv = oldTalEnv
		resetFluxhandlerCoverageHooks()
	})
	return td
}

func TestBootstrapFlows(t *testing.T) {
	setupFluxhandlerTest(t)
	helper.TalEnv["GITHUB_REPOSITORY"] = "repo"

	fluxGetYesOrNoFn = func(string) bool { return false }
	FluxBootstrap(context.Background())

	calls := 0
	fluxGetYesOrNoFn = func(string) bool { return true }
	fluxBootstrapFluxCDFn = func(context.Context) error {
		calls++
		if calls == 1 {
			return errors.New("first")
		}
		return errors.New("second")
	}
	fluxFatalErrMsgFn = func(error, string) {}
	FluxBootstrap(context.Background())

	fluxExitFn = func(int) { panic("exit") }
	fluxFatalErrMsgFn = func(error, string) { panic("exit") }
	mustPanic(t, func() { fluxFatalErrMsgFn(errors.New("x"), "x") })
	mustPanic(t, func() { fluxLogFatalErrMsgFn(errors.New("x"), "x") })

	fluxCheckGitRepoFn = func() error { return errors.New("git") }
	if err := bootstrapFluxCD(context.Background()); err == nil {
		t.Fatalf("expected checkGitRepo error")
	}
	fluxCheckGitRepoFn = func() error { return nil }
	fluxSetupFluxCDFn = func(context.Context, string) error { return errors.New("flux") }
	if err := bootstrapFluxCD(context.Background()); err == nil {
		t.Fatalf("expected setupFluxCD error")
	}
	fluxSetupFluxCDFn = func(context.Context, string) error { return nil }
	fluxSetupRepositoriesFn = func(context.Context, string) error { return errors.New("repos") }
	if err := bootstrapFluxCD(context.Background()); err == nil {
		t.Fatalf("expected setupRepositories error")
	}
	fluxSetupRepositoriesFn = func(context.Context, string) error { return nil }
	fluxKubectlApplyFn = func(context.Context, string) error { return errors.New("apply") }
	if err := bootstrapFluxCD(context.Background()); err == nil {
		t.Fatalf("expected kubectl apply error")
	}
	fluxKubectlApplyFn = func(context.Context, string) error { return nil }
	if err := bootstrapFluxCD(context.Background()); err != nil {
		t.Fatalf("unexpected bootstrap error: %v", err)
	}

	fluxIsCurrentDirGitRepoFn = func() (bool, error) { return false, nil }
	if err := checkGitRepo(); err == nil {
		t.Fatalf("expected not git repo error")
	}
	fluxIsCurrentDirGitRepoFn = func() (bool, error) { return false, errors.New("check") }
	if err := checkGitRepo(); err == nil {
		t.Fatalf("expected error")
	}
	fluxIsCurrentDirGitRepoFn = func() (bool, error) { return true, nil }
	if err := checkGitRepo(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNoOpLog(t *testing.T) {
	noOpLog("%s", "value")
}

func TestSetupFluxCDAndRepos(t *testing.T) {
	td := setupFluxhandlerTest(t)
	fluxRenameFluxBootstrapFilesFn = func(string, string, string, string) error { return errors.New("rename") }
	if err := setupFluxCD(context.Background(), td); err == nil {
		t.Fatalf("expected rename error")
	}
	fluxRenameFluxBootstrapFilesFn = func(string, string, string, string) error { return nil }
	fluxKubectlApplyKustomizeFn = func(context.Context, string) error { return errors.New("apply") }
	fluxRevertFluxBootstrapFilesFn = func(string, string, string, string) error { return errors.New("revert") }
	if err := setupFluxCD(context.Background(), td); err == nil {
		t.Fatalf("expected revert error")
	}
	fluxRevertFluxBootstrapFilesFn = func(string, string, string, string) error { return nil }
	if err := setupFluxCD(context.Background(), td); err == nil {
		t.Fatalf("expected apply error")
	}
	fluxKubectlApplyKustomizeFn = func(context.Context, string) error { return nil }
	fluxRevertFluxBootstrapFilesFn = func(string, string, string, string) error { return errors.New("revert2") }
	if err := setupFluxCD(context.Background(), td); err == nil {
		t.Fatalf("expected revert after success error")
	}
	fluxRevertFluxBootstrapFilesFn = func(string, string, string, string) error { return nil }
	if err := setupFluxCD(context.Background(), td); err != nil {
		t.Fatalf("unexpected setupFluxCD error: %v", err)
	}

	ops := 0
	fluxOSRenameFn = func(string, string) error {
		ops++
		if ops == 1 {
			return errors.New("first")
		}
		return nil
	}
	if err := renameFluxBootstrapFiles("x", "b", "k", "t"); err == nil {
		t.Fatalf("expected first rename error")
	}
	fluxOSRenameFn = func(string, string) error { return nil }
	if err := renameFluxBootstrapFiles("x", "b", "k", "t"); err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	ops = 0
	fluxOSRenameFn = func(string, string) error {
		ops++
		if ops == 2 {
			return errors.New("second")
		}
		return nil
	}
	if err := revertFluxBootstrapFiles("x", "b", "k", "t"); err == nil {
		t.Fatalf("expected second revert rename error")
	}
	fluxOSRenameFn = func(string, string) error { return errors.New("first-revert") }
	if err := revertFluxBootstrapFiles("x", "b", "k", "t"); err == nil {
		t.Fatalf("expected first revert rename error")
	}
	fluxOSRenameFn = func(string, string) error { return nil }
	if err := revertFluxBootstrapFiles("x", "b", "k", "t"); err != nil {
		t.Fatalf("unexpected revert success error: %v", err)
	}

	calls := 0
	fluxKubectlApplyFn = func(context.Context, string) error {
		calls++
		if calls == 1 {
			return errors.New("first")
		}
		return nil
	}
	if err := setupRepositories(context.Background(), "repos"); err == nil {
		t.Fatalf("expected first apply error")
	}
	fluxKubectlApplyFn = func(context.Context, string) error {
		calls++
		if calls == 3 {
			return errors.New("second")
		}
		return nil
	}
	if err := setupRepositories(context.Background(), "repos"); err == nil {
		t.Fatalf("expected second apply error")
	}
	fluxKubectlApplyFn = func(context.Context, string) error { return nil }
	if err := setupRepositories(context.Background(), "repos"); err != nil {
		t.Fatalf("unexpected setupRepositories error: %v", err)
	}
}

func TestHelmCore(t *testing.T) {
	setupFluxhandlerTest(t)
	if err := HelmInstall("", "", "", "", "", "", true, false, true); err != nil {
		t.Fatalf("dry run should be nil")
	}

	helmInitHelmActionConfigFn = func(string, bool) (*cli.EnvSettings, *action.Configuration, error) {
		return nil, nil, errors.New("init")
	}
	if err := HelmInstall("r", "c", "rel", "ns", "vals", "1.0", false, false, true); err == nil {
		t.Fatalf("expected init error")
	}

	helmInitHelmActionConfigFn = func(string, bool) (*cli.EnvSettings, *action.Configuration, error) {
		return &cli.EnvSettings{}, &action.Configuration{}, nil
	}
	helmEnsureNamespaceFn = func(*action.Configuration, string) error { return errors.New("ns") }
	if err := HelmInstall("r", "c", "rel", "ns", "vals", "1.0", false, false, true); err == nil {
		t.Fatalf("expected ensure namespace error")
	}

	helmEnsureNamespaceFn = func(*action.Configuration, string) error { return nil }
	helmResolveChartPathFn = func(string, string, string) (string, error) { return "", errors.New("path") }
	if err := HelmInstall("r", "c", "rel", "ns", "vals", "1.0", false, false, true); err == nil {
		t.Fatalf("expected resolve path error")
	}

	helmResolveChartPathFn = func(string, string, string) (string, error) { return "x", nil }
	helmLoaderLoadFn = func(string) (*chart.Chart, error) { return nil, errors.New("load") }
	if err := HelmInstall("r", "c", "rel", "ns", "vals", "1.0", false, false, true); err == nil {
		t.Fatalf("expected loader error")
	}

	helmLoaderLoadFn = func(string) (*chart.Chart, error) { return &chart.Chart{Values: map[string]interface{}{}}, nil }
	helmBuildReleaseValueFilesFn = func(string, string, map[string]interface{}, bool) ([]string, error) { return nil, errors.New("values") }
	if err := HelmInstall("r", "c", "rel", "ns", "vals", "1.0", false, false, true); err == nil {
		t.Fatalf("expected build values error")
	}

	helmBuildReleaseValueFilesFn = func(string, string, map[string]interface{}, bool) ([]string, error) { return []string{"a"}, nil }
	helmMergeValueFilesFn = func(*cli.EnvSettings, []string) (map[string]interface{}, error) { return nil, errors.New("merge") }
	if err := HelmInstall("r", "c", "rel", "ns", "vals", "1.0", false, false, true); err == nil {
		t.Fatalf("expected merge error")
	}

	helmMergeValueFilesFn = func(*cli.EnvSettings, []string) (map[string]interface{}, error) { return map[string]interface{}{}, nil }
	helmRunInstallWithTimeoutRetryFn = func(*action.Install, *chart.Chart, map[string]interface{}) (*release.Release, error) {
		return nil, errors.New("run")
	}
	if err := HelmInstall("r", "c", "rel", "ns", "vals", "1.0", false, false, true); err == nil {
		t.Fatalf("expected run error")
	}

	helmRunInstallWithTimeoutRetryFn = func(*action.Install, *chart.Chart, map[string]interface{}) (*release.Release, error) {
		return &release.Release{Name: "rel", Namespace: "ns", Config: map[string]interface{}{}}, nil
	}
	helmWaitForReleaseFn = func(*action.Configuration, string, string) {}
	if err := HelmInstall("r", "c", "rel", "ns", "vals", "1.0", false, true, true); err != nil {
		t.Fatalf("unexpected HelmInstall error: %v", err)
	}

	helmUpgradeRunFn = func(*action.Upgrade, string, *chart.Chart, map[string]interface{}) (*release.Release, error) {
		return &release.Release{Name: "rel", Namespace: "ns", Config: map[string]interface{}{}}, nil
	}
	if err := HelmUpgrade("r", "c", "rel", "ns", "vals", "1.0", true, true); err != nil {
		t.Fatalf("unexpected HelmUpgrade error: %v", err)
	}
	helmUpgradeRunFn = func(*action.Upgrade, string, *chart.Chart, map[string]interface{}) (*release.Release, error) {
		return nil, errors.New("upgrade")
	}
	if err := HelmUpgrade("r", "c", "rel", "ns", "vals", "1.0", false, true); err == nil {
		t.Fatalf("expected HelmUpgrade error")
	}
}

func TestHelmHelpers(t *testing.T) {
	setupFluxhandlerTest(t)

	seq := 0
	helmInstallRunFn = func(*action.Install, *chart.Chart, map[string]interface{}) (*release.Release, error) {
		seq++
		switch seq {
		case 1:
			return nil, errors.New("timed out 1")
		case 2:
			return &release.Release{Name: "ok"}, nil
		default:
			return nil, errors.New("x")
		}
	}
	helmSleepFn = func(time.Duration) {}
	if _, err := runInstallWithTimeoutRetry(&action.Install{}, &chart.Chart{}, map[string]interface{}{}); err != nil {
		t.Fatalf("unexpected retry success error: %v", err)
	}
	seq = 0
	helmInstallRunFn = func(*action.Install, *chart.Chart, map[string]interface{}) (*release.Release, error) {
		return &release.Release{Name: "ok"}, nil
	}
	if _, err := runInstallWithTimeoutRetry(&action.Install{}, &chart.Chart{}, map[string]interface{}{}); err != nil {
		t.Fatalf("expected immediate install success: %v", err)
	}

	seq = 0
	helmInstallRunFn = func(*action.Install, *chart.Chart, map[string]interface{}) (*release.Release, error) {
		seq++
		if seq == 1 {
			return nil, errors.New("timed out")
		}
		return nil, errors.New("timed out again")
	}
	if _, err := runInstallWithTimeoutRetry(&action.Install{}, &chart.Chart{}, map[string]interface{}{}); err == nil {
		t.Fatalf("expected timeout retry error")
	}
	helmInstallRunFn = func(*action.Install, *chart.Chart, map[string]interface{}) (*release.Release, error) {
		return nil, errors.New("boom")
	}
	if _, err := runInstallWithTimeoutRetry(&action.Install{}, &chart.Chart{}, map[string]interface{}{}); err == nil {
		t.Fatalf("expected immediate non-timeout error")
	}

	helmNamespaceGetFn = func(kubernetes.Interface, string) error { return nil }
	helmNamespaceCreateFn = func(kubernetes.Interface, string) error { return nil }
	helmGetKubernetesClientSetFn = func(*action.Configuration) (kubernetes.Interface, error) { return nil, nil }
	if err := ensureNamespace(&action.Configuration{}, "ns"); err != nil {
		t.Fatalf("unexpected ensureNamespace error: %v", err)
	}
	helmNamespaceGetFn = func(kubernetes.Interface, string) error { return errors.New("no") }
	helmNamespaceCreateFn = func(kubernetes.Interface, string) error { return errors.New("create") }
	if err := ensureNamespace(&action.Configuration{}, "ns"); err == nil {
		t.Fatalf("expected ensureNamespace create error")
	}

	helmActionConfigInitFn = func(*action.Configuration, *cli.EnvSettings, string, func(string, ...interface{})) error { return nil }
	if _, _, err := initHelmActionConfig("ns", true); err != nil {
		t.Fatalf("unexpected init config error: %v", err)
	}
	helmActionConfigInitFn = func(*action.Configuration, *cli.EnvSettings, string, func(string, ...interface{})) error {
		return errors.New("init")
	}
	if _, _, err := initHelmActionConfig("ns", false); err == nil {
		t.Fatalf("expected initHelmActionConfig error")
	}

	helmHelmPullFn = func(string, string, string, string, bool) error { return errors.New("pull") }
	if _, err := resolveChartPath("https://repo", "chart", "1.0"); err == nil {
		t.Fatalf("expected resolve pull error")
	}
	helmHelmPullFn = func(string, string, string, string, bool) error { return nil }
	if p, err := resolveChartPath("https://repo", "chart", "1.0"); err != nil || !strings.Contains(p, "chart-1.0.tgz") {
		t.Fatalf("unexpected resolved path: %s err=%v", p, err)
	}
	if p, err := resolveChartPath("/local/chart.tgz", "chart", "1.0"); err != nil || p != "/local/chart.tgz" {
		t.Fatalf("expected passthrough path")
	}

	td := t.TempDir()
	helper.HelmCache = td
	valuesPath := filepath.Join(td, "vals.yaml")
	if err := os.WriteFile(valuesPath, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	override := filepath.Join(td, "bootstrap-values.yaml.ct")
	if err := os.WriteFile(override, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_ = filepath.Join(td, "helm-release.yaml")
	helmLoadHelmReleaseFn = func(string) (*HelmRelease, error) {
		return &HelmRelease{Spec: Spec{Values: map[string]interface{}{"k": "v"}}}, nil
	}
	helmCreateValuesYAMLFn = func(map[string]interface{}, string) error { return nil }
	helmEnvSubstFn = func(string, map[string]string) (string, error) { return "", nil }
	if files, err := buildReleaseValueFiles("r", valuesPath, map[string]interface{}{}, false); err != nil || len(files) < 2 {
		t.Fatalf("expected value files, got %v err=%v", files, err)
	}
	helmCreateValuesYAMLFn = func(map[string]interface{}, string) error { return errors.New("temp") }
	if _, err := buildReleaseValueFiles("r", valuesPath, map[string]interface{}{}, false); err == nil {
		t.Fatalf("expected tempvalues create error")
	}
	helmCreateValuesYAMLFn = func(map[string]interface{}, string) error { return nil }
	helmLoadHelmReleaseFn = func(string) (*HelmRelease, error) { return nil, errors.New("hr") }
	if _, err := buildReleaseValueFiles("r", valuesPath, map[string]interface{}{}, true); err == nil {
		t.Fatalf("expected strict helm-release load error")
	}

	helmValuesMergeFn = func(*values.Options, *cli.EnvSettings) (map[string]interface{}, error) {
		return map[string]interface{}{"a": 1}, nil
	}
	if vals, err := mergeValueFiles(&cli.EnvSettings{}, []string{"a"}); err != nil || vals["a"] != 1 {
		t.Fatalf("mergeValueFiles unexpected: %v err=%v", vals, err)
	}
	helmValuesMergeFn = func(*values.Options, *cli.EnvSettings) (map[string]interface{}, error) {
		return nil, errors.New("merge")
	}
	if _, err := mergeValueFiles(&cli.EnvSettings{}, []string{"a"}); err == nil {
		t.Fatalf("expected merge error")
	}

	helmGetKubernetesClientSetFn = func(*action.Configuration) (kubernetes.Interface, error) { return nil, errors.New("client") }
	if _, err := namespaceExists(&action.Configuration{}, "ns"); err == nil {
		t.Fatalf("expected namespaceExists client error")
	}
	helmGetKubernetesClientSetFn = func(*action.Configuration) (kubernetes.Interface, error) { return nil, nil }
	helmNamespaceGetFn = func(kubernetes.Interface, string) error { return errors.New("no") }
	if ok, err := namespaceExists(&action.Configuration{}, "ns"); err != nil || ok {
		t.Fatalf("expected not found semantics, got ok=%v err=%v", ok, err)
	}
	helmNamespaceGetFn = func(kubernetes.Interface, string) error { return nil }
	if ok, err := namespaceExists(&action.Configuration{}, "ns"); err != nil || !ok {
		t.Fatalf("expected exists semantics")
	}

	helmGetKubernetesClientSetFn = func(*action.Configuration) (kubernetes.Interface, error) { return nil, errors.New("client") }
	if err := createNamespace(&action.Configuration{}, "ns"); err == nil {
		t.Fatalf("expected createNamespace client error")
	}
	helmGetKubernetesClientSetFn = func(*action.Configuration) (kubernetes.Interface, error) { return nil, nil }
	helmNamespaceCreateFn = func(kubernetes.Interface, string) error { return errors.New("already exists") }
	if err := createNamespace(&action.Configuration{}, "ns"); err != nil {
		t.Fatalf("already exists should be ignored: %v", err)
	}
	helmNamespaceCreateFn = func(kubernetes.Interface, string) error { return errors.New("boom") }
	if err := createNamespace(&action.Configuration{}, "ns"); err == nil {
		t.Fatalf("expected create namespace error")
	}

	helmYAMLMarshalFn = func(any) ([]byte, error) { return nil, errors.New("marshal") }
	if err := createValuesYAML(map[string]interface{}{}, filepath.Join(td, "x.yaml")); err == nil {
		t.Fatalf("expected marshal error")
	}
	helmYAMLMarshalFn = func(any) ([]byte, error) { return []byte("a: b"), nil }
	helmWriteFileFn = func(string, []byte, os.FileMode) error { return errors.New("write") }
	if err := createValuesYAML(map[string]interface{}{}, filepath.Join(td, "x2.yaml")); err == nil {
		t.Fatalf("expected write error")
	}

	dir := filepath.Join(td, "dir")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "x"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := removeFileIfExists(dir); err == nil {
		t.Fatalf("expected remove directory error")
	}

	statusCalls := 0
	helmNewStatusRunFn = func(*action.Configuration, string) (*release.Release, error) {
		statusCalls++
		if statusCalls == 1 {
			return &release.Release{Info: &release.Info{Status: release.StatusPendingInstall}}, errors.New("status")
		}
		return &release.Release{Info: &release.Info{Status: release.StatusDeployed}}, nil
	}
	helmSleepFn = func(time.Duration) {}
	waitForRelease(&action.Configuration{}, "r", "ns")
}

func TestHelmPullAndRepo(t *testing.T) {
	setupFluxhandlerTest(t)
	helmPullActionConfigInitFn = func(*action.Configuration, *cli.EnvSettings, func(string, ...interface{})) error {
		return errors.New("init")
	}
	if err := HelmPull("repo", "chart", "1", "", true); err == nil {
		t.Fatalf("expected init error")
	}
	resetFluxhandlerCoverageHooks()
	helmPullNewDefaultRegistryClientFn = func(bool, *cli.EnvSettings) (*registry.Client, error) {
		return nil, errors.New("registry")
	}
	if err := HelmPull("repo", "chart", "1", "", true); err == nil {
		t.Fatalf("expected registry error")
	}
	resetFluxhandlerCoverageHooks()
	helmPullMkdirAllFn = func(string, os.FileMode) error { return errors.New("mkdir") }
	if err := HelmPull("repo", "chart", "1", "", true); err == nil {
		t.Fatalf("expected mkdir error")
	}
	resetFluxhandlerCoverageHooks()
	helmPullResolveLinkFn = func(string, string) (string, string) { return "link", "repo" }
	helmPullClientRunFn = func(*action.Pull, string) (string, error) { return "", errors.New("run") }
	if err := HelmPull("repo", "chart", "1", "", true); err == nil {
		t.Fatalf("expected run error")
	}
	resetFluxhandlerCoverageHooks()
	helmPullClientRunFn = func(*action.Pull, string) (string, error) { return "ok", nil }
	if err := HelmPull("repo", "chart", "1", "", false); err != nil {
		t.Fatalf("unexpected helm pull error: %v", err)
	}

	pullClient := &action.Pull{}
	configureHelmPullVerification(pullClient, "https://charts.trueforge.org")
	configureHelmPullVerification(pullClient, "https://charts.jetstack.io")
	configureHelmPullVerification(pullClient, "https://example.com")
	link, repoURL := resolveHelmPullLink("https://repo", "chart")
	if link != "chart" || repoURL != "https://repo" {
		t.Fatalf("unexpected resolve link")
	}
	link, repoURL = resolveHelmPullLink("oci://repo", "chart")
	if !strings.Contains(link, "chart") || repoURL != "" {
		t.Fatalf("unexpected non-http link")
	}
	noOpLog("%s", "x")

	helmPullRepoNewChartRepositoryFn = func(*repo.Entry, *cli.EnvSettings) (*repo.ChartRepository, error) {
		return nil, errors.New("newrepo")
	}
	if err := updateHelmRepo("n", "u", true); err == nil {
		t.Fatalf("expected new repo error")
	}
	resetFluxhandlerCoverageHooks()
	helmPullMkdirAllFn = func(string, os.FileMode) error { return errors.New("mkdir") }
	if err := updateHelmRepo("n", "u", true); err == nil {
		t.Fatalf("expected cache mkdir error")
	}
	resetFluxhandlerCoverageHooks()
	helmPullRepoNewChartRepositoryFn = func(*repo.Entry, *cli.EnvSettings) (*repo.ChartRepository, error) {
		return &repo.ChartRepository{}, nil
	}
	helmPullMkdirAllFn = func(string, os.FileMode) error { return errors.New("mkdir-cache") }
	if err := updateHelmRepo("n", "u", true); err == nil {
		t.Fatalf("expected cache-dir mkdir branch error")
	}
	resetFluxhandlerCoverageHooks()
	helmPullDownloadIndexFileFn = func(*repo.ChartRepository) error { return errors.New("idx") }
	if err := updateHelmRepo("n", "u", true); err == nil {
		t.Fatalf("expected index download error")
	}
	resetFluxhandlerCoverageHooks()
	helmPullRepoLoadFileFn = func(string) (*repo.File, error) { return nil, errors.New("loadfile") }
	helmPullStatFn = func(string) (os.FileInfo, error) { return fakeInfo{}, nil }
	if err := updateHelmRepo("n", "u", true); err == nil {
		t.Fatalf("expected repo load file error")
	}
	resetFluxhandlerCoverageHooks()
	helmPullRepoWriteFileFn = func(*repo.File, string) error { return errors.New("write") }
	if err := updateHelmRepo("n", "u", true); err == nil {
		t.Fatalf("expected repo write file error")
	}
}

func TestInstallUpgradeAndSSHFlows(t *testing.T) {
	setupFluxhandlerTest(t)
	helmRepos := map[string]*HelmRepo{"repo": {Spec: HelmRepoSpec{URL: "https://repo"}}}
	chartItem := HelmChart{ChartPath: "/c", Retry: false, Wait: false}

	helmreleaseLoadHelmReleaseFn = func(string) (*HelmRelease, error) { return nil, errors.New("load") }
	helmreleaseExitFn = func(int) { panic("exit") }
	mustPanic(t, func() { processInstallChart(chartItem, helmRepos, false) })

	helmreleaseLoadHelmReleaseFn = func(string) (*HelmRelease, error) { return &HelmRelease{}, nil }
	mustPanic(t, func() { processInstallChart(chartItem, map[string]*HelmRepo{}, false) })
	mustPanic(t, func() { resolveInstallChartContext(chartItem, "x", map[string]*HelmRepo{}) })

	helmreleaseLoadHelmReleaseFn = func(string) (*HelmRelease, error) {
		return &HelmRelease{Metadata: Metadata{Name: "r", Namespace: "ns"}, Spec: Spec{Chart: Chart{Spec: ChartSpec{Chart: "c", Version: "1", SourceRef: SourceRef{Name: "repo"}}}}}, nil
	}
	helmreleaseHelmInstallFn = func(string, string, string, string, string, string, bool, bool, bool) error {
		return errors.New("webhook failed")
	}
	processInstallChart(chartItem, helmRepos, true)
	helmreleaseHelmInstallFn = func(string, string, string, string, string, string, bool, bool, bool) error {
		return errors.New("boom")
	}
	mustPanic(t, func() { processInstallChart(chartItem, helmRepos, false) })
	helmreleaseHelmInstallFn = func(string, string, string, string, string, string, bool, bool, bool) error { return nil }
	processInstallChart(chartItem, helmRepos, false)
	InstallCharts([]HelmChart{chartItem}, helmRepos, false)
	InstallCharts([]HelmChart{chartItem, chartItem}, helmRepos, true)

	helmupgradeLoadHelmReleaseFn = func(string) (*HelmRelease, error) { return nil, errors.New("load") }
	helmupgradeExitFn = func(int) { panic("exit") }
	mustPanic(t, func() { processUpgradeChart(chartItem, helmRepos, false) })
	processUpgradeChart(chartItem, helmRepos, true)
	helmupgradeLoadHelmReleaseFn = func(string) (*HelmRelease, error) { return nil, nil }
	mustPanic(t, func() { processUpgradeChart(chartItem, helmRepos, false) })
	helmupgradeLoadHelmReleaseFn = func(string) (*HelmRelease, error) {
		return &HelmRelease{Metadata: Metadata{Name: "meta", Namespace: "ns"}, Spec: Spec{ReleaseName: "override", Chart: Chart{Spec: ChartSpec{Chart: "c", Version: "1", SourceRef: SourceRef{Name: "repo"}}}}}, nil
	}
	helmupgradeHelmUpgradeFn = func(repoURL, chartName, releaseName, namespace, valuesFile, version string, wait, silent bool) error {
		if releaseName != "override" {
			return errors.New("releaseName override not used")
		}
		return nil
	}
	processUpgradeChart(chartItem, helmRepos, true)
	helmupgradeLoadHelmReleaseFn = helmreleaseLoadHelmReleaseFn
	mustPanic(t, func() { processUpgradeChart(chartItem, map[string]*HelmRepo{}, false) })
	helmupgradeHelmUpgradeFn = func(string, string, string, string, string, string, bool, bool) error { return errors.New("upg") }
	mustPanic(t, func() { processUpgradeChart(chartItem, helmRepos, false) })
	helmupgradeHelmUpgradeFn = func(string, string, string, string, string, string, bool, bool) error { return nil }
	processUpgradeChart(chartItem, helmRepos, false)
	UpgradeCharts([]HelmChart{chartItem}, helmRepos, false)
	UpgradeCharts([]HelmChart{chartItem, chartItem}, helmRepos, true)

	sshStatFn = func(string) (os.FileInfo, error) { return nil, os.ErrNotExist }
	sshCreateNewGitSecretFilesFn = func(string, string, string) error { return errors.New("new") }
	if err := CreateGitSecret("x"); err == nil {
		t.Fatalf("expected create new secret error")
	}
	sshCreateNewGitSecretFilesFn = func(string, string, string) error { return nil }
	if err := CreateGitSecret("x"); err != nil {
		t.Fatalf("unexpected create git secret error: %v", err)
	}
	sshStatFn = func(path string) (os.FileInfo, error) {
		if strings.Contains(path, "deploykey") {
			return nil, nil
		}
		return nil, os.ErrNotExist
	}
	sshWritePublicKeyFromExistingFn = func(string, string) error { return errors.New("existing") }
	if err := CreateGitSecret("x"); err == nil {
		t.Fatalf("expected existing secret path error")
	}

	sshGenerateKeyFn = func() (*ecdsa.PrivateKey, error) { return nil, errors.New("key") }
	if err := createNewGitSecretFiles("x", "s", "p"); err == nil {
		t.Fatalf("expected key generation error")
	}
	sshGenerateKeyFn = defaultSSHGenerateKeyFn
	sshPemBlockForKeyFn = func(*ecdsa.PrivateKey) ([]byte, error) { return nil, errors.New("pem") }
	if err := createNewGitSecretFiles("x", "s", "p"); err == nil {
		t.Fatalf("expected pem error")
	}
	sshPemBlockForKeyFn = defaultSSHPemBlockForKeyFn
	sshPublicKeyToOpenSSHFn = func(*ecdsa.PublicKey) (string, error) { return "", errors.New("pub") }
	if err := createNewGitSecretFiles("x", "s", "p"); err == nil {
		t.Fatalf("expected public key conversion error")
	}

	sshPublicKeyToOpenSSHFn = defaultSSHPublicKeyToOpenSSHFn
	sshWriteFileFn = func(string, []byte, os.FileMode) error { return errors.New("write") }
	if err := createNewGitSecretFiles("x", "s", "p"); err == nil {
		t.Fatalf("expected write public key error")
	}
	sshWriteFileFn = os.WriteFile
	sshBuildGitSecretYAMLFn = func(string, string, string) ([]byte, error) { return nil, errors.New("yaml") }
	if err := createNewGitSecretFiles("x", "s", "p"); err == nil {
		t.Fatalf("expected build yaml error")
	}
	sshBuildGitSecretYAMLFn = defaultSSHBuildGitSecretYAMLFn
	sshMkdirAllFn = func(string, os.FileMode) error { return errors.New("mkdir") }
	if err := createNewGitSecretFiles("x", "s", "p"); err == nil {
		t.Fatalf("expected mkdir error")
	}
	sshMkdirAllFn = os.MkdirAll
	sshWriteFileFn = func(string, []byte, os.FileMode) error { return errors.New("write2") }
	if err := createNewGitSecretFiles("x", filepath.Join(t.TempDir(), "a", "b", "c"), filepath.Join(t.TempDir(), "pub")); err == nil {
		t.Fatalf("expected write secret error")
	}

	sshReadFileFn = func(string) ([]byte, error) { return nil, errors.New("read") }
	if err := writePublicKeyFromExistingSecret("s", "p"); err == nil {
		t.Fatalf("expected read existing secret error")
	}
	sshReadFileFn = func(string) ([]byte, error) { return []byte("bad"), nil }
	sshYAMLUnmarshalFn = func([]byte, interface{}) error { return errors.New("unmarshal") }
	if err := writePublicKeyFromExistingSecret("s", "p"); err == nil {
		t.Fatalf("expected unmarshal error")
	}
	sshYAMLUnmarshalFn = defaultSSHYAMLUnmarshalFn
	sshReadFileFn = func(string) ([]byte, error) { return []byte("stringData: {}"), nil }
	if err := writePublicKeyFromExistingSecret("s", "p"); err == nil {
		t.Fatalf("expected missing identity.pub error")
	}

	sshYAMLMarshalFn = func(any) ([]byte, error) { return nil, errors.New("marshal") }
	if _, err := buildGitSecretYAML("i", "p", "k"); err == nil {
		t.Fatalf("expected buildGitSecretYAML marshal error")
	}
	sshYAMLMarshalFn = yaml.Marshal

	badKey := &ecdsa.PrivateKey{}
	if _, err := pemBlockForKey(badKey); err == nil {
		t.Fatalf("expected pemBlockForKey error")
	}
	mustPanic(t, func() {
		_, _ = publicKeyToOpenSSH(nil)
	})
}

func TestKustomizationErrorBranches(t *testing.T) {
	td := t.TempDir()
	if err := createOrUpdateKustomizationYaml(filepath.Join(td, "missing")); err == nil {
		t.Fatalf("expected missing dir error")
	}
	if err := ProcessDirectory(filepath.Join(td, "missing")); err == nil {
		t.Fatalf("expected process missing dir error")
	}
	d := filepath.Join(td, "x")
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(d, "kustomization.yaml"), []byte("apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\nresources:\n  - sub\n  - sub/ks.yaml\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := createOrUpdateKustomizationYaml(d); err != nil {
		t.Fatalf("unexpected kustomization update error: %v", err)
	}

	recurseParent := t.TempDir()
	recurseChild := filepath.Join(recurseParent, "child")
	if err := os.MkdirAll(recurseChild, 0o755); err != nil {
		t.Fatalf("mkdir recurse child: %v", err)
	}
	if err := os.Chmod(recurseChild, 0o500); err != nil {
		t.Fatalf("chmod recurse child: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(recurseChild, 0o700) })
	if err := ProcessDirectory(recurseParent); err == nil {
		t.Fatalf("expected recursive child error propagation")
	}
}

type fakeInfo struct{}

func (fakeInfo) Name() string       { return "f" }
func (fakeInfo) Size() int64        { return 0 }
func (fakeInfo) Mode() os.FileMode  { return 0 }
func (fakeInfo) ModTime() time.Time { return time.Now() }
func (fakeInfo) IsDir() bool        { return false }
func (fakeInfo) Sys() interface{}   { return nil }

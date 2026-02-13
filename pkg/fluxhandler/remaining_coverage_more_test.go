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

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

func TestRemainingCoverageBootstrapAndHelm(t *testing.T) {
	td := setupFluxhandlerTest(t)

	oldFatal := fluxLogFatalErrMsgFn
	fluxLogFatalErrMsgFn = func(error, string) {}
	t.Cleanup(func() { fluxLogFatalErrMsgFn = oldFatal })
	fluxFatalErrMsgFn(errors.New("fatal"), "msg")

	calls := 0
	fluxKubectlApplyFn = func(_ context.Context, _ string) error {
		calls++
		if calls == 2 {
			return errors.New("second apply")
		}
		return nil
	}
	if err := setupRepositories(nil, td); err == nil {
		t.Fatalf("expected second apply error")
	}
	fluxKubectlApplyFn = func(_ context.Context, _ string) error { return nil }
	if err := setupRepositories(nil, td); err != nil {
		t.Fatalf("expected setupRepositories success: %v", err)
	}

	mustPanic(t, func() {
		_, _ = defaultHelmUpgradeRunFn(nil, "r", &chart.Chart{}, map[string]interface{}{})
	})
	_ = defaultHelmActionConfigInitFn(new(action.Configuration), cli.New(), "ns", noOpLog)
	_, _ = defaultHelmValuesMergeFn(&values.Options{}, cli.New())
	mustPanic(t, func() {
		_, _ = defaultHelmGetKubernetesClientSetFn(new(action.Configuration))
	})
	mustPanic(t, func() { _ = defaultHelmNamespaceGetFn(nil, "ns") })
	mustPanic(t, func() { _ = defaultHelmNamespaceCreateFn(nil, "ns") })
	if _, err := defaultHelmYAMLMarshalFn(map[string]interface{}{"a": "b"}); err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}
	if err := defaultHelmWriteFileFn(filepath.Join(t.TempDir(), "v.yaml"), []byte("a: b"), 0o644); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	mustPanic(t, func() {
		_, _ = defaultHelmNewStatusRunFn(new(action.Configuration), "r")
	})
	mustPanic(t, func() {
		_, _ = defaultHelmInstallRunFn(nil, &chart.Chart{}, map[string]interface{}{})
	})

	runCalls := 0
	helmInstallRunFn = func(*action.Install, *chart.Chart, map[string]interface{}) (*release.Release, error) {
		runCalls++
		if runCalls == 1 {
			return nil, errors.New("timed out once")
		}
		return nil, errors.New("non-timeout")
	}
	helmSleepFn = func(time.Duration) {}
	if _, err := runInstallWithTimeoutRetry(&action.Install{}, &chart.Chart{}, map[string]interface{}{}); err == nil || !strings.Contains(err.Error(), "after retry") {
		t.Fatalf("expected post-retry error path, got: %v", err)
	}

	helmGetKubernetesClientSetFn = func(*action.Configuration) (kubernetes.Interface, error) { return nil, errors.New("client") }
	if err := ensureNamespace(&action.Configuration{}, "ns"); err == nil {
		t.Fatalf("expected ensureNamespace namespaceExists error")
	}
	helmGetKubernetesClientSetFn = func(*action.Configuration) (kubernetes.Interface, error) { return nil, nil }
	helmNamespaceGetFn = func(kubernetes.Interface, string) error { return errors.New("missing") }
	helmNamespaceCreateFn = func(kubernetes.Interface, string) error { return nil }
	if err := ensureNamespace(&action.Configuration{}, "ns"); err != nil {
		t.Fatalf("expected ensureNamespace create success: %v", err)
	}
	helmNamespaceCreateFn = func(kubernetes.Interface, string) error { return errors.New("boom") }
	if err := ensureNamespace(&action.Configuration{}, "ns"); err == nil {
		t.Fatalf("expected ensureNamespace create error")
	}
	helmNamespaceGetFn = func(kubernetes.Interface, string) error { return nil }
	if err := ensureNamespace(&action.Configuration{}, "ns"); err != nil {
		t.Fatalf("expected ensureNamespace exists success: %v", err)
	}

	helmInitHelmActionConfigFn = func(string, bool) (*cli.EnvSettings, *action.Configuration, error) {
		return nil, nil, errors.New("init")
	}
	if err := HelmUpgrade("r", "c", "rel", "ns", "vals", "1", false, true); err == nil {
		t.Fatalf("expected HelmUpgrade init error")
	}
	helmInitHelmActionConfigFn = func(string, bool) (*cli.EnvSettings, *action.Configuration, error) {
		return &cli.EnvSettings{}, &action.Configuration{}, nil
	}
	helmEnsureNamespaceFn = func(*action.Configuration, string) error { return errors.New("ensure") }
	if err := HelmUpgrade("r", "c", "rel", "ns", "vals", "1", false, true); err == nil {
		t.Fatalf("expected HelmUpgrade ensure error")
	}
	helmEnsureNamespaceFn = func(*action.Configuration, string) error { return nil }
	helmResolveChartPathFn = func(string, string, string) (string, error) { return "", errors.New("path") }
	if err := HelmUpgrade("r", "c", "rel", "ns", "vals", "1", false, true); err == nil {
		t.Fatalf("expected HelmUpgrade resolve error")
	}
	helmResolveChartPathFn = func(string, string, string) (string, error) { return "ok", nil }
	helmLoaderLoadFn = func(string) (*chart.Chart, error) { return nil, errors.New("load") }
	if err := HelmUpgrade("r", "c", "rel", "ns", "vals", "1", false, true); err == nil {
		t.Fatalf("expected HelmUpgrade load error")
	}
	helmLoaderLoadFn = func(string) (*chart.Chart, error) { return &chart.Chart{Values: map[string]interface{}{}}, nil }
	helmBuildReleaseValueFilesFn = func(string, string, map[string]interface{}, bool) ([]string, error) {
		return nil, errors.New("vals")
	}
	if err := HelmUpgrade("r", "c", "rel", "ns", "vals", "1", false, true); err == nil {
		t.Fatalf("expected HelmUpgrade build values error")
	}
	helmBuildReleaseValueFilesFn = func(string, string, map[string]interface{}, bool) ([]string, error) {
		return []string{"a"}, nil
	}
	helmMergeValueFilesFn = func(*cli.EnvSettings, []string) (map[string]interface{}, error) { return nil, errors.New("merge") }
	if err := HelmUpgrade("r", "c", "rel", "ns", "vals", "1", false, true); err == nil {
		t.Fatalf("expected HelmUpgrade merge error")
	}

	valuesPath := filepath.Join(td, "vals.yaml")
	if err := os.WriteFile(valuesPath, []byte("x"), 0o644); err != nil {
		t.Fatalf("write values: %v", err)
	}
	createCalls := 0
	helmCreateValuesYAMLFn = func(map[string]interface{}, string) error {
		createCalls++
		if createCalls == 2 {
			return errors.New("hr values")
		}
		return nil
	}
	helmLoadHelmReleaseFn = func(string) (*HelmRelease, error) {
		return &HelmRelease{Spec: Spec{Values: map[string]interface{}{"k": "v"}}}, nil
	}
	if _, err := buildReleaseValueFiles("r", valuesPath, map[string]interface{}{}, false); err == nil {
		t.Fatalf("expected temphrvalues creation error")
	}

	if err := removeFileIfExists("\x00"); err == nil {
		t.Fatalf("expected removeFileIfExists invalid-path error")
	}
}

func TestRemainingCoverageHelmPullAndRepo(t *testing.T) {
	setupFluxhandlerTest(t)

	mustPanic(t, func() {
		_, _ = defaultHelmPullClientRunFn(nil, "chart")
	})

	helmPullRegistryNewClientFn = func(...registry.ClientOption) (*registry.Client, error) {
		return nil, errors.New("registry")
	}
	if _, err := newDefaultRegistryClient(false, &cli.EnvSettings{}); err == nil {
		t.Fatalf("expected newDefaultRegistryClient error")
	}

	updatedRepo := false
	helmPullActionConfigInitFn = func(*action.Configuration, *cli.EnvSettings, func(string, ...interface{})) error { return nil }
	helmPullNewDefaultRegistryClientFn = func(bool, *cli.EnvSettings) (*registry.Client, error) {
		return &registry.Client{}, nil
	}
	helmPullNewPullWithOptsFn = func(...action.PullOpt) *action.Pull { return &action.Pull{} }
	helmPullMkdirAllFn = func(string, os.FileMode) error { return nil }
	helmPullConfigureVerificationFn = func(client *action.Pull, _ string) {
		client.Keyring = "set"
		client.Verify = true
	}
	helmPullResolveLinkFn = func(string, string) (string, string) { return "chart", "http://repo" }
	helmPullUpdateHelmRepoFn = func(string, string, bool) error {
		updatedRepo = true
		return nil
	}
	helmPullClientRunFn = func(*action.Pull, string) (string, error) { return "ok", nil }
	if err := HelmPull("http://charts.example.com", "chart", "1.0.0", t.TempDir(), false); err != nil {
		t.Fatalf("expected HelmPull success: %v", err)
	}
	if !updatedRepo {
		t.Fatalf("expected HelmPull to call update repo for http repos")
	}

	cacheDir := t.TempDir()
	repoFilePath := filepath.Join(cacheDir, "repositories.yaml")
	helmPullNewEnvSettingsFn = func() *cli.EnvSettings {
		return &cli.EnvSettings{RepositoryCache: cacheDir, RepositoryConfig: repoFilePath}
	}
	helmPullRepoNewChartRepositoryFn = func(*repo.Entry, *cli.EnvSettings) (*repo.ChartRepository, error) {
		return &repo.ChartRepository{}, nil
	}
	helmPullMkdirAllFn = func(string, os.FileMode) error { return nil }
	helmPullDownloadIndexFileFn = func(*repo.ChartRepository) error { return errors.New("index") }
	if err := updateHelmRepo("n", "u", true); err == nil {
		t.Fatalf("expected updateHelmRepo download index error")
	}

	helmPullDownloadIndexFileFn = func(*repo.ChartRepository) error { return nil }
	helmPullStatFn = func(string) (os.FileInfo, error) { return nil, nil }
	helmPullRepoLoadFileFn = func(string) (*repo.File, error) { return nil, errors.New("load") }
	if err := updateHelmRepo("n", "u", true); err == nil {
		t.Fatalf("expected updateHelmRepo load file error")
	}

	helmPullStatFn = func(string) (os.FileInfo, error) { return nil, os.ErrNotExist }
	helmPullRepoNewFileFn = func() *repo.File { return repo.NewFile() }
	helmPullRepoWriteFileFn = func(*repo.File, string) error { return errors.New("write") }
	if err := updateHelmRepo("n", "u", true); err == nil {
		t.Fatalf("expected updateHelmRepo write error")
	}

	helmPullRepoWriteFileFn = func(*repo.File, string) error { return nil }
	if err := updateHelmRepo("n", "u", false); err != nil {
		t.Fatalf("expected updateHelmRepo success: %v", err)
	}
}

func TestRemainingCoverageHelmReleaseAndUpgrade(t *testing.T) {
	setupFluxhandlerTest(t)
	chartItem := HelmChart{ChartPath: "/x"}

	helmreleaseExitFn = func(int) { panic("exit") }
	helmreleaseLoadHelmReleaseFn = func(string) (*HelmRelease, error) { return nil, nil }
	mustPanic(t, func() {
		_, _, _ = resolveInstallChartContext(chartItem, "hr.yaml", map[string]*HelmRepo{})
	})

	helmreleaseExitFn = func(int) {}
	helmreleaseLoadHelmReleaseFn = func(string) (*HelmRelease, error) {
		return &HelmRelease{
			Metadata: Metadata{Name: "a", Namespace: "ns"},
			Spec: Spec{
				ReleaseName: "override",
				Chart:       Chart{Spec: ChartSpec{SourceRef: SourceRef{Name: "repo"}}},
			},
		}, nil
	}
	releaseName, _, repoURL := resolveInstallChartContext(chartItem, "hr.yaml", map[string]*HelmRepo{
		"repo": {Spec: HelmRepoSpec{URL: "https://repo"}},
	})
	if releaseName != "override" || repoURL != "https://repo" {
		t.Fatalf("expected release override and repo URL")
	}
	_, _, _ = resolveInstallChartContext(chartItem, "hr.yaml", map[string]*HelmRepo{
		"repo": {Spec: HelmRepoSpec{URL: ""}},
	})

	helmupgradeExitFn = func(int) {}
	helmupgradeLoadHelmReleaseFn = func(string) (*HelmRelease, error) { return nil, errors.New("load") }
	processUpgradeChart(chartItem, map[string]*HelmRepo{}, false)
	helmupgradeLoadHelmReleaseFn = func(string) (*HelmRelease, error) { return nil, nil }
	processUpgradeChart(chartItem, map[string]*HelmRepo{}, false)
	helmupgradeLoadHelmReleaseFn = func(string) (*HelmRelease, error) {
		return &HelmRelease{Spec: Spec{Chart: Chart{Spec: ChartSpec{SourceRef: SourceRef{Name: "missing"}}}}}, nil
	}
	processUpgradeChart(chartItem, map[string]*HelmRepo{}, false)
}

func TestRemainingCoverageKustomizationsAndSSH(t *testing.T) {
	setupFluxhandlerTest(t)

	missingPath := filepath.Join(t.TempDir(), "missing")
	if err := createOrUpdateKustomizationYaml(missingPath); err == nil {
		t.Fatalf("expected createOrUpdateKustomizationYaml readdir error")
	}

	kustomDir := t.TempDir()
	kustomPath := filepath.Join(kustomDir, "kustomization.yaml")
	if err := os.WriteFile(kustomPath, []byte("resources:\n"), 0o000); err != nil {
		t.Fatalf("write kustomization: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(kustomPath, 0o644) })
	if err := createOrUpdateKustomizationYaml(kustomDir); err == nil {
		t.Fatalf("expected read error for unreadable kustomization.yaml")
	}

	resDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(resDir, "note.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write note: %v", err)
	}
	files, err := os.ReadDir(resDir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	var infos []os.FileInfo
	for _, entry := range files {
		info, infoErr := entry.Info()
		if infoErr != nil {
			t.Fatalf("entry info: %v", infoErr)
		}
		infos = append(infos, info)
	}
	if got := collectKustomizationResources(resDir, infos, ""); len(got) != 0 {
		t.Fatalf("expected no resources, got %v", got)
	}

	if err := ProcessDirectory(filepath.Join(t.TempDir(), "does-not-exist")); err == nil {
		t.Fatalf("expected ProcessDirectory read error")
	}

	hasKs := t.TempDir()
	if err := os.WriteFile(filepath.Join(hasKs, "ks.yaml"), []byte("kind: Kustomization"), 0o644); err != nil {
		t.Fatalf("write ks.yaml: %v", err)
	}
	if err := ProcessDirectory(hasKs); err != nil {
		t.Fatalf("expected ProcessDirectory success with existing ks.yaml: %v", err)
	}

	ksErrDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(ksErrDir, "app"), 0o755); err != nil {
		t.Fatalf("mkdir app: %v", err)
	}
	if err := os.Chmod(ksErrDir, 0o500); err != nil {
		t.Fatalf("chmod ksErrDir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(ksErrDir, 0o700) })
	if err := ProcessDirectory(ksErrDir); err == nil {
		t.Fatalf("expected ProcessDirectory createKsYaml error")
	}

	kuErrDir := t.TempDir()
	if err := os.Chmod(kuErrDir, 0o500); err != nil {
		t.Fatalf("chmod kuErrDir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(kuErrDir, 0o700) })
	if err := ProcessDirectory(kuErrDir); err == nil {
		t.Fatalf("expected ProcessDirectory createOrUpdate error")
	}

	recurseErrParent := t.TempDir()
	recurseErrChild := filepath.Join(recurseErrParent, "child")
	if err := os.MkdirAll(recurseErrChild, 0o755); err != nil {
		t.Fatalf("mkdir child: %v", err)
	}
	if err := os.Chmod(recurseErrChild, 0o000); err != nil {
		t.Fatalf("chmod recurse child: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(recurseErrChild, 0o700) })
	didPanic := false
	func() {
		defer func() {
			if recover() != nil {
				didPanic = true
			}
		}()
		_ = ProcessDirectory(recurseErrParent)
	}()
	if !didPanic {
		t.Fatalf("expected panic from ProcessDirectory recursive child path")
	}

	sshStatFn = func(string) (os.FileInfo, error) { return nil, nil }
	if err := CreateGitSecret("git.local"); err != nil {
		t.Fatalf("expected CreateGitSecret no-op success: %v", err)
	}

	sshGenerateKeyFn = defaultSSHGenerateKeyFn
	sshPemBlockForKeyFn = defaultSSHPemBlockForKeyFn
	sshPublicKeyToOpenSSHFn = defaultSSHPublicKeyToOpenSSHFn
	sshWriteFileFn = func(file string, _ []byte, _ os.FileMode) error {
		if strings.Contains(file, "pub") {
			return errors.New("write pub")
		}
		return nil
	}
	if err := createNewGitSecretFiles("git.local", "secret.yaml", "pub.key"); err == nil {
		t.Fatalf("expected createNewGitSecretFiles public-key write error")
	}

	sshWriteFileFn = func(string, []byte, os.FileMode) error { return nil }
	sshBuildGitSecretYAMLFn = func(string, string, string) ([]byte, error) { return nil, errors.New("yaml") }
	if err := createNewGitSecretFiles("git.local", "secret.yaml", "pub.key"); err == nil {
		t.Fatalf("expected createNewGitSecretFiles build yaml error")
	}

	sshBuildGitSecretYAMLFn = func(string, string, string) ([]byte, error) { return []byte("x"), nil }
	sshMkdirAllFn = func(string, os.FileMode) error { return errors.New("mkdir") }
	if err := createNewGitSecretFiles("git.local", "secret.yaml", "pub.key"); err == nil {
		t.Fatalf("expected createNewGitSecretFiles mkdir error")
	}

	sshMkdirAllFn = func(string, os.FileMode) error { return nil }
	sshWriteFileFn = func(file string, _ []byte, _ os.FileMode) error {
		if strings.Contains(file, "secret") {
			return errors.New("write secret")
		}
		return nil
	}
	if err := createNewGitSecretFiles("git.local", "secret.yaml", "pub.key"); err == nil {
		t.Fatalf("expected createNewGitSecretFiles secret write error")
	}

	sshYAMLMarshalFn = func(any) ([]byte, error) { return nil, errors.New("marshal") }
	if _, err := buildGitSecretYAML("id", "pub", "known"); err == nil {
		t.Fatalf("expected buildGitSecretYAML marshal error")
	}

	sshReadFileFn = func(string) ([]byte, error) { return nil, errors.New("read") }
	if err := writePublicKeyFromExistingSecret("secret.yaml", "pub.key"); err == nil {
		t.Fatalf("expected writePublicKeyFromExistingSecret read error")
	}
	sshReadFileFn = func(string) ([]byte, error) { return []byte("stringData:"), nil }
	sshYAMLUnmarshalFn = func([]byte, any) error { return errors.New("unmarshal") }
	if err := writePublicKeyFromExistingSecret("secret.yaml", "pub.key"); err == nil {
		t.Fatalf("expected writePublicKeyFromExistingSecret unmarshal error")
	}
	sshYAMLUnmarshalFn = func([]byte, any) error { return nil }
	if err := writePublicKeyFromExistingSecret("secret.yaml", "pub.key"); err == nil {
		t.Fatalf("expected writePublicKeyFromExistingSecret missing key error")
	}
	sshYAMLUnmarshalFn = func(_ []byte, out any) error {
		secret := out.(*corev1.Secret)
		secret.StringData = map[string]string{"identity.pub": "pub"}
		return nil
	}
	sshWriteFileFn = func(string, []byte, os.FileMode) error { return errors.New("write") }
	if err := writePublicKeyFromExistingSecret("secret.yaml", "pub.key"); err == nil {
		t.Fatalf("expected writePublicKeyFromExistingSecret write error")
	}
	sshWriteFileFn = func(string, []byte, os.FileMode) error { return nil }
	if err := writePublicKeyFromExistingSecret("secret.yaml", "pub.key"); err != nil {
		t.Fatalf("expected writePublicKeyFromExistingSecret success: %v", err)
	}

	if _, err := pemBlockForKey(&ecdsa.PrivateKey{}); err == nil {
		t.Fatalf("expected pemBlockForKey error for invalid key")
	}
	if _, err := publicKeyToOpenSSH(&ecdsa.PublicKey{}); err == nil {
		t.Fatalf("expected publicKeyToOpenSSH error for invalid key")
	}
}

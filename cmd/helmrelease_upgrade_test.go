package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/fluxhandler"
)

func TestRunHelmreleaseUpgradeCallsDependencies(t *testing.T) {
	oldLoadEnv := helmreleaseUpgradeLoadTalEnv
	oldLoadRepos := helmreleaseUpgradeLoadRepos
	oldUpgrade := helmreleaseUpgradeCharts
	t.Cleanup(func() {
		helmreleaseUpgradeLoadTalEnv = oldLoadEnv
		helmreleaseUpgradeLoadRepos = oldLoadRepos
		helmreleaseUpgradeCharts = oldUpgrade
	})

	loaded := false
	reposLoadedPath := ""
	upgraded := false
	var gotCharts []fluxhandler.HelmChart
	helmreleaseUpgradeLoadTalEnv = func(bool) error { loaded = true; return nil }
	helmreleaseUpgradeLoadRepos = func(path string) (map[string]*fluxhandler.HelmRepo, error) {
		reposLoadedPath = path
		return map[string]*fluxhandler.HelmRepo{}, nil
	}
	helmreleaseUpgradeCharts = func(charts []fluxhandler.HelmChart, _ map[string]*fluxhandler.HelmRepo, nowait bool) {
		upgraded = true
		gotCharts = charts
		if nowait {
			t.Fatalf("expected nowait=false")
		}
	}

	runHelmreleaseUpgrade("./clusters/main/kubernetes/apps/demo/app/helm-release.yaml")

	if !loaded {
		t.Fatalf("expected tal env loader to be called")
	}
	if reposLoadedPath != filepath.Join("./repositories", "helm") {
		t.Fatalf("unexpected repos path: %q", reposLoadedPath)
	}
	if !upgraded || len(gotCharts) != 1 {
		t.Fatalf("expected one chart upgrade call")
	}
	if gotCharts[0].ChartPath != filepath.Clean("./clusters/main/kubernetes/apps/demo/app") {
		t.Fatalf("unexpected chart dir: %q", gotCharts[0].ChartPath)
	}
}

func TestHelmreleaseUpgradeCommandRunCallsHelperPath(t *testing.T) {
	oldLoadEnv := helmreleaseUpgradeLoadTalEnv
	oldLoadRepos := helmreleaseUpgradeLoadRepos
	oldUpgrade := helmreleaseUpgradeCharts
	t.Cleanup(func() {
		helmreleaseUpgradeLoadTalEnv = oldLoadEnv
		helmreleaseUpgradeLoadRepos = oldLoadRepos
		helmreleaseUpgradeCharts = oldUpgrade
	})

	helmreleaseUpgradeLoadTalEnv = func(bool) error { return nil }
	helmreleaseUpgradeLoadRepos = func(string) (map[string]*fluxhandler.HelmRepo, error) {
		return map[string]*fluxhandler.HelmRepo{}, nil
	}
	called := false
	helmreleaseUpgradeCharts = func(charts []fluxhandler.HelmChart, _ map[string]*fluxhandler.HelmRepo, _ bool) {
		called = true
		if len(charts) != 1 {
			t.Fatalf("expected one chart")
		}
	}

	hrupgrade.Run(hrupgrade, []string{"./clusters/main/kubernetes/apps/demo/app/helm-release.yaml"})

	if !called {
		t.Fatalf("expected upgrade charts call")
	}
}

func TestHelperHelmreleaseUpgradeExitProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HR_UPGRADE_EXIT_HELPER") != "1" {
		return
	}
	hrupgrade.Run(hrupgrade, []string{"./clusters/main/kubernetes/apps/demo/app/helm-release.yaml"})
	os.Exit(0)
}

func TestHelmreleaseUpgradeRunExitsNonZero(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperHelmreleaseUpgradeExitProcess")
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "GO_WANT_HR_UPGRADE_EXIT_HELPER=1")
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected non-zero exit for hr-upgrade")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() == 0 {
		t.Fatalf("expected non-zero exit code for hr-upgrade, got %v", err)
	}
}

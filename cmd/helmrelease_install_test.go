package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/fluxhandler"
)

func TestResolveHelmReleaseDir(t *testing.T) {
	if got := resolveHelmReleaseDir("./clusters/main/app/helm-release.yaml"); got != filepath.Clean("./clusters/main/app") {
		t.Fatalf("unexpected resolved dir: %q", got)
	}
	if got := resolveHelmReleaseDir("./clusters/main/app"); got != filepath.Clean("./clusters/main") {
		t.Fatalf("unexpected resolved dir for directory-like input: %q", got)
	}
	if got := resolveHelmReleaseDir("/"); got != "/" {
		t.Fatalf("unexpected resolved dir for root path: %q", got)
	}
}

func TestRunHelmreleaseInstallCallsDependencies(t *testing.T) {
	oldLoadEnv := helmreleaseInstallLoadTalEnv
	oldLoadRepos := helmreleaseInstallLoadRepos
	oldInstall := helmreleaseInstallCharts
	t.Cleanup(func() {
		helmreleaseInstallLoadTalEnv = oldLoadEnv
		helmreleaseInstallLoadRepos = oldLoadRepos
		helmreleaseInstallCharts = oldInstall
	})

	loaded := false
	reposLoadedPath := ""
	installed := false
	var gotCharts []fluxhandler.HelmChart
	helmreleaseInstallLoadTalEnv = func(bool) error { loaded = true; return nil }
	helmreleaseInstallLoadRepos = func(path string) (map[string]*fluxhandler.HelmRepo, error) {
		reposLoadedPath = path
		return map[string]*fluxhandler.HelmRepo{}, nil
	}
	helmreleaseInstallCharts = func(charts []fluxhandler.HelmChart, _ map[string]*fluxhandler.HelmRepo, nowait bool) {
		installed = true
		gotCharts = charts
		if nowait {
			t.Fatalf("expected nowait=false")
		}
	}

	runHelmreleaseInstall("./clusters/main/kubernetes/apps/demo/app/helm-release.yaml")

	if !loaded {
		t.Fatalf("expected tal env loader to be called")
	}
	if reposLoadedPath != filepath.Join("./repositories", "helm") {
		t.Fatalf("unexpected repos path: %q", reposLoadedPath)
	}
	if !installed || len(gotCharts) != 1 {
		t.Fatalf("expected one chart install call")
	}
	if gotCharts[0].ChartPath != filepath.Clean("./clusters/main/kubernetes/apps/demo/app") {
		t.Fatalf("unexpected chart dir: %q", gotCharts[0].ChartPath)
	}
}

func TestHelmreleaseInstallCommandRunCallsHelperPath(t *testing.T) {
	oldLoadEnv := helmreleaseInstallLoadTalEnv
	oldLoadRepos := helmreleaseInstallLoadRepos
	oldInstall := helmreleaseInstallCharts
	t.Cleanup(func() {
		helmreleaseInstallLoadTalEnv = oldLoadEnv
		helmreleaseInstallLoadRepos = oldLoadRepos
		helmreleaseInstallCharts = oldInstall
	})

	helmreleaseInstallLoadTalEnv = func(bool) error { return nil }
	helmreleaseInstallLoadRepos = func(string) (map[string]*fluxhandler.HelmRepo, error) {
		return map[string]*fluxhandler.HelmRepo{}, nil
	}
	called := false
	helmreleaseInstallCharts = func(charts []fluxhandler.HelmChart, _ map[string]*fluxhandler.HelmRepo, _ bool) {
		called = true
		if len(charts) != 1 {
			t.Fatalf("expected one chart")
		}
	}

	hrinstall.Run(hrinstall, []string{"./clusters/main/kubernetes/apps/demo/app/helm-release.yaml"})

	if !called {
		t.Fatalf("expected install charts call")
	}
}

func TestHelperHelmreleaseInstallExitProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HR_INSTALL_EXIT_HELPER") != "1" {
		return
	}
	hrinstall.Run(hrinstall, []string{"./clusters/main/kubernetes/apps/demo/app/helm-release.yaml"})
	os.Exit(0)
}

func TestHelmreleaseInstallRunExitsNonZero(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperHelmreleaseInstallExitProcess")
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "GO_WANT_HR_INSTALL_EXIT_HELPER=1")
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected non-zero exit for hr-install")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() == 0 {
		t.Fatalf("expected non-zero exit code for hr-install, got %v", err)
	}
}

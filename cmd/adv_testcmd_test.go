package cmd

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

func TestAdvTestManifestPathsUsesKubernetesPath(t *testing.T) {
	oldPath := helper.KubernetesPath
	t.Cleanup(func() { helper.KubernetesPath = oldPath })

	helper.KubernetesPath = filepath.Join("root", "k8s")

	paths := advTestManifestPaths()
	if len(paths) != 3 {
		t.Fatalf("expected 3 manifest paths, got %d", len(paths))
	}
	if !strings.Contains(paths[0], filepath.Join("root", "k8s", "flux-system", "flux", "sopssecret.secret.yaml")) {
		t.Fatalf("unexpected first manifest path: %s", paths[0])
	}
}

func TestRunAdvTestCommandAppliesAllManifests(t *testing.T) {
	oldApply := advTestKubectlApply
	t.Cleanup(func() { advTestKubectlApply = oldApply })

	manifestPaths := []string{"a.yaml", "b.yaml", "c.yaml"}
	var called []string
	advTestKubectlApply = func(_ context.Context, filePath string) error {
		called = append(called, filePath)
		return nil
	}

	if err := runAdvTestCommand(context.Background(), manifestPaths); err != nil {
		t.Fatalf("runAdvTestCommand returned unexpected error: %v", err)
	}

	if len(called) != len(manifestPaths) {
		t.Fatalf("expected %d apply calls, got %d", len(manifestPaths), len(called))
	}
	for i := range manifestPaths {
		if called[i] != manifestPaths[i] {
			t.Fatalf("expected call %d to %q, got %q", i, manifestPaths[i], called[i])
		}
	}
}

func TestRunAdvTestCommandReturnsWrappedError(t *testing.T) {
	oldApply := advTestKubectlApply
	t.Cleanup(func() { advTestKubectlApply = oldApply })

	advTestKubectlApply = func(_ context.Context, filePath string) error {
		if filePath == "fail.yaml" {
			return errors.New("apply failed")
		}
		return nil
	}

	err := runAdvTestCommand(context.Background(), []string{"ok.yaml", "fail.yaml"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "fail.yaml") {
		t.Fatalf("expected wrapped error to include failing file, got: %v", err)
	}
}

func TestHelperAdvTestExitProcess(t *testing.T) {
	if os.Getenv("GO_WANT_ADV_TEST_EXIT_HELPER") != "1" {
		return
	}
	testcmd.Run(testcmd, nil)
	os.Exit(0)
}

func TestAdvTestRunExitsNonZero(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperAdvTestExitProcess")
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "GO_WANT_ADV_TEST_EXIT_HELPER=1")
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected non-zero exit for adv-test")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() == 0 {
		t.Fatalf("expected non-zero exit code for adv-test, got %v", err)
	}
}

func TestAdvTestRunSuccessPath(t *testing.T) {
	oldLoadEnv := advTestLoadTalEnv
	oldLoadConfig := advTestLoadTalConfig
	oldApply := advTestKubectlApply
	oldExit := advTestExit
	t.Cleanup(func() {
		advTestLoadTalEnv = oldLoadEnv
		advTestLoadTalConfig = oldLoadConfig
		advTestKubectlApply = oldApply
		advTestExit = oldExit
	})

	loadedEnv := false
	loadedCfg := false
	exitCalled := false
	advTestLoadTalEnv = func() { loadedEnv = true }
	advTestLoadTalConfig = func() { loadedCfg = true }
	advTestKubectlApply = func(_ context.Context, _ string) error { return nil }
	advTestExit = func(int) { exitCalled = true }

	testcmd.Run(testcmd, nil)

	if !loadedEnv || !loadedCfg {
		t.Fatalf("expected tal env and tal config to be loaded")
	}
	if exitCalled {
		t.Fatalf("did not expect exit on success path")
	}
}

func TestAdvTestRunFailurePathCallsExit(t *testing.T) {
	oldLoadEnv := advTestLoadTalEnv
	oldLoadConfig := advTestLoadTalConfig
	oldApply := advTestKubectlApply
	oldExit := advTestExit
	t.Cleanup(func() {
		advTestLoadTalEnv = oldLoadEnv
		advTestLoadTalConfig = oldLoadConfig
		advTestKubectlApply = oldApply
		advTestExit = oldExit
	})

	advTestLoadTalEnv = func() {}
	advTestLoadTalConfig = func() {}
	advTestKubectlApply = func(_ context.Context, _ string) error { return errors.New("boom") }
	exitCode := 0
	advTestExit = func(code int) { exitCode = code }

	testcmd.Run(testcmd, nil)

	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
}

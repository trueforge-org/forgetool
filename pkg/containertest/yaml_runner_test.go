package containertest

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setYAMLRunnerSeams(t *testing.T) {
	t.Helper()
	oldLoad := loadContainerTestYAMLFn
	oldWaits := checkWaitsFn
	oldFiles := checkFilesExistFn
	oldCommands := checkCommandsFn
	oldStandardRun := checkStandardRunFn
	t.Cleanup(func() {
		loadContainerTestYAMLFn = oldLoad
		checkWaitsFn = oldWaits
		checkFilesExistFn = oldFiles
		checkCommandsFn = oldCommands
		checkStandardRunFn = oldStandardRun
	})
}

func TestLoadContainerTestYAML(t *testing.T) {
	if _, err := LoadContainerTestYAML("does-not-exist.yaml"); err == nil {
		t.Fatalf("expected file read error")
	}

	dir := t.TempDir()
	badPath := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(badPath, []byte("http: ["), 0o600); err != nil {
		t.Fatalf("failed writing bad yaml: %v", err)
	}
	if _, err := LoadContainerTestYAML(badPath); err == nil {
		t.Fatalf("expected parse error")
	}

	goodPath := filepath.Join(dir, "good.yaml")
	content := "timeoutSeconds: 130\nstandardRun: true\nreadOnlyRootfs: true\nhttp:\n  - port: \"8080\"\n    path: /health\n"
	if err := os.WriteFile(goodPath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed writing good yaml: %v", err)
	}
	config, err := LoadContainerTestYAML(goodPath)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if config.TimeoutSeconds != 130 || !config.StandardRun || len(config.HTTP) != 1 || config.HTTP[0].Port != "8080" {
		t.Fatalf("unexpected config: %+v", config)
	}
	if !config.ReadOnlyRootfs {
		t.Fatalf("expected ReadOnlyRootfs=true, got false")
	}
}

func TestRunChecksFromYAMLValidationAndErrors(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{}, errors.New("load boom")
	}
	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err == nil {
		t.Fatalf("expected load error")
	}

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{}, nil
	}
	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err == nil {
		t.Fatalf("expected empty checks error")
	}

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{StandardRun: true}, nil
	}
	checkStandardRunFn = func(context.Context, string, *ContainerConfig) error {
		return errors.New("standard run boom")
	}
	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err == nil {
		t.Fatalf("expected standard run error")
	}

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{Commands: []CommandTestConfig{{Command: " "}}}, nil
	}
	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err == nil || !strings.Contains(err.Error(), "commands[0].command") {
		t.Fatalf("expected command validation error, got %v", err)
	}

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{FilePaths: []string{" "}}, nil
	}
	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err == nil || !strings.Contains(err.Error(), "filePaths[0]") {
		t.Fatalf("expected file path validation error, got %v", err)
	}

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{HTTP: []HTTPTestConfig{{Port: "8080"}}}, nil
	}
	checkWaitsFn = func(context.Context, string, []HTTPTestConfig, []TCPTestConfig, *ContainerConfig) error {
		return errors.New("wait boom")
	}
	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err == nil {
		t.Fatalf("expected waits error")
	}

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{FilePaths: []string{"/x"}}, nil
	}
	checkWaitsFn = CheckWaits
	checkFilesExistFn = func(context.Context, string, []string, *ContainerConfig) error {
		return errors.New("files boom")
	}
	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err == nil {
		t.Fatalf("expected files error")
	}

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{Commands: []CommandTestConfig{{Command: "echo ok"}}}, nil
	}
	checkFilesExistFn = CheckFilesExist
	checkCommandsFn = func(context.Context, string, *ContainerConfig, []CommandTestConfig) error {
		return errors.New("commands boom")
	}
	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err == nil {
		t.Fatalf("expected commands error")
	}
}

func TestRunChecksFromYAMLCallsAllCheckTypes(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)

	calledWaits := 0
	calledFiles := 0
	calledCommands := 0
	calledStandardRun := 0

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			TimeoutSeconds: 1,
			HTTP:           []HTTPTestConfig{{Port: "8080"}},
			TCP:            []TCPTestConfig{{Port: "9090"}},
			FilePaths:      []string{"/etc/hosts"},
			Commands:       []CommandTestConfig{{Command: "echo ok"}},
			StandardRun:    true,
		}, nil
	}

	checkWaitsFn = func(waitCtx context.Context, image string, http []HTTPTestConfig, tcp []TCPTestConfig, cfg *ContainerConfig) error {
		calledWaits++
		if image != "img" || len(http) != 1 || len(tcp) != 1 {
			t.Fatalf("unexpected waits args")
		}
		deadline, ok := waitCtx.Deadline()
		if !ok {
			t.Fatalf("expected timeout deadline")
		}
		if time.Until(deadline) < 119*time.Second {
			t.Fatalf("expected minimum timeout floor to apply")
		}
		return nil
	}
	checkFilesExistFn = func(context.Context, string, []string, *ContainerConfig) error {
		calledFiles++
		return nil
	}
	checkCommandsFn = func(context.Context, string, *ContainerConfig, []CommandTestConfig) error {
		calledCommands++
		return nil
	}
	checkStandardRunFn = func(context.Context, string, *ContainerConfig) error {
		calledStandardRun++
		return nil
	}

	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", &ContainerConfig{Env: map[string]string{"A": "1"}}); err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}

	if calledWaits != 1 || calledFiles != 1 || calledCommands != 1 || calledStandardRun != 1 {
		t.Fatalf("expected all checks once, got waits=%d files=%d commands=%d standardRun=%d", calledWaits, calledFiles, calledCommands, calledStandardRun)
	}
}

func TestRunChecksFromYAMLReadOnlyRootfs(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)

	var gotConfig *ContainerConfig

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			StandardRun:    true,
			ReadOnlyRootfs: true,
		}, nil
	}
	checkStandardRunFn = func(_ context.Context, _ string, cfg *ContainerConfig) error {
		gotConfig = cfg
		return nil
	}

	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotConfig == nil || !gotConfig.ReadOnlyRootfs {
		t.Fatalf("expected ReadOnlyRootfs=true to be propagated to ContainerConfig, got %+v", gotConfig)
	}
}

func TestRunChecksFromYAMLReadOnlyRootfsMergesExistingConfig(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)

	var gotConfig *ContainerConfig

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			StandardRun:    true,
			ReadOnlyRootfs: true,
		}, nil
	}
	checkStandardRunFn = func(_ context.Context, _ string, cfg *ContainerConfig) error {
		gotConfig = cfg
		return nil
	}

	existing := &ContainerConfig{Env: map[string]string{"FOO": "bar"}}
	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", existing); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotConfig == nil || !gotConfig.ReadOnlyRootfs {
		t.Fatalf("expected ReadOnlyRootfs=true in merged config, got %+v", gotConfig)
	}
	if gotConfig.Env["FOO"] != "bar" {
		t.Fatalf("expected existing env to be preserved, got %+v", gotConfig.Env)
	}
}

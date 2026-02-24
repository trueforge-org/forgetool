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

func intPtr(value int) *int {
	return &value
}

func setYAMLRunnerSeams(t *testing.T) {
	t.Helper()
	oldLoad := loadContainerTestYAMLFn
	oldHealth := checkHealthFn
	oldWaits := checkWaitsFn
	oldHealthAndWaits := checkHealthAndWaitsFn
	oldHealthCommands := checkHealthCommandsFn
	oldStandardRun := checkStandardRunFn
	oldRunnerOutput := checkRunnerOutputFn
	t.Cleanup(func() {
		loadContainerTestYAMLFn = oldLoad
		checkHealthFn = oldHealth
		checkWaitsFn = oldWaits
		checkHealthAndWaitsFn = oldHealthAndWaits
		checkHealthCommandsFn = oldHealthCommands
		checkStandardRunFn = oldStandardRun
		checkRunnerOutputFn = oldRunnerOutput
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
	content := "readOnlyRootfs: true\nhttp:\n  - port: \"8080\"\n    path: /health\n"
	if err := os.WriteFile(goodPath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed writing good yaml: %v", err)
	}
	config, err := LoadContainerTestYAML(goodPath)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if len(config.HTTP) != 1 || config.HTTP[0].Port != "8080" {
		t.Fatalf("unexpected config: %+v", config)
	}

	if !config.ReadOnlyRootfs {
		t.Fatalf("expected ReadOnlyRootfs=true, got false")
	}

	mountsPath := filepath.Join(dir, "mounts.yaml")
	mountsContent := "mounts:\n  - path: /config\n    chmod: \"755\"\n    chown: \"568:568\"\n  - path: /data\n"
	if err := os.WriteFile(mountsPath, []byte(mountsContent), 0o600); err != nil {
		t.Fatalf("failed writing mounts yaml: %v", err)
	}
	mountsConfig, err := LoadContainerTestYAML(mountsPath)
	if err != nil {
		t.Fatalf("unexpected mounts load error: %v", err)
	}
	if len(mountsConfig.Mounts) != 2 {
		t.Fatalf("expected 2 mounts, got %d", len(mountsConfig.Mounts))
	}
	if mountsConfig.Mounts[0].Path != "/config" || mountsConfig.Mounts[0].Chmod != "755" || mountsConfig.Mounts[0].Chown != "568:568" {
		t.Fatalf("unexpected first mount: %+v", mountsConfig.Mounts[0])
	}
	if mountsConfig.Mounts[1].Path != "/data" {
		t.Fatalf("unexpected second mount: %+v", mountsConfig.Mounts[1])
	}

	// Runners section
	runnersPath := filepath.Join(dir, "runners.yaml")
	runnersContent := "timeoutSeconds: 240\nmainRunner:\n  env:\n    MAIN: runner\n  entrypoint: ignored-main\n  cmd: --ignored-main\n  expectedOutput: ignored\n  exitCode: 99\nrunners:\n  - env:\n      FOO: bar\n    entrypoint: myapp\n    cmd: --version\n    expectedOutput: \"v1.0\"\n    exitCode: 7\n  - {}\nhealthCommands:\n  - command: mycommand\n    expectedExitCode: 7\n    expectedContent: ok\n    matchContent: true\n"
	if err := os.WriteFile(runnersPath, []byte(runnersContent), 0o600); err != nil {
		t.Fatalf("failed writing runners yaml: %v", err)
	}
	runnersConfig, err := LoadContainerTestYAML(runnersPath)
	if err != nil {
		t.Fatalf("unexpected runners load error: %v", err)
	}
	if len(runnersConfig.Runners) != 2 {
		t.Fatalf("expected 2 runners, got %d", len(runnersConfig.Runners))
	}
	r0 := runnersConfig.Runners[0]
	if r0.Env["FOO"] != "bar" || r0.Entrypoint != "myapp" || r0.Cmd != "--version" || r0.ExpectedOutput != "v1.0" {
		t.Fatalf("unexpected runner[0]: %+v", r0)
	}
	if runnersConfig.TimeoutSeconds != 240 {
		t.Fatalf("expected timeoutSeconds=240, got %d", runnersConfig.TimeoutSeconds)
	}
	if r0.ExitCode == nil || *r0.ExitCode != 7 {
		t.Fatalf("expected runner[0].ExitCode=7, got %v", r0.ExitCode)
	}
	r1 := runnersConfig.Runners[1]
	if r1.ExitCode != nil {
		t.Fatalf("expected runner[1].ExitCode=nil (default), got %v", r1.ExitCode)
	}
	if runnersConfig.MainRunner == nil {
		t.Fatalf("expected mainRunner to be present")
	}
	if runnersConfig.MainRunner.Env["MAIN"] != "runner" {
		t.Fatalf("unexpected mainRunner: %+v", runnersConfig.MainRunner)
	}
	if len(runnersConfig.HealthCommands) != 1 {
		t.Fatalf("expected 1 health command, got %d", len(runnersConfig.HealthCommands))
	}
	h0 := runnersConfig.HealthCommands[0]
	if h0.Command != "mycommand" || h0.ExpectedExitCode == nil || *h0.ExpectedExitCode != 7 || h0.ExpectedContent != "ok" || !h0.MatchContent {
		t.Fatalf("unexpected health command config: %+v", h0)
	}
}

func TestRunChecksFromYAMLValidationAndErrors(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)
	checkHealthFn = func(context.Context, string, *ContainerConfig) error { return nil }
	checkHealthAndWaitsFn = func(context.Context, string, []HTTPTestConfig, []TCPTestConfig, []string, *ContainerConfig) error {
		return nil
	}

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{}, errors.New("load boom")
	}
	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err == nil {
		t.Fatalf("expected load error")
	}

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{}, nil
	}
	standardRunCalls := 0
	checkStandardRunFn = func(context.Context, string, *ContainerConfig) error {
		standardRunCalls++
		return nil
	}
	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err != nil {
		t.Fatalf("expected success when no checks are configured, got %v", err)
	}
	if standardRunCalls != 0 {
		t.Fatalf("expected no standard runs when no runners are configured, got %d", standardRunCalls)
	}

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{Runners: []RunnerConfig{{}}}, nil
	}
	checkStandardRunFn = func(context.Context, string, *ContainerConfig) error {
		return errors.New("standard run boom")
	}
	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err == nil {
		t.Fatalf("expected standard run error")
	}

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{HTTP: []HTTPTestConfig{{Port: "8080"}}}, nil
	}
	checkHealthAndWaitsFn = func(context.Context, string, []HTTPTestConfig, []TCPTestConfig, []string, *ContainerConfig) error {
		return errors.New("wait boom")
	}
	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err == nil {
		t.Fatalf("expected waits error")
	}

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{HTTP: []HTTPTestConfig{{Port: "8080"}}}, nil
	}
	checkHealthAndWaitsFn = func(context.Context, string, []HTTPTestConfig, []TCPTestConfig, []string, *ContainerConfig) error {
		return errors.New("health boom")
	}
	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err == nil {
		t.Fatalf("expected health error")
	}
	checkHealthAndWaitsFn = func(context.Context, string, []HTTPTestConfig, []TCPTestConfig, []string, *ContainerConfig) error {
		return nil
	}

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{FilePaths: []string{"/x"}}, nil
	}
	checkHealthAndWaitsFn = func(context.Context, string, []HTTPTestConfig, []TCPTestConfig, []string, *ContainerConfig) error {
		return errors.New("files boom")
	}
	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err == nil {
		t.Fatalf("expected files error")
	}

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{HealthCommands: []HealthCommandTestConfig{{Command: " "}}}, nil
	}
	checkHealthAndWaitsFn = func(context.Context, string, []HTTPTestConfig, []TCPTestConfig, []string, *ContainerConfig) error {
		t.Fatalf("did not expect wait checks")
		return nil
	}
	checkHealthCommandsFn = func(context.Context, string, *ContainerConfig, []HealthCommandTestConfig) error {
		return errors.New("health commands boom")
	}
	checkStandardRunFn = func(context.Context, string, *ContainerConfig) error { return nil }
	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err == nil || !strings.Contains(err.Error(), "health commands boom") {
		t.Fatalf("expected healthCommands to run on waits runner, got %v", err)
	}
}

func TestRunChecksFromYAMLFileWaitsAlsoRunHealthCheckWaitOnMainRunner(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)

	healthCalls := 0
	combinedCalls := 0

	checkHealthFn = func(context.Context, string, *ContainerConfig) error {
		healthCalls++
		return nil
	}
	checkHealthAndWaitsFn = func(context.Context, string, []HTTPTestConfig, []TCPTestConfig, []string, *ContainerConfig) error {
		combinedCalls++
		return nil
	}
	checkStandardRunFn = func(context.Context, string, *ContainerConfig) error { return nil }

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			FilePaths: []string{"/etc/hosts"},
		}, nil
	}

	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if healthCalls != 1 {
		t.Fatalf("expected health wait to run once on main runner, got %d", healthCalls)
	}
	if combinedCalls != 1 {
		t.Fatalf("expected combined waits to run once on main runner, got %d", combinedCalls)
	}
}

func TestRunChecksFromYAMLFilePathsValidationRejectsBlank(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{FilePaths: []string{"   "}}, nil
	}

	err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil)
	if err == nil || !strings.Contains(err.Error(), "filePaths[0] must not be empty") {
		t.Fatalf("expected blank filePath validation error, got %v", err)
	}
}

func TestRunChecksFromYAMLFileWaitsHealthCheckError(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)

	checkHealthFn = func(context.Context, string, *ContainerConfig) error {
		return errors.New("health wait boom")
	}
	checkHealthAndWaitsFn = func(context.Context, string, []HTTPTestConfig, []TCPTestConfig, []string, *ContainerConfig) error {
		t.Fatalf("did not expect combined waits when health wait already failed")
		return nil
	}
	checkStandardRunFn = func(context.Context, string, *ContainerConfig) error {
		t.Fatalf("did not expect standard run when health wait already failed")
		return nil
	}

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{FilePaths: []string{"/etc/hosts"}}, nil
	}

	err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil)
	if err == nil || !strings.Contains(err.Error(), "health wait boom") {
		t.Fatalf("expected health wait error, got %v", err)
	}
}

func TestRunChecksFromYAMLMainRunnerSpecAppliedAndOutputFieldsIgnored(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)

	var waitCfg *ContainerConfig
	checkHealthFn = func(context.Context, string, *ContainerConfig) error { return nil }
	checkHealthAndWaitsFn = func(waitCtx context.Context, _ string, _ []HTTPTestConfig, _ []TCPTestConfig, _ []string, cfg *ContainerConfig) error {
		waitCfg = cfg
		deadline, ok := waitCtx.Deadline()
		if !ok {
			t.Fatalf("expected timeout deadline")
		}
		if time.Until(deadline) < 239*time.Second {
			t.Fatalf("expected mainRunner timeout floor + buffer, got remaining=%v", time.Until(deadline))
		}
		return nil
	}
	checkStandardRunFn = func(context.Context, string, *ContainerConfig) error { return nil }

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			TimeoutSeconds: 1,
			MainRunner: &RunnerConfig{
				Env:            map[string]string{"MAIN": "1"},
				Entrypoint:     "ignored-main",
				Cmd:            "--ignored-main",
				ExpectedOutput: "ignored",
				ExitCode:       intPtr(9),
			},
			FilePaths: []string{"/etc/hosts"},
		}, nil
	}

	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", &ContainerConfig{Command: []string{"base-cmd"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if waitCfg == nil {
		t.Fatalf("expected main runner config to be captured")
	}
	if waitCfg.Env["MAIN"] != "1" {
		t.Fatalf("expected mainRunner env to apply, got %+v", waitCfg)
	}
	if len(waitCfg.Command) != 2 || waitCfg.Command[0] != "ignored-main" || waitCfg.Command[1] != "--ignored-main" {
		t.Fatalf("expected mainRunner entrypoint/cmd to override wait container command, got command=%v", waitCfg.Command)
	}
}

func TestRunChecksFromYAMLMountValidation(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)
	checkHealthAndWaitsFn = func(context.Context, string, []HTTPTestConfig, []TCPTestConfig, []string, *ContainerConfig) error {
		return nil
	}

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			Mounts: []MountConfig{{Path: "  "}},
		}, nil
	}
	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err == nil || !strings.Contains(err.Error(), "mounts[0].path") {
		t.Fatalf("expected mount path validation error, got %v", err)
	}

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			Mounts: []MountConfig{{Path: "relative/path"}},
		}, nil
	}
	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err == nil || !strings.Contains(err.Error(), "must be an absolute path") {
		t.Fatalf("expected mount absolute-path validation error, got %v", err)
	}
}

func TestRunChecksFromYAMLMergesMountsIntoConfig(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)
	checkHealthAndWaitsFn = func(context.Context, string, []HTTPTestConfig, []TCPTestConfig, []string, *ContainerConfig) error {
		return nil
	}

	yamlMounts := []MountConfig{{Path: "/config", Chmod: "755"}}
	callerMounts := []MountConfig{{Path: "/data"}}

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			Mounts:  yamlMounts,
			Runners: []RunnerConfig{{}},
		}, nil
	}

	var receivedConfig *ContainerConfig
	checkStandardRunFn = func(_ context.Context, _ string, cfg *ContainerConfig) error {
		receivedConfig = cfg
		return nil
	}

	callerConfig := &ContainerConfig{Mounts: callerMounts}
	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", callerConfig); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedConfig == nil {
		t.Fatalf("expected non-nil config")
	}
	if len(receivedConfig.Mounts) != 2 {
		t.Fatalf("expected 2 mounts (caller + yaml), got %d", len(receivedConfig.Mounts))
	}
	if receivedConfig.Mounts[0].Path != "/data" || receivedConfig.Mounts[1].Path != "/config" {
		t.Fatalf("unexpected merged mounts: %+v", receivedConfig.Mounts)
	}

	// Verify original caller config was not mutated.
	if len(callerConfig.Mounts) != 1 || callerConfig.Mounts[0].Path != "/data" {
		t.Fatalf("caller config mounts were mutated: %+v", callerConfig.Mounts)
	}
}

func TestRunChecksFromYAMLCallsAllCheckTypes(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)

	callOrder := []string{}
	calledHealthAndWaits := 0
	calledStandardRun := 0

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			HTTP:           []HTTPTestConfig{{Port: "8080"}},
			TCP:            []TCPTestConfig{{Port: "9090"}},
			FilePaths:      []string{"/etc/hosts"},
			TimeoutSeconds: 1,
		}, nil
	}

	checkHealthAndWaitsFn = func(waitCtx context.Context, image string, http []HTTPTestConfig, tcp []TCPTestConfig, filePaths []string, cfg *ContainerConfig) error {
		calledHealthAndWaits++
		callOrder = append(callOrder, "health+waits")
		if image != "img" || len(http) != 1 || len(tcp) != 1 || len(filePaths) != 1 || filePaths[0] != "/etc/hosts" {
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
	checkStandardRunFn = func(context.Context, string, *ContainerConfig) error {
		calledStandardRun++
		return nil
	}

	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", &ContainerConfig{Env: map[string]string{"A": "1"}}); err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}

	if calledHealthAndWaits != 1 || calledStandardRun != 0 {
		t.Fatalf("expected only main-runner checks with no configured runners, got health+waits=%d standardRun=%d", calledHealthAndWaits, calledStandardRun)
	}

	if len(callOrder) < 1 || callOrder[0] != "health+waits" {
		t.Fatalf("expected health+waits call order, got %v", callOrder)
	}
}

func TestRunChecksFromYAMLTopLevelFilePathsUseCombinedChecks(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)
	checkHealthFn = func(context.Context, string, *ContainerConfig) error { return nil }

	combinedCalls := 0
	checkHealthAndWaitsFn = func(_ context.Context, _ string, _ []HTTPTestConfig, _ []TCPTestConfig, filePaths []string, _ *ContainerConfig) error {
		combinedCalls++
		if len(filePaths) != 2 || filePaths[0] != "/path" || filePaths[1] != "/anotherpath" {
			t.Fatalf("unexpected filePaths: %v", filePaths)
		}
		return nil
	}

	standardRunCalls := 0
	checkStandardRunFn = func(context.Context, string, *ContainerConfig) error {
		standardRunCalls++
		return nil
	}

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			FilePaths: []string{"/path", "/anotherpath"},
		}, nil
	}

	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}

	if combinedCalls != 1 {
		t.Fatalf("expected combined checks once, got %d", combinedCalls)
	}
	if standardRunCalls != 0 {
		t.Fatalf("expected no standard runs when no runners are configured, got %d", standardRunCalls)
	}
}

func TestRunChecksFromYAMLTopLevelTimeoutApplies(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)
	checkHealthFn = func(context.Context, string, *ContainerConfig) error { return nil }

	checkHealthAndWaitsFn = func(waitCtx context.Context, _ string, _ []HTTPTestConfig, _ []TCPTestConfig, filePaths []string, _ *ContainerConfig) error {
		if len(filePaths) != 1 || filePaths[0] != "/etc/hosts" {
			t.Fatalf("unexpected filePaths: %v", filePaths)
		}
		deadline, ok := waitCtx.Deadline()
		if !ok {
			t.Fatalf("expected timeout deadline")
		}
		remaining := time.Until(deadline)
		if remaining < 179*time.Second {
			t.Fatalf("expected runner timeout to be used, got remaining=%v", remaining)
		}
		return nil
	}

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			FilePaths:      []string{"/etc/hosts"},
			TimeoutSeconds: 180,
		}, nil
	}

	checkStandardRunFn = func(context.Context, string, *ContainerConfig) error { return nil }
	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}
}

func TestRunChecksFromYAMLTopLevelTimeoutIgnoresShorterParentDeadline(t *testing.T) {
	parentCtx, parentCancel := context.WithTimeout(context.Background(), time.Minute)
	defer parentCancel()
	setYAMLRunnerSeams(t)
	checkHealthFn = func(context.Context, string, *ContainerConfig) error { return nil }

	checkHealthAndWaitsFn = func(waitCtx context.Context, _ string, _ []HTTPTestConfig, _ []TCPTestConfig, filePaths []string, _ *ContainerConfig) error {
		if len(filePaths) != 1 || filePaths[0] != "/etc/hosts" {
			t.Fatalf("unexpected filePaths: %v", filePaths)
		}
		deadline, ok := waitCtx.Deadline()
		if !ok {
			t.Fatalf("expected timeout deadline")
		}
		remaining := time.Until(deadline)
		if remaining < 179*time.Second {
			t.Fatalf("expected runner timeout to override shorter parent deadline, got remaining=%v", remaining)
		}
		return nil
	}

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			FilePaths:      []string{"/etc/hosts"},
			TimeoutSeconds: 180,
		}, nil
	}

	checkStandardRunFn = func(context.Context, string, *ContainerConfig) error { return nil }
	if err := RunChecksFromYAML(parentCtx, "img", "cfg.yaml", nil); err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}
}

func TestRunChecksFromYAMLHealthTimeoutHasExtraBuffer(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			TimeoutSeconds: 180,
			HTTP:           []HTTPTestConfig{{Port: "8080"}},
		}, nil
	}

	checkHealthAndWaitsFn = func(waitCtx context.Context, _ string, _ []HTTPTestConfig, _ []TCPTestConfig, _ []string, _ *ContainerConfig) error {
		deadline, ok := waitCtx.Deadline()
		if !ok {
			t.Fatalf("expected timeout deadline for combined health+wait checks")
		}
		remaining := time.Until(deadline)
		if remaining < 239*time.Second {
			t.Fatalf("expected combined timeout to include +60s buffer, got remaining=%v", remaining)
		}
		return nil
	}

	checkStandardRunFn = func(context.Context, string, *ContainerConfig) error { return nil }
	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}
}

func TestRunChecksFromYAMLReadOnlyRootfs(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)
	checkHealthAndWaitsFn = func(context.Context, string, []HTTPTestConfig, []TCPTestConfig, []string, *ContainerConfig) error {
		return nil
	}

	var gotConfig *ContainerConfig

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			ReadOnlyRootfs: true,
			Runners:        []RunnerConfig{{}},
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
	checkHealthAndWaitsFn = func(context.Context, string, []HTTPTestConfig, []TCPTestConfig, []string, *ContainerConfig) error {
		return nil
	}

	var gotConfig *ContainerConfig

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			ReadOnlyRootfs: true,
			Runners:        []RunnerConfig{{}},
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

// --- runners feature tests ---

func TestBuildRunnerContainerConfig(t *testing.T) {
	// No base, no runner overrides
	got := buildRunnerContainerConfig(RunnerConfig{}, nil, false, nil)
	if got == nil {
		t.Fatalf("expected non-nil config")
	}
	if got.ReadOnlyRootfs || len(got.Env) > 0 || len(got.Command) > 0 || len(got.Mounts) > 0 {
		t.Fatalf("expected empty config, got %+v", got)
	}

	// YAML-level readOnlyRootfs propagates
	got = buildRunnerContainerConfig(RunnerConfig{}, nil, true, nil)
	if !got.ReadOnlyRootfs {
		t.Fatalf("expected ReadOnlyRootfs from YAML level")
	}

	// Runner env merged on top of base env; runner wins on conflict
	base := &ContainerConfig{Env: map[string]string{"A": "1", "B": "2"}}
	got = buildRunnerContainerConfig(RunnerConfig{Env: map[string]string{"B": "runner", "C": "3"}}, base, false, nil)
	if got.Env["A"] != "1" || got.Env["B"] != "runner" || got.Env["C"] != "3" {
		t.Fatalf("unexpected merged env: %+v", got.Env)
	}
	// base not mutated
	if base.Env["B"] != "2" {
		t.Fatalf("base env was mutated")
	}

	// Runner entrypoint+cmd become Command slice
	got = buildRunnerContainerConfig(RunnerConfig{Entrypoint: "myapp", Cmd: "--flag arg"}, nil, false, nil)
	if len(got.Command) != 3 || got.Command[0] != "myapp" || got.Command[1] != "--flag" || got.Command[2] != "arg" {
		t.Fatalf("unexpected command slice: %+v", got.Command)
	}

	// Empty runner entrypoint/cmd leaves base command intact
	base = &ContainerConfig{Command: []string{"existing"}}
	got = buildRunnerContainerConfig(RunnerConfig{}, base, false, nil)
	if len(got.Command) != 1 || got.Command[0] != "existing" {
		t.Fatalf("unexpected command: %+v", got.Command)
	}

	// YAML mounts appended after base mounts
	base = &ContainerConfig{Mounts: []MountConfig{{Path: "/base"}}}
	yamlMounts := []MountConfig{{Path: "/yaml"}}
	got = buildRunnerContainerConfig(RunnerConfig{}, base, false, yamlMounts)
	if len(got.Mounts) != 2 || got.Mounts[0].Path != "/base" || got.Mounts[1].Path != "/yaml" {
		t.Fatalf("unexpected mounts: %+v", got.Mounts)
	}
	// base not mutated
	if len(base.Mounts) != 1 {
		t.Fatalf("base mounts were mutated")
	}
}

func TestRunChecksFromYAMLConfiguredRunnerUsesCallerConfig(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)

	var gotConfig *ContainerConfig
	checkHealthAndWaitsFn = func(context.Context, string, []HTTPTestConfig, []TCPTestConfig, []string, *ContainerConfig) error {
		return nil
	}
	checkStandardRunFn = func(_ context.Context, _ string, cfg *ContainerConfig) error {
		gotConfig = cfg
		return nil
	}

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{Runners: []RunnerConfig{{}}}, nil
	}

	caller := &ContainerConfig{Env: map[string]string{"X": "1"}}
	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", caller); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotConfig == nil || gotConfig.Env["X"] != "1" {
		t.Fatalf("expected caller env in default runner config, got %+v", gotConfig)
	}
	// Caller not mutated
	if len(caller.Env) != 1 {
		t.Fatalf("caller config was mutated")
	}
}

func TestRunChecksFromYAMLMultipleRunnersEachRunChecks(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)

	healthCalls := 0
	standardRunCalls := 0
	var receivedEnvs []string

	checkHealthAndWaitsFn = func(_ context.Context, _ string, _ []HTTPTestConfig, _ []TCPTestConfig, _ []string, cfg *ContainerConfig) error {
		healthCalls++
		return nil
	}
	checkStandardRunFn = func(_ context.Context, _ string, cfg *ContainerConfig) error {
		standardRunCalls++
		if cfg != nil {
			receivedEnvs = append(receivedEnvs, cfg.Env["RUNNER"])
		}
		return nil
	}

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			Runners: []RunnerConfig{
				{Env: map[string]string{"RUNNER": "first"}},
				{Env: map[string]string{"RUNNER": "second"}},
			},
		}, nil
	}

	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if healthCalls != 0 {
		t.Fatalf("expected health not to run without tcp/http/healthCommands, got %d", healthCalls)
	}
	if standardRunCalls != 2 {
		t.Fatalf("expected standard run called twice for two runners, got %d", standardRunCalls)
	}
	if len(receivedEnvs) != 2 || receivedEnvs[0] != "first" || receivedEnvs[1] != "second" {
		t.Fatalf("unexpected runner envs: %v", receivedEnvs)
	}
}

func TestRunChecksFromYAMLRunnerExpectedOutputUsesRunnerOverrides(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)

	healthCalled := false
	checkHealthAndWaitsFn = func(context.Context, string, []HTTPTestConfig, []TCPTestConfig, []string, *ContainerConfig) error {
		healthCalled = true
		return nil
	}

	var gotCommand, gotExpected string
	var gotCfg *ContainerConfig
	var gotExitCode *int
	checkRunnerOutputFn = func(_ context.Context, _ string, cfg *ContainerConfig, cmd, expected string, exitCode *int) error {
		gotCfg = cfg
		gotCommand = cmd
		gotExpected = expected
		gotExitCode = exitCode
		return nil
	}
	checkStandardRunFn = func(context.Context, string, *ContainerConfig) error { return nil }

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			Runners: []RunnerConfig{
				{
					Entrypoint:     "myapp",
					Cmd:            "--version",
					ExpectedOutput: "v1.2.3",
					ExitCode:       intPtr(7),
					Env:            map[string]string{"FOO": "bar"},
				},
			},
		}, nil
	}

	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if healthCalled {
		t.Fatalf("did not expect health check when no file/http/tcp/healthCommands are configured")
	}
	if gotCommand != "myapp --version" {
		t.Fatalf("expected command=%q, got %q", "myapp --version", gotCommand)
	}
	if gotExpected != "v1.2.3" {
		t.Fatalf("expected expectedOutput=%q, got %q", "v1.2.3", gotExpected)
	}
	if gotExitCode == nil || *gotExitCode != 7 {
		t.Fatalf("expected exitCode=7, got %v", gotExitCode)
	}
	if gotCfg == nil || gotCfg.Env["FOO"] != "bar" {
		t.Fatalf("unexpected runner config: %+v", gotCfg)
	}
}

func TestRunChecksFromYAMLRunnerExpectedOutputWithCmdOnlyUsesCmd(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)

	var gotCommand string
	checkRunnerOutputFn = func(_ context.Context, _ string, _ *ContainerConfig, cmd, _ string, _ *int) error {
		gotCommand = cmd
		return nil
	}
	checkStandardRunFn = func(context.Context, string, *ContainerConfig) error { return nil }

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			Runners: []RunnerConfig{{
				Cmd:            "--version",
				ExpectedOutput: "v1.2.3",
			}},
		}, nil
	}

	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotCommand != "--version" {
		t.Fatalf("expected command to be cmd-only override, got %q", gotCommand)
	}
}

func TestRunChecksFromYAMLRunnerExpectedOutputRunsTestsByDefault(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)

	outputCalled := false
	healthCalled := false

	checkRunnerOutputFn = func(context.Context, string, *ContainerConfig, string, string, *int) error {
		outputCalled = true
		return nil
	}
	checkHealthAndWaitsFn = func(context.Context, string, []HTTPTestConfig, []TCPTestConfig, []string, *ContainerConfig) error {
		healthCalled = true
		return nil
	}
	checkStandardRunFn = func(context.Context, string, *ContainerConfig) error { return nil }

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			Runners: []RunnerConfig{
				{Entrypoint: "myapp", Cmd: "--version", ExpectedOutput: "v1.2.3"},
			},
		}, nil
	}

	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !outputCalled {
		t.Fatalf("expected output check to be called")
	}
	if healthCalled {
		t.Fatalf("did not expect health check when no file/http/tcp/healthCommands are configured")
	}
}

func TestRunChecksFromYAMLSkipsHealthWhenNoDeclarativeChecks(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)

	healthCalled := false
	checkHealthFn = func(context.Context, string, *ContainerConfig) error {
		healthCalled = true
		return nil
	}
	checkStandardRunFn = func(context.Context, string, *ContainerConfig) error { return nil }

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{}, nil
	}

	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if healthCalled {
		t.Fatalf("did not expect health check when no file/http/tcp/healthCommands are configured")
	}
}

func TestRunChecksFromYAMLRunnerOutputCheckError(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)

	checkRunnerOutputFn = func(context.Context, string, *ContainerConfig, string, string, *int) error {
		return errors.New("output mismatch")
	}

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			Runners: []RunnerConfig{
				{Entrypoint: "myapp", Cmd: "--version", ExpectedOutput: "v9.9.9"},
			},
		}, nil
	}

	err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil)
	if err == nil {
		t.Fatalf("expected error from output check")
	}
	if !strings.Contains(err.Error(), "runner[0] output check failed") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestRunChecksFromYAMLMixedRunners(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)

	outputCalled := false
	healthCallCount := 0

	checkRunnerOutputFn = func(context.Context, string, *ContainerConfig, string, string, *int) error {
		outputCalled = true
		return nil
	}
	checkHealthAndWaitsFn = func(context.Context, string, []HTTPTestConfig, []TCPTestConfig, []string, *ContainerConfig) error {
		healthCallCount++
		return nil
	}
	checkStandardRunFn = func(context.Context, string, *ContainerConfig) error { return nil }

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			Runners: []RunnerConfig{
				{Entrypoint: "myapp", Cmd: "--version", ExpectedOutput: "v1"},
				{Env: map[string]string{"MODE": "normal"}},
			},
		}, nil
	}

	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !outputCalled {
		t.Fatalf("expected output check to be called for first runner")
	}
	// No tcp/http/healthCommands are configured, so health should not run for either runner.
	if healthCallCount != 0 {
		t.Fatalf("expected health not to run when no tcp/http/healthCommands are configured, got %d", healthCallCount)
	}
}

func TestRunChecksFromYAMLOutputOnlyRunnerCountsAsCheck(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)

	checkRunnerOutputFn = func(context.Context, string, *ContainerConfig, string, string, *int) error {
		return nil
	}
	checkHealthAndWaitsFn = func(context.Context, string, []HTTPTestConfig, []TCPTestConfig, []string, *ContainerConfig) error {
		return nil
	}
	checkStandardRunFn = func(context.Context, string, *ContainerConfig) error { return nil }

	// No http/tcp/filePaths/healthCommands — only a runner with expectedOutput.
	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			Runners: []RunnerConfig{
				{Entrypoint: "myapp", Cmd: "--version", ExpectedOutput: "v1"},
			},
		}, nil
	}

	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err != nil {
		t.Fatalf("runner with expectedOutput should count as a configured check, got: %v", err)
	}
}

func TestRunChecksFromYAMLWithHTTPStillRunsStandardRun(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)

	healthCalled := false
	standardRunCalled := false

	checkHealthAndWaitsFn = func(context.Context, string, []HTTPTestConfig, []TCPTestConfig, []string, *ContainerConfig) error {
		healthCalled = true
		return nil
	}
	checkStandardRunFn = func(context.Context, string, *ContainerConfig) error {
		standardRunCalled = true
		return nil
	}

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			Runners: []RunnerConfig{{}},
			HTTP:    []HTTPTestConfig{{Port: "8080"}},
		}, nil
	}

	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !healthCalled {
		t.Fatalf("health checks should run when HTTP waits are configured")
	}
	if !standardRunCalled {
		t.Fatalf("standard run should still execute")
	}
}

func TestRunChecksFromYAMLRunnerStandardRunError(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			Runners: []RunnerConfig{{}},
		}, nil
	}

	checkStandardRunFn = func(context.Context, string, *ContainerConfig) error {
		return errors.New("standard run failed")
	}

	err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil)
	if err == nil {
		t.Fatalf("expected standard run error")
	}
	if !strings.Contains(err.Error(), "standard run failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func boolPtr(v bool) *bool {
	return &v
}

func TestRunChecksFromYAMLMainRunnerEnabledFalseSkipsHealthWaits(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)

	healthCalled := false
	healthCommandsCalled := false

	checkHealthFn = func(context.Context, string, *ContainerConfig) error {
		healthCalled = true
		return nil
	}
	checkHealthAndWaitsFn = func(context.Context, string, []HTTPTestConfig, []TCPTestConfig, []string, *ContainerConfig) error {
		healthCalled = true
		return nil
	}
	checkHealthCommandsFn = func(context.Context, string, *ContainerConfig, []HealthCommandTestConfig) error {
		healthCommandsCalled = true
		return nil
	}
	checkStandardRunFn = func(context.Context, string, *ContainerConfig) error { return nil }

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			MainRunner: &RunnerConfig{Enabled: boolPtr(false)},
			HTTP:       []HTTPTestConfig{{Port: "8080"}},
			TCP:        []TCPTestConfig{{Port: "9090"}},
			FilePaths:  []string{"/etc/hosts"},
			HealthCommands: []HealthCommandTestConfig{
				{Command: "healthcheck"},
			},
			Runners: []RunnerConfig{{}},
		}, nil
	}

	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if healthCalled {
		t.Fatalf("expected health/wait checks to be skipped when mainRunner.enabled=false")
	}
	if healthCommandsCalled {
		t.Fatalf("expected healthCommands to be skipped when mainRunner.enabled=false")
	}
}

func TestRunChecksFromYAMLMainRunnerEnabledDefaultsToTrue(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)

	healthCalled := false
	checkHealthAndWaitsFn = func(context.Context, string, []HTTPTestConfig, []TCPTestConfig, []string, *ContainerConfig) error {
		healthCalled = true
		return nil
	}
	checkStandardRunFn = func(context.Context, string, *ContainerConfig) error { return nil }

	// mainRunner without an explicit enabled field — health waits must still run.
	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			MainRunner: &RunnerConfig{},
			HTTP:       []HTTPTestConfig{{Port: "8080"}},
			Runners:    []RunnerConfig{{}},
		}, nil
	}

	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !healthCalled {
		t.Fatalf("expected health/wait checks to run when mainRunner.enabled is unset (defaults to true)")
	}
}

func TestLoadContainerTestYAMLMainRunnerEnabledField(t *testing.T) {
	dir := t.TempDir()

	// enabled: false should parse correctly.
	path := filepath.Join(dir, "enabled-false.yaml")
	content := "mainRunner:\n  enabled: false\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed writing yaml: %v", err)
	}
	config, err := LoadContainerTestYAML(path)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if config.MainRunner == nil {
		t.Fatalf("expected mainRunner to be present")
	}
	if config.MainRunner.Enabled == nil || *config.MainRunner.Enabled {
		t.Fatalf("expected mainRunner.enabled=false, got %v", config.MainRunner.Enabled)
	}

	// Unset enabled should yield nil (default true).
	pathUnset := filepath.Join(dir, "enabled-unset.yaml")
	if err := os.WriteFile(pathUnset, []byte("mainRunner: {}\n"), 0o600); err != nil {
		t.Fatalf("failed writing yaml: %v", err)
	}
	configUnset, err := LoadContainerTestYAML(pathUnset)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if configUnset.MainRunner == nil {
		t.Fatalf("expected mainRunner to be present")
	}
	if configUnset.MainRunner.Enabled != nil {
		t.Fatalf("expected mainRunner.enabled to be nil when unset, got %v", configUnset.MainRunner.Enabled)
	}
}

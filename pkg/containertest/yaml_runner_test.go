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
	oldHealth := checkHealthFn
	oldWaits := checkWaitsFn
	oldFiles := checkFilesExistFn
	oldCommands := checkCommandsFn
	oldStandardRun := checkStandardRunFn
	oldRunnerOutput := checkRunnerOutputFn
	t.Cleanup(func() {
		loadContainerTestYAMLFn = oldLoad
		checkHealthFn = oldHealth
		checkWaitsFn = oldWaits
		checkFilesExistFn = oldFiles
		checkCommandsFn = oldCommands
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

	mountsPath := filepath.Join(dir, "mounts.yaml")
	mountsContent := "standardRun: true\nmounts:\n  - path: /config\n    chmod: \"755\"\n    chown: \"568:568\"\n  - path: /data\n"
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
	runnersContent := "runners:\n  - env:\n      FOO: bar\n    command: myapp --version\n    readOnlyRoot: true\n    expectedOutput: \"v1.0\"\n    runTests: false\n  - {}\nfilePaths:\n  - /bin/sh\n"
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
	if r0.Env["FOO"] != "bar" || r0.Command != "myapp --version" || !r0.ReadOnlyRoot || r0.ExpectedOutput != "v1.0" {
		t.Fatalf("unexpected runner[0]: %+v", r0)
	}
	if r0.RunTests == nil || *r0.RunTests {
		t.Fatalf("expected runner[0].RunTests=false, got %v", r0.RunTests)
	}
	// runner[1] has no runTests key: should be nil (defaults to true at runtime)
	r1 := runnersConfig.Runners[1]
	if r1.RunTests != nil {
		t.Fatalf("expected runner[1].RunTests=nil (default), got %v", r1.RunTests)
	}
}

func TestRunChecksFromYAMLValidationAndErrors(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)
	checkHealthFn = func(context.Context, string, *ContainerConfig) error { return nil }

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
		return ContainerTestYAML{HTTP: []HTTPTestConfig{{Port: "8080"}}}, nil
	}
	checkHealthFn = func(context.Context, string, *ContainerConfig) error {
		return errors.New("health boom")
	}
	checkWaitsFn = CheckWaits
	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err == nil {
		t.Fatalf("expected health error")
	}
	checkHealthFn = func(context.Context, string, *ContainerConfig) error { return nil }

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

func TestRunChecksFromYAMLMountValidation(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)
	checkHealthFn = func(context.Context, string, *ContainerConfig) error { return nil }

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			StandardRun: true,
			Mounts:      []MountConfig{{Path: "  "}},
		}, nil
	}
	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err == nil || !strings.Contains(err.Error(), "mounts[0].path") {
		t.Fatalf("expected mount path validation error, got %v", err)
	}
}

func TestRunChecksFromYAMLMergesMountsIntoConfig(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)
	checkHealthFn = func(context.Context, string, *ContainerConfig) error { return nil }

	yamlMounts := []MountConfig{{Path: "/config", Chmod: "755"}}
	callerMounts := []MountConfig{{Path: "/data"}}

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			StandardRun: true,
			Mounts:      yamlMounts,
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
	calledHealth := 0
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

	checkHealthFn = func(context.Context, string, *ContainerConfig) error {
		calledHealth++
		callOrder = append(callOrder, "health")
		return nil
	}
	checkWaitsFn = func(waitCtx context.Context, image string, http []HTTPTestConfig, tcp []TCPTestConfig, cfg *ContainerConfig) error {
		calledWaits++
		callOrder = append(callOrder, "waits")
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
		callOrder = append(callOrder, "files")
		return nil
	}
	checkCommandsFn = func(context.Context, string, *ContainerConfig, []CommandTestConfig) error {
		calledCommands++
		callOrder = append(callOrder, "commands")
		return nil
	}
	checkStandardRunFn = func(context.Context, string, *ContainerConfig) error {
		calledStandardRun++
		return nil
	}

	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", &ContainerConfig{Env: map[string]string{"A": "1"}}); err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}

	if calledHealth != 1 || calledWaits != 1 || calledFiles != 1 || calledCommands != 1 || calledStandardRun != 1 {
		t.Fatalf("expected all checks once, got health=%d waits=%d files=%d commands=%d standardRun=%d", calledHealth, calledWaits, calledFiles, calledCommands, calledStandardRun)
	}

	// New order: health → files → waits → commands
	if len(callOrder) < 3 || callOrder[0] != "health" || callOrder[1] != "files" || callOrder[2] != "waits" {
		t.Fatalf("expected health→files→waits call order, got %v", callOrder)
	}
}

func TestRunChecksFromYAMLReadOnlyRootfs(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)
	checkHealthFn = func(context.Context, string, *ContainerConfig) error { return nil }

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
	checkHealthFn = func(context.Context, string, *ContainerConfig) error { return nil }

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

	// Runner readOnlyRoot propagates
	got = buildRunnerContainerConfig(RunnerConfig{ReadOnlyRoot: true}, nil, false, nil)
	if !got.ReadOnlyRootfs {
		t.Fatalf("expected ReadOnlyRootfs from runner")
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

	// Runner command becomes Command slice
	got = buildRunnerContainerConfig(RunnerConfig{Command: "myapp --flag arg"}, nil, false, nil)
	if len(got.Command) != 3 || got.Command[0] != "myapp" || got.Command[1] != "--flag" || got.Command[2] != "arg" {
		t.Fatalf("unexpected command slice: %+v", got.Command)
	}

	// Empty runner command leaves base command intact
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

func TestRunChecksFromYAMLDefaultRunnerUsesCallerConfig(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)

	var gotConfig *ContainerConfig
	checkHealthFn = func(context.Context, string, *ContainerConfig) error { return nil }
	checkStandardRunFn = func(_ context.Context, _ string, cfg *ContainerConfig) error {
		gotConfig = cfg
		return nil
	}

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{StandardRun: true}, nil
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
	var receivedEnvs []string

	checkHealthFn = func(_ context.Context, _ string, cfg *ContainerConfig) error {
		healthCalls++
		if cfg != nil {
			receivedEnvs = append(receivedEnvs, cfg.Env["RUNNER"])
		}
		return nil
	}
	checkStandardRunFn = func(context.Context, string, *ContainerConfig) error { return nil }

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			StandardRun: true,
			Runners: []RunnerConfig{
				{Env: map[string]string{"RUNNER": "first"}},
				{Env: map[string]string{"RUNNER": "second"}},
			},
		}, nil
	}

	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if healthCalls != 2 {
		t.Fatalf("expected health called twice for two runners, got %d", healthCalls)
	}
	if len(receivedEnvs) != 2 || receivedEnvs[0] != "first" || receivedEnvs[1] != "second" {
		t.Fatalf("unexpected runner envs: %v", receivedEnvs)
	}
}

func TestRunChecksFromYAMLRunnerRunTestsFalseSkipsOtherChecks(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)

	healthCalled := false
	checkHealthFn = func(context.Context, string, *ContainerConfig) error {
		healthCalled = true
		return nil
	}

	var gotCommand, gotExpected string
	var gotCfg *ContainerConfig
	checkRunnerOutputFn = func(_ context.Context, _ string, cfg *ContainerConfig, cmd, expected string) error {
		gotCfg = cfg
		gotCommand = cmd
		gotExpected = expected
		return nil
	}

	runTestsFalse := false
	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			Runners: []RunnerConfig{
				{
					Command:        "myapp --version",
					ExpectedOutput: "v1.2.3",
					Env:            map[string]string{"FOO": "bar"},
					ReadOnlyRoot:   true,
					RunTests:       &runTestsFalse,
				},
			},
			// These checks are skipped because runTests=false.
			FilePaths: []string{"/bin/sh"},
		}, nil
	}

	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if healthCalled {
		t.Fatalf("health check should be skipped when runTests=false")
	}
	if gotCommand != "myapp --version" {
		t.Fatalf("expected command=%q, got %q", "myapp --version", gotCommand)
	}
	if gotExpected != "v1.2.3" {
		t.Fatalf("expected expectedOutput=%q, got %q", "v1.2.3", gotExpected)
	}
	if gotCfg == nil || gotCfg.Env["FOO"] != "bar" || !gotCfg.ReadOnlyRootfs {
		t.Fatalf("unexpected runner config: %+v", gotCfg)
	}
}

func TestRunChecksFromYAMLRunnerExpectedOutputRunsTestsByDefault(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)

	outputCalled := false
	healthCalled := false

	checkRunnerOutputFn = func(context.Context, string, *ContainerConfig, string, string) error {
		outputCalled = true
		return nil
	}
	checkHealthFn = func(context.Context, string, *ContainerConfig) error {
		healthCalled = true
		return nil
	}
	checkStandardRunFn = func(context.Context, string, *ContainerConfig) error { return nil }

	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			StandardRun: true,
			Runners: []RunnerConfig{
				// runTests not set → defaults to true → output check + normal checks both run.
				{Command: "myapp --version", ExpectedOutput: "v1.2.3"},
			},
		}, nil
	}

	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !outputCalled {
		t.Fatalf("expected output check to be called")
	}
	if !healthCalled {
		t.Fatalf("expected health check to be called (runTests defaults to true)")
	}
}

func TestRunChecksFromYAMLRunnerOutputCheckError(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)

	checkRunnerOutputFn = func(context.Context, string, *ContainerConfig, string, string) error {
		return errors.New("output mismatch")
	}

	runTestsFalse := false
	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			Runners: []RunnerConfig{
				{Command: "myapp --version", ExpectedOutput: "v9.9.9", RunTests: &runTestsFalse},
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

	checkRunnerOutputFn = func(context.Context, string, *ContainerConfig, string, string) error {
		outputCalled = true
		return nil
	}
	checkHealthFn = func(context.Context, string, *ContainerConfig) error {
		healthCallCount++
		return nil
	}
	checkStandardRunFn = func(context.Context, string, *ContainerConfig) error { return nil }

	runTestsFalse := false
	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			StandardRun: true,
			Runners: []RunnerConfig{
				// runTests=false: only output check, no health.
				{Command: "myapp --version", ExpectedOutput: "v1", RunTests: &runTestsFalse},
				// runTests defaults to true: health runs.
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
	// Health should only be called for the second runner (runTests=false on first).
	if healthCallCount != 1 {
		t.Fatalf("expected health called once (second runner only), got %d", healthCallCount)
	}
}

func TestRunChecksFromYAMLOutputOnlyRunnerCountsAsCheck(t *testing.T) {
	ctx := context.Background()
	setYAMLRunnerSeams(t)

	checkRunnerOutputFn = func(context.Context, string, *ContainerConfig, string, string) error {
		return nil
	}
	// Health runs by default (runTests=true); mock it to avoid Docker calls.
	checkHealthFn = func(context.Context, string, *ContainerConfig) error { return nil }

	// No http/tcp/filePaths/commands/standardRun — only a runner with expectedOutput.
	loadContainerTestYAMLFn = func(string) (ContainerTestYAML, error) {
		return ContainerTestYAML{
			Runners: []RunnerConfig{
				{Command: "myapp --version", ExpectedOutput: "v1"},
			},
		}, nil
	}

	if err := RunChecksFromYAML(ctx, "img", "cfg.yaml", nil); err != nil {
		t.Fatalf("runner with expectedOutput should count as a configured check, got: %v", err)
	}
}

package containertest

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const minYAMLTimeoutSeconds = 180
const healthTimeoutExtraSeconds = 60

var (
	loadContainerTestYAMLFn = LoadContainerTestYAML
	checkHealthFn           = CheckHealth
	checkWaitsFn            = CheckWaits
	checkHealthAndWaitsFn   = CheckHealthAndWaits
	checkHealthCommandsFn   = CheckHealthCommands
	checkStandardRunFn      = CheckStandardRun
	checkRunnerOutputFn     = CheckRunnerOutput
)

// RunnerConfig defines a single runner: one container configuration under which
// all checks (or an output check) are executed.
//
// Supported keys:
//   - env: map[string]string        environment variables to set in the container
//   - entrypoint: string            override the container entrypoint for this runner
//   - cmd: string                   override container args/CMD for this runner
//   - expectedOutput: string        when non-empty, run the container (with entrypoint/cmd if set)
//     and assert this string is present in the output
//   - exitCode: int                 optional expected container exit code; when set,
//     the runner output check verifies the container exits with this exact code
//   - enabled: *bool                when explicitly false on mainRunner, skips main health/wait checks entirely;
//     defaults to true when unset
type RunnerConfig struct {
	Env            map[string]string `yaml:"env"`
	Entrypoint     string            `yaml:"entrypoint"`
	Cmd            string            `yaml:"cmd"`
	ExpectedOutput string            `yaml:"expectedOutput"`
	ExitCode       *int              `yaml:"exitCode"`
	Enabled        *bool             `yaml:"enabled"`
}

// ContainerTestYAML defines the struct-based container-test.yaml schema.
//
// Supported keys:
// - mainRunner: RunnerConfig
// - runners: []RunnerConfig
// - http: []HTTPTestConfig
// - tcp: []TCPTestConfig
// - filePaths: []string
// - healthCommands: []HealthCommandTestConfig
// - timeoutSeconds: int
// - readOnlyRootfs: bool
// - mounts: []MountConfig
//
// Note: this intentionally mirrors the exported helper structs used by runtime checks.
type ContainerTestYAML struct {
	MainRunner     *RunnerConfig             `yaml:"mainRunner"`
	Runners        []RunnerConfig            `yaml:"runners"`
	HTTP           []HTTPTestConfig          `yaml:"http"`
	TCP            []TCPTestConfig           `yaml:"tcp"`
	FilePaths      []string                  `yaml:"filePaths"`
	HealthCommands []HealthCommandTestConfig `yaml:"healthCommands"`
	TimeoutSeconds int                       `yaml:"timeoutSeconds"`
	ReadOnlyRootfs bool                      `yaml:"readOnlyRootfs"`
	Mounts         []MountConfig             `yaml:"mounts"`
}

// LoadContainerTestYAML reads and parses a container-test YAML file.
func LoadContainerTestYAML(filePath string) (ContainerTestYAML, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return ContainerTestYAML{}, fmt.Errorf("failed reading %s: %w", filePath, err)
	}

	var config ContainerTestYAML
	if err := yaml.Unmarshal(content, &config); err != nil {
		return ContainerTestYAML{}, fmt.Errorf("failed parsing %s: %w", filePath, err)
	}

	return config, nil
}

// buildRunnerContainerConfig merges a RunnerConfig with the caller-supplied base
// ContainerConfig and any YAML-level readOnlyRootfs / mounts settings into a
// fresh ContainerConfig that is safe to mutate.
func buildRunnerContainerConfig(runner RunnerConfig, base *ContainerConfig, yamlReadOnlyRootfs bool, yamlMounts []MountConfig) *ContainerConfig {
	merged := &ContainerConfig{}
	if base != nil {
		// Shallow-copy scalars; deep-copy slices/maps so callers are not mutated.
		merged.ReadOnlyRootfs = base.ReadOnlyRootfs
		if len(base.Env) > 0 {
			merged.Env = make(map[string]string, len(base.Env))
			for k, v := range base.Env {
				merged.Env[k] = v
			}
		}
		merged.Command = append([]string{}, base.Command...)
		merged.Mounts = append([]MountConfig{}, base.Mounts...)
	}

	// Apply YAML-level settings
	if yamlReadOnlyRootfs {
		merged.ReadOnlyRootfs = true
	}
	merged.Mounts = append(merged.Mounts, yamlMounts...)

	// Apply runner-specific overrides
	if len(runner.Env) > 0 {
		if merged.Env == nil {
			merged.Env = make(map[string]string, len(runner.Env))
		}
		for k, v := range runner.Env {
			merged.Env[k] = v
		}
	}
	runnerEntrypoint := strings.TrimSpace(runner.Entrypoint)
	runnerCmd := strings.TrimSpace(runner.Cmd)
	if runnerEntrypoint != "" || runnerCmd != "" {
		merged.Command = nil
		if runnerEntrypoint != "" {
			merged.Command = append(merged.Command, strings.Fields(runnerEntrypoint)...)
		}
		if runnerCmd != "" {
			merged.Command = append(merged.Command, strings.Fields(runnerCmd)...)
		}
	}
	return merged
}

func buildMainRunnerSpec(spec *RunnerConfig) RunnerConfig {
	if spec == nil {
		return RunnerConfig{}
	}

	mainRunner := RunnerConfig{
		Entrypoint: spec.Entrypoint,
		Cmd:        spec.Cmd,
	}
	if len(spec.Env) > 0 {
		mainRunner.Env = make(map[string]string, len(spec.Env))
		for key, value := range spec.Env {
			mainRunner.Env[key] = value
		}
	}

	return mainRunner
}

// RunChecksFromYAML runs container checks defined in a struct-based container-test YAML file.
//
// When the YAML defines one or more runners, each runner spawns its own container
// configuration (env, command) and checks execute per runner.
// The optional mainRunner may override env/entrypoint/cmd for wait checks only;
// mainRunner expectedOutput/exitCode are ignored.
// For each runner:
//   - If expectedOutput is non-empty, an output check is performed first.
//   - Wait checks are executed once by the dedicated waits runner.
//   - A standard container run is performed for that runner.
//
// When no runners are defined, runner-specific checks are skipped.
func RunChecksFromYAML(ctx context.Context, image string, yamlPath string, containerConfig *ContainerConfig) error {
	config, err := loadContainerTestYAMLFn(yamlPath)
	if err != nil {
		return err
	}

	logInfo("Loaded container test YAML: path=%s runners=%d http=%d tcp=%d", yamlPath, len(config.Runners), len(config.HTTP), len(config.TCP))

	for index, mount := range config.Mounts {
		trimmedPath := strings.TrimSpace(mount.Path)
		if trimmedPath == "" {
			return fmt.Errorf("mounts[%d].path must not be empty", index)
		}
		if !strings.HasPrefix(trimmedPath, "/") {
			return fmt.Errorf("mounts[%d].path must be an absolute path starting with '/'", index)
		}
		config.Mounts[index].Path = trimmedPath
	}

	// Build the list of configured runners. If none are configured, runner-specific
	// checks are skipped.
	runners := config.Runners

	for index, filePath := range config.FilePaths {
		trimmedPath := strings.TrimSpace(filePath)
		if trimmedPath == "" {
			return fmt.Errorf("filePaths[%d] must not be empty", index)
		}
		config.FilePaths[index] = trimmedPath
	}

	hasTCPHTTPChecks := len(config.HTTP) > 0 || len(config.TCP) > 0
	mainRunnerFilePaths := append([]string{}, config.FilePaths...)
	mainRunnerSpec := buildMainRunnerSpec(config.MainRunner)

	mainRunnerDisabled := config.MainRunner != nil && config.MainRunner.Enabled != nil && !*config.MainRunner.Enabled
	hasFileWaits := len(mainRunnerFilePaths) > 0
	hasHealthCommandWaits := len(config.HealthCommands) > 0
	hasCombinedWaits := hasTCPHTTPChecks || hasFileWaits
	mainRunnerTimeoutSeconds := config.TimeoutSeconds
	if !mainRunnerDisabled && (hasCombinedWaits || hasHealthCommandWaits) {
		mainRunnerCtx := ctx
		if mainRunnerTimeoutSeconds > 0 {
			effectiveTimeoutSeconds := mainRunnerTimeoutSeconds
			if effectiveTimeoutSeconds < minYAMLTimeoutSeconds {
				effectiveTimeoutSeconds = minYAMLTimeoutSeconds
			}
			timeoutBaseCtx := ctx
			configuredTimeout := time.Duration(effectiveTimeoutSeconds) * time.Second
			if hasTCPHTTPChecks || hasFileWaits {
				configuredTimeout += healthTimeoutExtraSeconds * time.Second
			}
			if deadline, hasDeadline := ctx.Deadline(); hasDeadline {
				remaining := time.Until(deadline)
				if remaining > 0 && remaining < configuredTimeout {
					logWarn("main runner: parent context deadline (%s) is shorter than configured timeout (%s); honoring configured timeout", remaining.Round(time.Second), configuredTimeout.Round(time.Second))
					timeoutBaseCtx = context.WithoutCancel(ctx)
				}
			}

			var mainRunnerCancel context.CancelFunc
			mainRunnerCtx, mainRunnerCancel = context.WithTimeout(timeoutBaseCtx, configuredTimeout)
			defer mainRunnerCancel()
		}

		mainRunnerCfg := buildRunnerContainerConfig(mainRunnerSpec, containerConfig, config.ReadOnlyRootfs, config.Mounts)
		if hasCombinedWaits {
			if hasFileWaits && !hasTCPHTTPChecks {
				logInfo("Main runner: running health check wait for file-based waits")
				if err := checkHealthFn(mainRunnerCtx, image, mainRunnerCfg); err != nil {
					return err
				}
			}
			logInfo("Main runner: running combined checks (http=%d tcp=%d filePaths=%d)", len(config.HTTP), len(config.TCP), len(mainRunnerFilePaths))
			if err := checkHealthAndWaitsFn(mainRunnerCtx, image, config.HTTP, config.TCP, mainRunnerFilePaths, mainRunnerCfg); err != nil {
				return err
			}
		}
		if hasHealthCommandWaits {
			logInfo("Main runner: running health command waits (healthCommands=%d)", len(config.HealthCommands))
			if err := checkHealthCommandsFn(mainRunnerCtx, image, mainRunnerCfg, config.HealthCommands); err != nil {
				return err
			}
		}
	} else {
		if mainRunnerDisabled {
			logInfo("Main runner: skipping health/wait/file checks (mainRunner.enabled=false)")
		} else {
			logInfo("Main runner: skipping health/wait/file checks (no tcp/http/filePaths/healthCommands configured)")
		}
	}

	for i, runner := range runners {
		logInfo("Runner[%d]: starting (expectedOutput=%t exitCodeSet=%t timeoutSeconds=%d)", i, strings.TrimSpace(runner.ExpectedOutput) != "", runner.ExitCode != nil, config.TimeoutSeconds)
		runnerCtx := ctx
		if config.TimeoutSeconds > 0 {
			effectiveTimeoutSeconds := config.TimeoutSeconds
			if effectiveTimeoutSeconds < minYAMLTimeoutSeconds {
				logWarn("timeoutSeconds=%d is very low for container startup; using minimum %d seconds", config.TimeoutSeconds, minYAMLTimeoutSeconds)
				effectiveTimeoutSeconds = minYAMLTimeoutSeconds
			}

			timeoutBaseCtx := ctx
			configuredTimeout := time.Duration(effectiveTimeoutSeconds) * time.Second
			if deadline, hasDeadline := ctx.Deadline(); hasDeadline {
				remaining := time.Until(deadline)
				if remaining > 0 && remaining < configuredTimeout {
					logWarn("runner[%d]: parent context deadline (%s) is shorter than configured timeout (%s); honoring configured timeout", i, remaining.Round(time.Second), configuredTimeout.Round(time.Second))
					timeoutBaseCtx = context.WithoutCancel(ctx)
				}
			}

			var cancel context.CancelFunc
			runnerCtx, cancel = context.WithTimeout(timeoutBaseCtx, configuredTimeout)
			defer cancel()
		}

		runnerCfg := buildRunnerContainerConfig(runner, containerConfig, config.ReadOnlyRootfs, config.Mounts)

		// Run output check if expectedOutput is set.
		if strings.TrimSpace(runner.ExpectedOutput) != "" {
			logInfo("Runner[%d]: running output check", i)
			runnerCommand := strings.TrimSpace(runner.Entrypoint)
			runnerCmd := strings.TrimSpace(runner.Cmd)
			if runnerCmd != "" {
				if runnerCommand != "" {
					runnerCommand += " " + runnerCmd
				} else {
					runnerCommand = runnerCmd
				}
			}
			if err := checkRunnerOutputFn(runnerCtx, image, runnerCfg, runnerCommand, runner.ExpectedOutput, runner.ExitCode); err != nil {
				return fmt.Errorf("runner[%d] output check failed: %w", i, err)
			}
		}

		logInfo("Runner[%d]: wait checks already executed by main runner", i)

		logInfo("Runner[%d]: running standard run check", i)
		if err := checkStandardRunFn(runnerCtx, image, runnerCfg); err != nil {
			return err
		}
		logInfo("Runner[%d]: completed", i)
	}

	return nil
}

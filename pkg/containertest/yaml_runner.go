package containertest

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const minYAMLTimeoutSeconds = 120

var (
	loadContainerTestYAMLFn = LoadContainerTestYAML
	checkHealthFn           = CheckHealth
	checkWaitsFn            = CheckWaits
	checkFilesExistFn       = CheckFilesExist
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
//   - command: string               legacy alias for entrypoint (backward compatible, DO NOT USE)
//   - readOnlyRoot: bool            mount the container root filesystem as read-only
//   - timeoutSeconds: int            optional per-runner timeout in seconds (min 120 enforced)
//   - expectedOutput: string        when non-empty, run the container (with entrypoint/cmd if set)
//     and assert this string is present in the output
//   - exitCode: int                 optional expected container exit code; when set,
//     the runner output check verifies the container exits with this exact code
//   - runTests: bool                whether to run the other checks (health, file, tcp, http,
//     standard run) for this runner; defaults to true when omitted
type RunnerConfig struct {
	Env            map[string]string `yaml:"env"`
	Entrypoint     string            `yaml:"entrypoint"`
	Cmd            string            `yaml:"cmd"`
	Command        string            `yaml:"command"`
	ReadOnlyRoot   bool              `yaml:"readOnlyRoot"`
	TimeoutSeconds int               `yaml:"timeoutSeconds"`
	ExpectedOutput string            `yaml:"expectedOutput"`
	ExitCode       *int              `yaml:"exitCode"`
	RunTests       *bool             `yaml:"runTests"`
}

// ContainerTestYAML defines the struct-based container-test.yaml schema.
//
// Supported keys:
// - runners: []RunnerConfig
// - http: []HTTPTestConfig
// - tcp: []TCPTestConfig
// - healthCommands: []HealthCommandTestConfig
// - filePaths: []string
// - readOnlyRootfs: bool
// - mounts: []MountConfig
//
// Note: this intentionally mirrors the exported helper structs used by runtime checks.
type ContainerTestYAML struct {
	Runners        []RunnerConfig            `yaml:"runners"`
	HTTP           []HTTPTestConfig          `yaml:"http"`
	TCP            []TCPTestConfig           `yaml:"tcp"`
	HealthCommands []HealthCommandTestConfig `yaml:"healthCommands"`
	FilePaths      []string                  `yaml:"filePaths"`
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
	if runnerEntrypoint == "" {
		runnerEntrypoint = strings.TrimSpace(runner.Command)
	}
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
	if runner.ReadOnlyRoot {
		merged.ReadOnlyRootfs = true
	}

	return merged
}

// RunChecksFromYAML runs container checks defined in a struct-based container-test YAML file.
//
// When the YAML defines one or more runners, each runner spawns its own container
// configuration (env, command, readOnlyRoot) and checks execute per runner.
// For each runner:
//   - If expectedOutput is non-empty, an output check is performed first.
//   - If runTests is true (the default when omitted), the normal check sequence follows:
//     health (when tcp/http/healthCommands are configured) → file → tcp/http waits → healthCommands → standardRun.
//   - If runTests is explicitly false, health/file/wait/health-command checks are skipped,
//     but a standard container run is still performed for that runner.
//
// When no runners are defined a single default runner is used, which is equivalent
// to the previous single-pass behaviour.
func RunChecksFromYAML(ctx context.Context, image string, yamlPath string, containerConfig *ContainerConfig) error {
	config, err := loadContainerTestYAMLFn(yamlPath)
	if err != nil {
		return err
	}

	logInfo("Loaded container test YAML: path=%s runners=%d http=%d tcp=%d fileChecks=%d healthCommands=%d", yamlPath, len(config.Runners), len(config.HTTP), len(config.TCP), len(config.FilePaths), len(config.HealthCommands))

	for index, command := range config.HealthCommands {
		if strings.TrimSpace(command.Command) == "" {
			return fmt.Errorf("healthCommands[%d].command must not be empty", index)
		}
	}

	for index, filePath := range config.FilePaths {
		if strings.TrimSpace(filePath) == "" {
			return fmt.Errorf("filePaths[%d] must not be empty", index)
		}
	}

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

	// Build the list of runners; default to a single empty runner for backward compatibility.
	runners := config.Runners
	if len(runners) == 0 {
		runners = []RunnerConfig{{}}
	}

	hasHealthCheckPreconditions := len(config.HTTP) > 0 || len(config.TCP) > 0 || len(config.HealthCommands) > 0

	for i, runner := range runners {
		logInfo("Runner[%d]: starting (expectedOutput=%t exitCodeSet=%t runTests=%t timeoutSeconds=%d)", i, strings.TrimSpace(runner.ExpectedOutput) != "", runner.ExitCode != nil, runner.RunTests == nil || *runner.RunTests, runner.TimeoutSeconds)
		runnerCtx := ctx
		if runner.TimeoutSeconds > 0 {
			effectiveTimeoutSeconds := runner.TimeoutSeconds
			if effectiveTimeoutSeconds < minYAMLTimeoutSeconds {
				logWarn("runner[%d].timeoutSeconds=%d is very low for container startup; using minimum %d seconds", i, runner.TimeoutSeconds, minYAMLTimeoutSeconds)
				effectiveTimeoutSeconds = minYAMLTimeoutSeconds
			}

			var cancel context.CancelFunc
			runnerCtx, cancel = context.WithTimeout(ctx, time.Duration(effectiveTimeoutSeconds)*time.Second)
			defer cancel()
		}

		runnerCfg := buildRunnerContainerConfig(runner, containerConfig, config.ReadOnlyRootfs, config.Mounts)

		// Run output check if expectedOutput is set.
		if strings.TrimSpace(runner.ExpectedOutput) != "" {
			logInfo("Runner[%d]: running output check", i)
			runnerCommand := strings.TrimSpace(runner.Entrypoint)
			if runnerCommand == "" {
				runnerCommand = strings.TrimSpace(runner.Command)
			}
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

		// Skip normal checks when runTests is explicitly false; default is true.
		if runner.RunTests != nil && !*runner.RunTests {
			logInfo("Runner[%d]: skipping health/file/wait/health-command checks (runTests=false)", i)
			if err := checkStandardRunFn(runnerCtx, image, runnerCfg); err != nil {
				return err
			}
			continue
		}

		// Normal check sequence: health (when tcp/http/healthCommands are configured) → file → tcp/http waits → healthCommands → standardRun.
		if hasHealthCheckPreconditions {
			logInfo("Runner[%d]: running health check", i)
			if err := checkHealthFn(runnerCtx, image, runnerCfg); err != nil {
				return err
			}
		} else {
			logInfo("Runner[%d]: skipping health check (no tcp/http/healthCommands configured)", i)
		}

		if len(config.FilePaths) > 0 {
			logInfo("Runner[%d]: running file checks (%d)", i, len(config.FilePaths))
			if err := checkFilesExistFn(runnerCtx, image, config.FilePaths, runnerCfg); err != nil {
				return err
			}
		}

		if len(config.HTTP) > 0 || len(config.TCP) > 0 {
			logInfo("Runner[%d]: running wait checks (http=%d tcp=%d)", i, len(config.HTTP), len(config.TCP))
			if err := checkWaitsFn(runnerCtx, image, config.HTTP, config.TCP, runnerCfg); err != nil {
				return err
			}
		}

		if len(config.HealthCommands) > 0 {
			logInfo("Runner[%d]: running health commands (%d)", i, len(config.HealthCommands))
			if err := checkHealthCommandsFn(runnerCtx, image, runnerCfg, config.HealthCommands); err != nil {
				return err
			}
		}

		logInfo("Runner[%d]: running standard run check", i)
		if err := checkStandardRunFn(runnerCtx, image, runnerCfg); err != nil {
			return err
		}
		logInfo("Runner[%d]: completed", i)
	}

	return nil
}

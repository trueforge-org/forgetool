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
	checkCommandsFn         = CheckCommands
	checkStandardRunFn      = CheckStandardRun
	checkRunnerOutputFn     = CheckRunnerOutput
)

// RunnerConfig defines a single runner: one container configuration under which
// all checks (or an output check) are executed.
//
// Supported keys:
//   - env: map[string]string        environment variables to set in the container
//   - command: string               override the container CMD; used as "sh -c <command>"
//     when expectedOutput is set, otherwise applied as CMD
//   - readOnlyRoot: bool            mount the container root filesystem as read-only
//   - expectedOutput: string        when non-empty, run the container (with command if set)
//     and assert this string is present in the output
//   - runTests: bool                whether to run the other checks (health, file, tcp, http,
//     commands, standardRun) for this runner; defaults to true when omitted
type RunnerConfig struct {
	Env            map[string]string `yaml:"env"`
	Command        string            `yaml:"command"`
	ReadOnlyRoot   bool              `yaml:"readOnlyRoot"`
	ExpectedOutput string            `yaml:"expectedOutput"`
	RunTests       *bool             `yaml:"runTests"`
}

// ContainerTestYAML defines the struct-based container-test.yaml schema.
//
// Supported keys:
// - timeoutSeconds
// - runners: []RunnerConfig
// - http: []HTTPTestConfig
// - tcp: []TCPTestConfig
// - healthCommands: []HealthCommandTestConfig
// - commands: []CommandTestConfig
// - filePaths: []string
// - standardRun: bool
// - readOnlyRootfs: bool
// - mounts: []MountConfig
//
// Note: this intentionally mirrors the exported helper structs used by runtime checks.
type ContainerTestYAML struct {
	TimeoutSeconds int                       `yaml:"timeoutSeconds"`
	Runners        []RunnerConfig            `yaml:"runners"`
	HTTP           []HTTPTestConfig          `yaml:"http"`
	TCP            []TCPTestConfig           `yaml:"tcp"`
	HealthCommands []HealthCommandTestConfig `yaml:"healthCommands"`
	Commands       []CommandTestConfig       `yaml:"commands"`
	FilePaths      []string                  `yaml:"filePaths"`
	StandardRun    bool                      `yaml:"standardRun"`
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
	if runner.Command != "" {
		merged.Command = strings.Fields(runner.Command)
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
//     health → file → tcp/http waits → healthCommands → commands → standardRun.
//   - If runTests is explicitly false, the normal check sequence is skipped for that runner.
//
// When no runners are defined a single default runner is used, which is equivalent
// to the previous single-pass behaviour.
func RunChecksFromYAML(ctx context.Context, image string, yamlPath string, containerConfig *ContainerConfig) error {
	config, err := loadContainerTestYAMLFn(yamlPath)
	if err != nil {
		return err
	}

	if config.TimeoutSeconds > 0 {
		effectiveTimeoutSeconds := config.TimeoutSeconds
		if effectiveTimeoutSeconds < minYAMLTimeoutSeconds {
			logWarn("timeoutSeconds=%d is very low for container startup; using minimum %d seconds", config.TimeoutSeconds, minYAMLTimeoutSeconds)
			effectiveTimeoutSeconds = minYAMLTimeoutSeconds
		}

		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(effectiveTimeoutSeconds)*time.Second)
		defer cancel()
	}

	// Validate that at least one check or output-check runner is configured.
	hasChecks := len(config.HTTP) > 0 || len(config.TCP) > 0 || len(config.FilePaths) > 0 || len(config.HealthCommands) > 0 || len(config.Commands) > 0 || config.StandardRun
	if !hasChecks {
		for _, r := range config.Runners {
			if strings.TrimSpace(r.ExpectedOutput) != "" {
				hasChecks = true
				break
			}
		}
	}
	if !hasChecks {
		return fmt.Errorf("no checks configured in %s", yamlPath)
	}

	for index, command := range config.Commands {
		if strings.TrimSpace(command.Command) == "" {
			return fmt.Errorf("commands[%d].command must not be empty", index)
		}
	}

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

	for i, runner := range runners {
		runnerCfg := buildRunnerContainerConfig(runner, containerConfig, config.ReadOnlyRootfs, config.Mounts)

		// Run output check if expectedOutput is set.
		if strings.TrimSpace(runner.ExpectedOutput) != "" {
			logInfo("Runner[%d]: running output check", i)
			if err := checkRunnerOutputFn(ctx, image, runnerCfg, runner.Command, runner.ExpectedOutput); err != nil {
				return fmt.Errorf("runner[%d] output check failed: %w", i, err)
			}
		}

		// Skip normal checks when runTests is explicitly false; default is true.
		if runner.RunTests != nil && !*runner.RunTests {
			logInfo("Runner[%d]: skipping other checks (runTests=false)", i)
			continue
		}

		// Normal check sequence: health → file → tcp/http waits → healthCommands → commands → standardRun.
		if err := checkHealthFn(ctx, image, runnerCfg); err != nil {
			return err
		}

		if len(config.FilePaths) > 0 {
			if err := checkFilesExistFn(ctx, image, config.FilePaths, runnerCfg); err != nil {
				return err
			}
		}

		if len(config.HTTP) > 0 || len(config.TCP) > 0 {
			if err := checkWaitsFn(ctx, image, config.HTTP, config.TCP, runnerCfg); err != nil {
				return err
			}
		}

		if len(config.HealthCommands) > 0 {
			if err := checkHealthCommandsFn(ctx, image, runnerCfg, config.HealthCommands); err != nil {
				return err
			}
		}

		if len(config.Commands) > 0 {
			if err := checkCommandsFn(ctx, image, runnerCfg, config.Commands); err != nil {
				return err
			}
		}

		if config.StandardRun {
			if err := checkStandardRunFn(ctx, image, runnerCfg); err != nil {
				return err
			}
		}
	}

	return nil
}

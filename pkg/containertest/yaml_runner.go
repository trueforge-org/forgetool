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
	checkCommandsFn         = CheckCommands
	checkStandardRunFn      = CheckStandardRun
)

// ContainerTestYAML defines the struct-based container-test.yaml schema.
//
// Supported keys:
// - timeoutSeconds
// - http: []HTTPTestConfig
// - tcp: []TCPTestConfig
// - commands: []CommandTestConfig
// - filePaths: []string
// - standardRun: bool
// - mounts: []MountConfig
//
// Note: this intentionally mirrors the exported helper structs used by runtime checks.
type ContainerTestYAML struct {
	TimeoutSeconds int                 `yaml:"timeoutSeconds"`
	HTTP           []HTTPTestConfig    `yaml:"http"`
	TCP            []TCPTestConfig     `yaml:"tcp"`
	Commands       []CommandTestConfig `yaml:"commands"`
	FilePaths      []string            `yaml:"filePaths"`
	StandardRun    bool                `yaml:"standardRun"`
	Mounts         []MountConfig       `yaml:"mounts"`
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

// RunChecksFromYAML runs container checks defined in a struct-based container-test YAML file.
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

	if len(config.HTTP) == 0 && len(config.TCP) == 0 && len(config.FilePaths) == 0 && len(config.Commands) == 0 && !config.StandardRun {
		return fmt.Errorf("no checks configured in %s", yamlPath)
	}

	for index, command := range config.Commands {
		if strings.TrimSpace(command.Command) == "" {
			return fmt.Errorf("commands[%d].command must not be empty", index)
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

	// Merge YAML mounts into the effective container config so every check run picks them up.
	effectiveConfig := containerConfig
	if len(config.Mounts) > 0 {
		merged := &ContainerConfig{}
		if containerConfig != nil {
			*merged = *containerConfig
		}
		merged.Mounts = append(append([]MountConfig{}, merged.Mounts...), config.Mounts...)
		effectiveConfig = merged
	}

	if err := checkHealthFn(ctx, image, effectiveConfig); err != nil {
		return err
	}

	if len(config.HTTP) > 0 || len(config.TCP) > 0 {
		if err := checkWaitsFn(ctx, image, config.HTTP, config.TCP, effectiveConfig); err != nil {
			return err
		}
	}

	if len(config.FilePaths) > 0 {
		if err := checkFilesExistFn(ctx, image, config.FilePaths, effectiveConfig); err != nil {
			return err
		}
	}

	if len(config.Commands) > 0 {
		if err := checkCommandsFn(ctx, image, effectiveConfig, config.Commands); err != nil {
			return err
		}
	}

	if config.StandardRun {
		if err := checkStandardRunFn(ctx, image, effectiveConfig); err != nil {
			return err
		}
	}

	return nil
}

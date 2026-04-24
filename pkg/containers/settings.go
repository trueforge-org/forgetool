package containers

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// AppSettings represents the contents of an app's settings.yaml file.
type AppSettings struct {
	SchemaVersion  int             `yaml:"schema_version"`
	UpstreamEnvURL string          `yaml:"upstream_env_url"`
	Ports          []PortSetting   `yaml:"ports"`
	Env            []EnvSetting    `yaml:"env"`
	Volumes        []VolumeSetting `yaml:"volumes"`
}

// PortSetting is a single port entry in settings.yaml.
type PortSetting struct {
	Port     int    `yaml:"port"`
	Protocol string `yaml:"protocol"`
	Required bool   `yaml:"required"`
}

// EnvSetting is a single environment variable entry in settings.yaml.
type EnvSetting struct {
	Name     string `yaml:"name"`
	Default  string `yaml:"default"`
	Required bool   `yaml:"required"`
}

// VolumeSetting is a single volume entry in settings.yaml.
type VolumeSetting struct {
	Path     string `yaml:"path"`
	Required bool   `yaml:"required"`
}

// ParseSettings reads an app's settings.yaml file. It returns (settings, true,
// nil) on success, (zero, false, nil) if the file does not exist, or an error
// on any other failure.
func ParseSettings(path string) (AppSettings, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return AppSettings{}, false, nil
		}
		return AppSettings{}, false, err
	}
	var s AppSettings
	if err := yaml.Unmarshal(data, &s); err != nil {
		return AppSettings{}, false, fmt.Errorf("unmarshal %s: %w", path, err)
	}
	return s, true, nil
}

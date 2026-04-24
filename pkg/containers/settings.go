package containers

import (
	"errors"
	"fmt"
	"os"

	"github.com/compose-spec/compose-go/v2/loader"
	composetypes "github.com/compose-spec/compose-go/v2/types"
	"gopkg.in/yaml.v3"
)

// AppSettings represents the contents of an app's settings.yaml file.
type AppSettings struct {
	SchemaVersion   int             `yaml:"schema_version"`
	UpstreamEnvURL  string          `yaml:"upstream_env_url"`
	Ports           []PortSetting   `yaml:"ports"`
	Env             []EnvSetting    `yaml:"env"`
	Volumes         []VolumeSetting `yaml:"volumes"`
	Dependencies    []Dependency    `yaml:"dependencies"`
	OptDependencies []Dependency    `yaml:"opt_dependencies"`
	// Compose is an optional docker-compose service-spec fragment merged on
	// top of the service generated for the app itself. Same merge semantics
	// as Dependency.Compose.
	Compose composetypes.ServiceConfig `yaml:"compose,omitempty"`
}

// UnmarshalYAML decodes the AppSettings document, routing the top-level
// "compose" field through compose-go's loader.Transform so that every
// compose syntax variant is supported (list-form environment, port shorthand
// strings, durations, ...).
func (s *AppSettings) UnmarshalYAML(value *yaml.Node) error {
	type rawAppSettings struct {
		SchemaVersion   int             `yaml:"schema_version"`
		UpstreamEnvURL  string          `yaml:"upstream_env_url"`
		Ports           []PortSetting   `yaml:"ports"`
		Env             []EnvSetting    `yaml:"env"`
		Volumes         []VolumeSetting `yaml:"volumes"`
		Dependencies    []Dependency    `yaml:"dependencies"`
		OptDependencies []Dependency    `yaml:"opt_dependencies"`
		Compose         map[string]any  `yaml:"compose"`
	}
	var raw rawAppSettings
	if err := value.Decode(&raw); err != nil {
		return err
	}
	s.SchemaVersion = raw.SchemaVersion
	s.UpstreamEnvURL = raw.UpstreamEnvURL
	s.Ports = raw.Ports
	s.Env = raw.Env
	s.Volumes = raw.Volumes
	s.Dependencies = raw.Dependencies
	s.OptDependencies = raw.OptDependencies
	s.Compose = composetypes.ServiceConfig{}
	if len(raw.Compose) > 0 {
		var svc composetypes.ServiceConfig
		if err := loader.Transform(raw.Compose, &svc); err != nil {
			return fmt.Errorf("decode top-level compose override: %w", err)
		}
		s.Compose = svc
	}
	return nil
}

// Dependency describes another container in the same repository that this
// container depends on at runtime. The "name" field identifies the
// dependency's app directory (used to look up its settings.yaml) and is also
// used as the docker-compose service name. The optional "compose" field is
// a docker-compose service-spec fragment (the "services.<name>" subsection
// of the compose specification, modelled by compose-go's ServiceConfig);
// when present its fields are merged on top of the service generated from
// the dependency's settings.yaml using compose-spec merge semantics.
//
// Example settings.yaml entry:
//
//	dependencies:
//	  - name: postgresql
//	    compose:
//	      restart: always
//	      environment:
//	        POSTGRES_PASSWORD: secret
type Dependency struct {
	Name    string                     `yaml:"name"`
	Compose composetypes.ServiceConfig `yaml:"compose,omitempty"`
}

// UnmarshalYAML decodes a dependency entry. The "compose" field is routed
// through compose-go's loader.Transform so that every compose syntax variant
// accepted by docker-compose (list-form environment, port shorthand strings,
// durations, ...) is supported.
func (d *Dependency) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		Name    string         `yaml:"name"`
		Compose map[string]any `yaml:"compose"`
	}
	if err := value.Decode(&raw); err != nil {
		return err
	}
	d.Name = raw.Name
	d.Compose = composetypes.ServiceConfig{}
	if len(raw.Compose) > 0 {
		var svc composetypes.ServiceConfig
		if err := loader.Transform(raw.Compose, &svc); err != nil {
			return fmt.Errorf("decode dependency %q compose override: %w", raw.Name, err)
		}
		d.Compose = svc
	}
	return nil
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

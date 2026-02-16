package containertest

import (
	"context"
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// A minimal config used only for running command entrypoints.
type config struct {
	Image    string            `yaml:"image"`
	Env      map[string]string `yaml:"env"`
	Commands []string          `yaml:"commands"`
}

// Test reads a YAML file (path from CONTAINER_TEST_CONFIG or containertest.yml)
// and runs each command listed in `commands` as a container entrypoint.
func Test(t *testing.T) {
	ctx := context.Background()

	cfgPath := os.Getenv("CONTAINER_TEST_CONFIG")
	if cfgPath == "" {
		cfgPath = "containertest.yml"
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("failed to read config %q: %v", cfgPath, err)
	}

	var cfg config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("failed to parse config %q: %v", cfgPath, err)
	}

	image := GetTestImage(cfg.Image)
	if image == "" {
		t.Fatalf("no image specified (set `image` in %s or TEST_IMAGE)", cfgPath)
	}

	if len(cfg.Commands) == 0 {
		t.Fatalf("no commands defined in %s", cfgPath)
	}

	for _, cmd := range cfg.Commands {
		cmd := cmd
		t.Run(cmd, func(t *testing.T) {
			parts := strings.Fields(cmd)
			if len(parts) == 0 {
				t.Fatalf("empty command")
			}
			entry := parts[0]
			args := []string{}
			if len(parts) > 1 {
				args = parts[1:]
			}

			TestCommandSucceeds(t, ctx, image, &ContainerConfig{Env: cfg.Env}, entry, args...)
		})
	}
}

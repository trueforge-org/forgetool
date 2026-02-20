package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/trueforge-org/forgetool/pkg/containertest"
)

var containerTestLongHelp = strings.TrimSpace(`
Run container checks from a YAML file.

Before any configured checks are executed, the container must report healthy
via Docker HEALTHCHECK.
`)

var (
	containerTestImage       string
	containerTestConfigPath  string
	containerTestEnvPairs    []string
	containerTestRunChecksFn = containertest.RunChecksFromYAML
)

func parseContainerTestEnv(pairs []string) (map[string]string, error) {
	env := map[string]string{}
	for _, pair := range pairs {
		key, value, ok := strings.Cut(pair, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("invalid --env value %q, expected KEY=VALUE", pair)
		}
		env[key] = value
	}
	return env, nil
}

func runContainerTest(ctx context.Context, image string, configPath string, envPairs []string) error {
	if strings.TrimSpace(image) == "" {
		return fmt.Errorf("--image is required")
	}
	if strings.TrimSpace(configPath) == "" {
		return fmt.Errorf("--config is required")
	}

	env, err := parseContainerTestEnv(envPairs)
	if err != nil {
		return err
	}

	config := &containertest.ContainerConfig{Env: env}
	if err := containerTestRunChecksFn(ctx, image, configPath, config); err != nil {
		return fmt.Errorf("check failed: %w", err)
	}

	return nil
}

var containerTestCmd = &cobra.Command{
	Use:     "test",
	Short:   "Run container tests from YAML configuration",
	Example: "forgetool container test --image ghcr.io/trueforge-org/myimage:latest --config ./container-test.yaml",
	Long:    containerTestLongHelp,
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runContainerTest(cmd.Context(), containerTestImage, containerTestConfigPath, containerTestEnvPairs)
	},
}

func init() {
	containerTestCmd.Flags().StringVar(&containerTestImage, "image", "", "container image to run")
	containerTestCmd.Flags().StringVar(&containerTestConfigPath, "config", "", "path to container-test.yaml")
	containerTestCmd.Flags().StringArrayVar(&containerTestEnvPairs, "env", nil, "environment variable (KEY=VALUE), repeatable")

	containerCmd.AddCommand(containerTestCmd)
}

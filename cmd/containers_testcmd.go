package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/trueforge-org/forgetool/pkg/containertest"
)

var containersTestLongHelp = strings.TrimSpace(`
Run container checks from a YAML file.

Before any configured checks are executed, the container must report healthy
via Docker HEALTHCHECK.
`)

var (
	containersTestImage       string
	containersTestConfigPath  string
	containersTestEnvPairs    []string
	containersTestRunChecksFn = containertest.RunChecksFromYAML
)

func parseContainersTestEnv(pairs []string) (map[string]string, error) {
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

func runContainersTest(ctx context.Context, image string, configPath string, envPairs []string) error {
	if strings.TrimSpace(image) == "" {
		return fmt.Errorf("--image is required")
	}
	if strings.TrimSpace(configPath) == "" {
		return fmt.Errorf("--config is required")
	}

	env, err := parseContainersTestEnv(envPairs)
	if err != nil {
		return err
	}

	config := &containertest.ContainerConfig{Env: env}
	if err := containersTestRunChecksFn(ctx, image, configPath, config); err != nil {
		return fmt.Errorf("check failed: %w", err)
	}

	return nil
}

var containersTestCmd = &cobra.Command{
	Use:     "test",
	Short:   "Run container tests from YAML configuration",
	Example: "forgetool containers test --image ghcr.io/trueforge-org/myimage:latest --config ./container-test.yaml",
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

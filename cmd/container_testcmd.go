package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/trueforge-org/forgetool/pkg/containertest"
)

var containerTestLongHelp = strings.TrimSpace(`
Run container checks from a YAML file.
`)

var (
	containerTestConfigPath = "container-test.yaml"
	containerTestRunner     = containertest.RunFromConfigFile
)

func runContainerTest(configPath string) error {
	return containerTestRunner(configPath)
}

var containerTestCmd = &cobra.Command{
	Use:     "test",
	Short:   "Run container tests from YAML configuration",
	Example: "forgetool container test --config ./container-test.yaml",
	Long:    containerTestLongHelp,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runContainerTest(containerTestConfigPath)
	},
}

func init() {
	containerTestCmd.Flags().StringVarP(&containerTestConfigPath, "config", "c", "container-test.yaml", "Path to container test config YAML")
	containerCmd.AddCommand(containerTestCmd)
}

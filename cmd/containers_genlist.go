package cmd

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/trueforge-org/forgetool/pkg/containers/website"
)

var containersGenListLongHelp = strings.TrimSpace(`
Generates a JSON file listing all containers found under the given path(s).
Each container is identified by its docker-bake.hcl file.
`)

func defaultContainersGenListOptionsFactory() *website.ContainerListOptions {
	return &website.ContainerListOptions{
		OutputPath: "./containers.json",
	}
}

func defaultContainersGenListGetData(opts *website.ContainerListOptions) fs.WalkDirFunc {
	return opts.GetContainerData
}

func defaultContainersGenListWrite(opts *website.ContainerListOptions) error {
	return opts.WriteContainerList()
}

var defaultContainersGenListWalk = func(paths []string, fn fs.WalkDirFunc) error {
	if len(paths) == 0 {
		paths = []string{"./apps"}
	}
	for _, dir := range paths {
		if err := filepath.WalkDir(dir, fn); err != nil {
			return fmt.Errorf("error walking directory %s: %w", dir, err)
		}
	}
	return nil
}

var (
	containersGenListWalk           = defaultContainersGenListWalk
	containersGenListOptionsFactory = defaultContainersGenListOptionsFactory
	containersGenListGetData        = defaultContainersGenListGetData
	containersGenListWrite          = defaultContainersGenListWrite
	containersGenListRunner         = runContainersGenList
	containersGenListOnError        = func(err error) { log.Fatal().Err(err).Msg("container list generation failed") }
)

func runContainersGenList(args []string) error {
	opts := containersGenListOptionsFactory()
	if err := containersGenListWalk(args, containersGenListGetData(opts)); err != nil {
		return fmt.Errorf("failed to generate container list json file: %w", err)
	}

	if err := containersGenListWrite(opts); err != nil {
		return fmt.Errorf("failed to write container list json file: %w", err)
	}

	return nil
}

var genContainersListCmd = &cobra.Command{
	Use:     "gencontainerslist",
	Short:   "Generate containers list json file",
	Long:    containersGenListLongHelp,
	Example: "forgetool containers gencontainerslist <path to apps folder>",
	Run: func(cmd *cobra.Command, args []string) {
		if err := containersGenListRunner(args); err != nil {
			containersGenListOnError(err)
		}
	},
}

func init() {
	containersCmd.AddCommand(genContainersListCmd)
}

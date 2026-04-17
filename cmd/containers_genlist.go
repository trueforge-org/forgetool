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

var containerGenListLongHelp = strings.TrimSpace(`
Generates a JSON file listing all containers found under the given path(s).
Each container is identified by its docker-bake.hcl file.
`)

func defaultContainerGenListOptionsFactory() *website.ContainerListOptions {
	return &website.ContainerListOptions{
		OutputPath: "./containers.json",
	}
}

func defaultContainerGenListGetData(opts *website.ContainerListOptions) fs.WalkDirFunc {
	return opts.GetContainerData
}

func defaultContainerGenListWrite(opts *website.ContainerListOptions) error {
	return opts.WriteContainerList()
}

var defaultContainerGenListWalk = func(paths []string, fn fs.WalkDirFunc) error {
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
	containerGenListWalk           = defaultContainerGenListWalk
	containerGenListOptionsFactory = defaultContainerGenListOptionsFactory
	containerGenListGetData        = defaultContainerGenListGetData
	containerGenListWrite          = defaultContainerGenListWrite
	containerGenListRunner         = runContainerGenList
	containerGenListOnError        = func(err error) { log.Fatal().Err(err).Msg("container list generation failed") }
)

func runContainerGenList(args []string) error {
	opts := containerGenListOptionsFactory()
	if err := containerGenListWalk(args, containerGenListGetData(opts)); err != nil {
		return fmt.Errorf("failed to generate container list json file: %w", err)
	}

	if err := containerGenListWrite(opts); err != nil {
		return fmt.Errorf("failed to write container list json file: %w", err)
	}

	return nil
}

var genContainerListCmd = &cobra.Command{
	Use:     "gencontainerlist",
	Short:   "Generate container list json file",
	Long:    containerGenListLongHelp,
	Example: "forgetool containers gencontainerlist <path to apps folder>",
	Run: func(cmd *cobra.Command, args []string) {
		if err := containerGenListRunner(args); err != nil {
			containerGenListOnError(err)
		}
	},
}

func init() {
	containerCmd.AddCommand(genContainerListCmd)
}

package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/spf13/cobra"
	"github.com/trueforge-org/forgetool/pkg/changelog"
)

var containerGenChangelogLongHelp = strings.TrimSpace(`

`)

func defaultContainerGenChangelogGenerate(opts *changelog.ChangelogOptions) error {
	return opts.Generate()
}

func defaultContainerGenChangelogRender(opts *changelog.ChangelogOptions) error {
	return opts.Render()
}

var (
	containerGenChangelogGenerate = defaultContainerGenChangelogGenerate
	containerGenChangelogRender   = defaultContainerGenChangelogRender
	containerGenChangelogRunner   = runContainerGenChangelog
	containerGenChangelogOnError  = func(err error) { log.Fatal().Err(err) }
)

func runContainerGenChangelog(args []string) error {
	if len(args) < 3 {
		return errors.New("missing required arguments. please provide the repo path, template path and apps directory")
	}

	opts := &changelog.ChangelogOptions{
		RepoPath:                  args[0],
		TemplatePath:              args[1],
		AppsDir:                   args[2],
		ChangelogFileName:         "CHANGELOG.md",
		JSONOutputPath:            "./changelog.json",
		PrettyJSON:                true,
		StatusUpdateInterval:      5,
		SkipCommitsWithBadMessage: false,
		AppType:                   changelog.AppTypeContainer,
	}
	if err := containerGenChangelogGenerate(opts); err != nil {
		return fmt.Errorf("generate changelog: %w", err)
	}
	if err := containerGenChangelogRender(opts); err != nil {
		return fmt.Errorf("render changelog: %w", err)
	}

	return nil
}

var containerGenChangelogCmd = &cobra.Command{
	Use:     "genchangelog",
	Short:   "Generate changelog for containers",
	Long:    containerGenChangelogLongHelp,
	Example: "forgetool containers genchangelog <repo path> <template path> <apps dir>",
	Run: func(cmd *cobra.Command, args []string) {
		if err := containerGenChangelogRunner(args); err != nil {
			containerGenChangelogOnError(err)
		}
	},
}

func init() {
	containerCmd.AddCommand(containerGenChangelogCmd)
}

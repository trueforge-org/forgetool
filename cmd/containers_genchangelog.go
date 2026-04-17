package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/spf13/cobra"
	"github.com/trueforge-org/forgetool/pkg/changelog"
)

var containersGenChangelogLongHelp = strings.TrimSpace(`

`)

func defaultContainersGenChangelogGenerate(opts *changelog.ChangelogOptions) error {
	return opts.Generate()
}

func defaultContainersGenChangelogRender(opts *changelog.ChangelogOptions) error {
	return opts.Render()
}

var (
	containersGenChangelogGenerate = defaultContainersGenChangelogGenerate
	containersGenChangelogRender   = defaultContainersGenChangelogRender
	containersGenChangelogRunner   = runContainersGenChangelog
	containersGenChangelogOnError  = func(err error) { log.Fatal().Err(err) }
)

func runContainersGenChangelog(args []string) error {
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
	if err := containersGenChangelogGenerate(opts); err != nil {
		return fmt.Errorf("generate changelog: %w", err)
	}
	if err := containersGenChangelogRender(opts); err != nil {
		return fmt.Errorf("render changelog: %w", err)
	}

	return nil
}

var containersGenChangelogCmd = &cobra.Command{
	Use:     "genchangelog",
	Short:   "Generate changelog for containers",
	Long:    containersGenChangelogLongHelp,
	Example: "forgetool containers genchangelog <repo path> <template path> <apps dir>",
	Run: func(cmd *cobra.Command, args []string) {
		if err := containersGenChangelogRunner(args); err != nil {
			containersGenChangelogOnError(err)
		}
	},
}

func init() {
	containersCmd.AddCommand(containersGenChangelogCmd)
}

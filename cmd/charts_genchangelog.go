package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/spf13/cobra"
	"github.com/trueforge-org/forgetool/pkg/charts/changelog"
)

var chartsGenChangelogLongHelp = strings.TrimSpace(`

`)

var (
	chartsGenChangelogGenerate = func(opts *changelog.ChangelogOptions) error { return opts.Generate() }
	chartsGenChangelogRender   = func(opts *changelog.ChangelogOptions) error { return opts.Render() }
	chartsGenChangelogRunner   = runChartsGenChangelog
	chartsGenChangelogOnError  = func(err error) { log.Fatal().Err(err) }
)

func runChartsGenChangelog(args []string) error {
	if len(args) < 3 {
		return errors.New("missing required arguments. please provide the repo path, template path and charts directory")
	}

	opts := &changelog.ChangelogOptions{
		RepoPath:                  args[0],
		TemplatePath:              args[1],
		ChartsDir:                 args[2],
		ChangelogFileName:         "CHANGELOG.md",
		JSONOutputPath:            "./changelog.json",
		PrettyJSON:                true,
		StatusUpdateInterval:      5,
		SkipCommitsWithBadMessage: false,
	}
	if err := chartsGenChangelogGenerate(opts); err != nil {
		return fmt.Errorf("generate changelog: %w", err)
	}
	if err := chartsGenChangelogRender(opts); err != nil {
		return fmt.Errorf("render changelog: %w", err)
	}

	return nil
}

var genChangelogCmd = &cobra.Command{
	Use:     "genchangelog",
	Short:   "Generate changelog for charts",
	Long:    chartsGenChangelogLongHelp,
	Example: "forgetool charts genchangelog <repo path> <template path> <charts dir>",
	Run: func(cmd *cobra.Command, args []string) {
		if err := chartsGenChangelogRunner(args); err != nil {
			chartsGenChangelogOnError(err)
		}
	},
}

func init() {
	charts.AddCommand(genChangelogCmd)
}

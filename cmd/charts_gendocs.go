package cmd

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/trueforge-org/forgetool/pkg/website"
)

var chartsGenDocsLongHelp = strings.TrimSpace(`
Generates the website documentation pages for one or more Helm charts.

This is the Go replacement for the legacy .github/scripts/chart-docs.sh script
in truecharts. With no positional arguments, every chart under the charts
directory that has a Chart.yaml is processed; otherwise pass one or more
"train/chart" identifiers to limit the run.
`)

var (
	chartsGenDocsChartsDir     string
	chartsGenDocsWebsiteDir    string
	chartsGenDocsPrepare       bool
	chartsGenDocsChangelogsDir string

	chartsGenDocsRunner  = runChartsGenDocs
	chartsGenDocsOnError = func(err error) { log.Fatal().Err(err).Msg("chart docs generation failed") }
)

func runChartsGenDocs(args []string) error {
	baseOpts := website.ChartOptions{
		ChartsDir:  chartsGenDocsChartsDir,
		WebsiteDir: chartsGenDocsWebsiteDir,
	}

	if chartsGenDocsPrepare {
		if err := website.PrepareChartWebsite(baseOpts); err != nil {
			return fmt.Errorf("prepare website: %w", err)
		}
	}

	type pair struct{ train, chart string }
	var work []pair
	if len(args) == 0 {
		discovered, err := website.DiscoverCharts(chartsGenDocsChartsDir)
		if err != nil {
			return fmt.Errorf("discover charts: %w", err)
		}
		for _, p := range discovered {
			work = append(work, pair{train: p[0], chart: p[1]})
		}
	} else {
		for _, a := range args {
			parts := strings.SplitN(a, "/", 2)
			if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
				return fmt.Errorf("invalid chart identifier %q (expected train/chart)", a)
			}
			work = append(work, pair{train: parts[0], chart: parts[1]})
		}
	}

	for _, w := range work {
		opts := baseOpts
		opts.Train = w.train
		opts.Chart = w.chart
		if err := website.ProcessChart(opts); err != nil {
			return fmt.Errorf("process %s/%s: %w", w.train, w.chart, err)
		}
	}

	if chartsGenDocsChangelogsDir != "" {
		if err := website.FinalizeChartWebsite(baseOpts, chartsGenDocsChangelogsDir); err != nil {
			return fmt.Errorf("finalize website: %w", err)
		}
	}
	return nil
}

var chartsGenDocsCmd = &cobra.Command{
	Use:     "gendocs [train/chart...]",
	Short:   "Generate chart website docs",
	Long:    chartsGenDocsLongHelp,
	Example: "forgetool charts gendocs\nforgetool charts gendocs stable/sonarr",
	Run: func(cmd *cobra.Command, args []string) {
		if err := chartsGenDocsRunner(args); err != nil {
			chartsGenDocsOnError(err)
		}
	},
}

func init() {
	chartsGenDocsCmd.Flags().StringVar(&chartsGenDocsChartsDir, "charts-dir", "charts", "directory containing chart trains")
	chartsGenDocsCmd.Flags().StringVar(&chartsGenDocsWebsiteDir, "website-dir", "website", "root of the website checkout")
	chartsGenDocsCmd.Flags().BoolVar(&chartsGenDocsPrepare, "prepare", false, "prepare the website docs/charts tree (mkdir, wipe, restore index.mdx) before processing")
	chartsGenDocsCmd.Flags().StringVar(&chartsGenDocsChangelogsDir, "changelogs-dir", "", "if set, copy this directory's contents into docs/charts after processing")
	charts.AddCommand(chartsGenDocsCmd)
}

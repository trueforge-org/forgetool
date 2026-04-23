package website

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// ChartOptions configures a single chart docs build.
type ChartOptions struct {
	// Train is the train name (e.g. "stable", "incubator").
	Train string
	// Chart is the chart name.
	Chart string
	// ChartsDir is the root directory containing the trains. Defaults to
	// "charts".
	ChartsDir string
	// WebsiteDir is the root of the website checkout. Defaults to "website".
	WebsiteDir string
}

func (o *ChartOptions) applyDefaults() {
	if o.ChartsDir == "" {
		o.ChartsDir = "charts"
	}
	if o.WebsiteDir == "" {
		o.WebsiteDir = "website"
	}
}

type chartPaths struct {
	chartDir       string
	chartYaml      string
	docsSrc        string
	readme         string
	icon           string
	iconSmall      string
	screenshots    string
	docsBase       string
	tmpDocsBase    string
	docsDir        string
	indexFile      string
	iconsDir       string
	iconsSmallDir  string
	screenshotsDir string
}

func (o *ChartOptions) paths() chartPaths {
	docsBase := filepath.Join(o.WebsiteDir, "truecharts", "src", "content", "docs", "charts")
	tmpDocsBase := filepath.Join("tmpwebsite", "src", "content", "docs", "charts")
	chartDir := filepath.Join(o.ChartsDir, o.Train, o.Chart)
	return chartPaths{
		chartDir:       chartDir,
		chartYaml:      filepath.Join(chartDir, "Chart.yaml"),
		docsSrc:        filepath.Join(chartDir, "docs"),
		readme:         filepath.Join(chartDir, "README.md"),
		icon:           filepath.Join(chartDir, "icon.webp"),
		iconSmall:      filepath.Join(chartDir, "icon-small.webp"),
		screenshots:    filepath.Join(chartDir, "screenshots"),
		docsBase:       docsBase,
		tmpDocsBase:    tmpDocsBase,
		docsDir:        filepath.Join(docsBase, o.Train, o.Chart),
		indexFile:      filepath.Join(docsBase, o.Train, o.Chart, "index.md"),
		iconsDir:       filepath.Join(o.WebsiteDir, "truecharts", "public", "img", "hotlink-ok", "chart-icons"),
		iconsSmallDir:  filepath.Join(o.WebsiteDir, "truecharts", "public", "img", "hotlink-ok", "chart-icons-small"),
		screenshotsDir: filepath.Join(o.WebsiteDir, "truecharts", "public", "img", "hotlink-ok", "chart-screenshots", o.Chart),
	}
}

// chartMeta is the subset of Chart.yaml fields used in the index page.
type chartMeta struct {
	Version     string   `yaml:"version"`
	AppVersion  string   `yaml:"appVersion"`
	Description string   `yaml:"description"`
	Sources     []string `yaml:"sources"`
}

func readChartMeta(path string) (chartMeta, error) {
	var m chartMeta
	data, err := os.ReadFile(path)
	if err != nil {
		return m, err
	}
	if err := yaml.Unmarshal(data, &m); err != nil {
		return m, fmt.Errorf("parse %s: %w", path, err)
	}
	return m, nil
}

// ProcessChart regenerates the website documentation for a single chart. It is
// the Go equivalent of running .github/scripts/chart-docs.sh TRAIN/CHART.
func ProcessChart(opts ChartOptions) error {
	opts.applyDefaults()
	if opts.Train == "" || opts.Chart == "" {
		return errors.New("website: Train and Chart must be set")
	}
	p := opts.paths()

	if _, err := os.Stat(p.chartYaml); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Info().Msgf("chart path does not exist for %s/%s, skipping", opts.Train, opts.Chart)
			return nil
		}
		return err
	}

	log.Info().Msgf("copying docs to website for %s", opts.Chart)

	if err := keepDocsSafe(p.docsBase, p.tmpDocsBase, filepath.Join(opts.Train, opts.Chart)); err != nil {
		return err
	}
	if err := removeChartFromAllTrains(p.docsBase, opts.Chart); err != nil {
		return err
	}
	if err := os.MkdirAll(p.docsDir, 0o755); err != nil {
		return err
	}
	if err := copyChartAssets(p); err != nil {
		return err
	}
	if err := processChartIndex(opts, p); err != nil {
		return err
	}
	if err := restoreSafeDocs(p.docsBase, p.tmpDocsBase, filepath.Join(opts.Train, opts.Chart)); err != nil {
		return err
	}
	log.Info().Msgf("finished processing %s/%s", opts.Train, opts.Chart)
	return nil
}

// removeChartFromAllTrains removes any existing docs directory for chart in
// every train under docsBase. This mirrors `rm -rf $docs_base/*/${chart}` in
// the legacy script and lets a chart move trains cleanly.
func removeChartFromAllTrains(docsBase, chart string) error {
	entries, err := os.ReadDir(docsBase)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(docsBase, e.Name(), chart)
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
	}
	return nil
}

func copyChartAssets(p chartPaths) error {
	if err := copyTreeContents(p.docsSrc, p.docsDir); err != nil {
		return fmt.Errorf("copy docs: %w", err)
	}
	chart := filepath.Base(p.chartDir)
	if err := copyFileIfExists(p.icon, filepath.Join(p.iconsDir, chart+".webp")); err != nil {
		return err
	}
	if err := copyFileIfExists(p.iconSmall, filepath.Join(p.iconsSmallDir, chart+".webp")); err != nil {
		return err
	}
	if err := copyTreeContents(p.screenshots, p.screenshotsDir); err != nil {
		return fmt.Errorf("copy screenshots: %w", err)
	}
	return nil
}

func processChartIndex(opts ChartOptions, p chartPaths) error {
	meta, err := readChartMeta(p.chartYaml)
	if err != nil {
		return err
	}

	links, err := collectDocsLinks(p.docsDir)
	if err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString("---\n")
	fmt.Fprintf(&sb, "title: %s\n", opts.Chart)
	sb.WriteString("---\n\n")

	fmt.Fprintf(&sb,
		"![Version: %s](https://img.shields.io/badge/Version-%s-informational?style=flat-square) "+
			"![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) "+
			"![AppVersion: %s](https://img.shields.io/badge/AppVersion-%s-informational?style=flat-square)\n\n",
		meta.Version, meta.Version, meta.AppVersion, meta.AppVersion,
	)
	if meta.Description != "" {
		fmt.Fprintf(&sb, "%s\n\n", meta.Description)
	}

	sb.WriteString("## Chart Sources\n\n")
	if len(meta.Sources) > 0 {
		out, err := yaml.Marshal(meta.Sources)
		if err != nil {
			return err
		}
		sb.Write(out)
		sb.WriteString("\n")
	}

	sb.WriteString("## Available Documentation\n\n")
	for _, l := range links {
		fmt.Fprintf(&sb, "- [**%s**](./%s)\n", l.Title, l.Slug)
	}
	sb.WriteString("\n\n---\n\n")

	readme, err := readReadmeBody(p.readme, 3)
	if err != nil {
		return err
	}
	if strings.TrimSpace(readme) != "" {
		sb.WriteString("## Readme\n\n")
		sb.WriteString(readme)
		if !strings.HasSuffix(readme, "\n") {
			sb.WriteString("\n")
		}
	}

	if err := os.MkdirAll(filepath.Dir(p.indexFile), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p.indexFile, []byte(sb.String()), 0o644)
}

// DiscoverCharts walks chartsDir and returns every train/chart pair that has a
// Chart.yaml file. Useful for processing every chart in one invocation.
func DiscoverCharts(chartsDir string) ([][2]string, error) {
	trainEntries, err := os.ReadDir(chartsDir)
	if err != nil {
		return nil, err
	}
	var out [][2]string
	for _, te := range trainEntries {
		if !te.IsDir() {
			continue
		}
		chartEntries, err := os.ReadDir(filepath.Join(chartsDir, te.Name()))
		if err != nil {
			return nil, err
		}
		for _, ce := range chartEntries {
			if !ce.IsDir() {
				continue
			}
			yamlPath := filepath.Join(chartsDir, te.Name(), ce.Name(), "Chart.yaml")
			if _, err := os.Stat(yamlPath); err == nil {
				out = append(out, [2]string{te.Name(), ce.Name()})
			}
		}
	}
	return out, nil
}

package cmd

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/trueforge-org/forgetool/v4/pkg/website"
)

var containersGenDocsLongHelp = strings.TrimSpace(`
Generates the website documentation pages for one or more container apps.

This is the Go replacement for the legacy .github/scripts/container-docs.sh
script in containerforge. With no positional arguments, every app under the
apps directory that has a docker-bake.hcl is processed; otherwise only the
named apps are processed.
`)

var (
	containersGenDocsAppsDir             string
	containersGenDocsWebsiteDir          string
	containersGenDocsTemplatePath        string
	containersGenDocsComposeTemplatePath string
	containersGenDocsIconBaseURL         string
	containersGenDocsChangelogsDir       string

	containersGenDocsRunner          = runContainersGenDocs
	containersGenDocsOnError         = func(err error) { log.Fatal().Err(err).Msg("container docs generation failed") }
	containersGenDocsPrepareWebsite  = website.PrepareContainerWebsite
	containersGenDocsDiscoverApps    = website.DiscoverApps
	containersGenDocsProcessApp      = website.ProcessApp
	containersGenDocsFinalizeWebsite = website.FinalizeContainerWebsite
)

func runContainersGenDocs(args []string) error {
	baseOpts := website.ContainerOptions{
		AppsDir:             containersGenDocsAppsDir,
		WebsiteDir:          containersGenDocsWebsiteDir,
		TemplatePath:        containersGenDocsTemplatePath,
		ComposeTemplatePath: containersGenDocsComposeTemplatePath,
		IconFallbackBaseURL: containersGenDocsIconBaseURL,
	}

	if err := containersGenDocsPrepareWebsite(baseOpts); err != nil {
		return fmt.Errorf("prepare website: %w", err)
	}

	apps := args
	if len(apps) == 0 {
		discovered, err := containersGenDocsDiscoverApps(containersGenDocsAppsDir)
		if err != nil {
			return fmt.Errorf("discover apps: %w", err)
		}
		apps = discovered
	}

	for _, app := range apps {
		opts := baseOpts
		opts.App = app
		if err := containersGenDocsProcessApp(opts); err != nil {
			return fmt.Errorf("process %s: %w", app, err)
		}
	}

	if err := containersGenDocsFinalizeWebsite(baseOpts, containersGenDocsChangelogsDir); err != nil {
		return fmt.Errorf("finalize website: %w", err)
	}
	return nil
}

var containersGenDocsCmd = &cobra.Command{
	Use:     "gendocs [app...]",
	Short:   "Generate container app website docs",
	Long:    containersGenDocsLongHelp,
	Example: "forgetool containers gendocs\nforgetool containers gendocs sonarr radarr",
	Run: func(cmd *cobra.Command, args []string) {
		if err := containersGenDocsRunner(args); err != nil {
			containersGenDocsOnError(err)
		}
	},
}

func init() {
	containersGenDocsCmd.Flags().StringVar(&containersGenDocsAppsDir, "apps-dir", "apps", "directory containing app sources")
	containersGenDocsCmd.Flags().StringVar(&containersGenDocsWebsiteDir, "website-dir", "website", "root of the website checkout")
	containersGenDocsCmd.Flags().StringVar(&containersGenDocsTemplatePath, "template", "templates/README.md.tmpl", "index template path")
	containersGenDocsCmd.Flags().StringVar(&containersGenDocsComposeTemplatePath, "compose-template", "templates/docker-compose.md.tmpl", "docker-compose page template path")
	containersGenDocsCmd.Flags().StringVar(&containersGenDocsIconBaseURL, "icon-fallback-base-url", "", "base URL used to fetch icons when not present locally")
	containersGenDocsCmd.Flags().StringVar(&containersGenDocsChangelogsDir, "changelogs-dir", "changelogs", "directory whose contents are copied into docs/containers after processing (no-op if missing/empty)")
	containersCmd.AddCommand(containersGenDocsCmd)
}

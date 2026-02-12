package fluxhandler

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/trueforge-org/forgetool/pkg/helper"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/repo"
)

func newDefaultRegistryClient(plainHTTP bool, settings *cli.EnvSettings) (*registry.Client, error) {
	opts := []registry.ClientOption{
		registry.ClientOptDebug(settings.Debug),
		registry.ClientOptEnableCache(true),
		registry.ClientOptWriter(os.Stdout),
		registry.ClientOptCredentialsFile(settings.RegistryConfig),
	}
	if plainHTTP {
		opts = append(opts, registry.ClientOptPlainHTTP())
	}

	// Create a new registry client
	registryClient, err := registry.NewClient(opts...)
	if err != nil {
		return nil, err
	}
	return registryClient, nil
}

// HelmPull downloads a Helm chart from a repository
func HelmPull(repo string, name string, version string, dest string, silent bool) error {
	settings := cli.New()
	actionConfig := new(action.Configuration)

	// Define logger based on the silent parameter
	var logger func(string, ...interface{})
	if silent {
		logger = noOpLog
	} else {
		logger = log.Printf
	}

	// Initialize actionConfig with the appropriate logger
	if err := actionConfig.Init(settings.RESTClientGetter(), "", os.Getenv("HELM_DRIVER"), logger); err != nil {
		return fmt.Errorf("failed to initialize Helm action config: %w", err)
	}

	registryClient, err := newDefaultRegistryClient(false, settings)
	if err != nil {
		return err
	}
	actionConfig.RegistryClient = registryClient

	client := action.NewPullWithOpts(action.WithConfig(actionConfig))
	client.Settings = settings
	client.RepoURL = repo
	client.Version = version
	if dest != "" {
		client.DestDir = dest
	} else {
		client.DestDir = helper.HelmCache
	}

	// Create cache directory
	if err := os.MkdirAll(client.DestDir, os.ModePerm); err != nil {
		return fmt.Errorf("❌ Failed to create cache directory: %s", err)
	}
	configureHelmPullVerification(client, repo)

	link, repoURL := resolveHelmPullLink(repo, name)
	client.RepoURL = repoURL
	if strings.HasPrefix(repo, "http") {
		repoName := cleanRepoURL(repo)
		updateHelmRepo(repoName, repo, silent)
	}

	output, err := client.Run(link)

	if err != nil {
		os.Remove(path.Join(dest, fmt.Sprintf("%s-%s.tgz", name, version)))
		return err
	}
	if !silent {
		log.Info().Msg("✅ Dependency Downloaded!")
	}
	if client.Keyring != "" && client.Keyring != "nil" {
		if !silent {
			log.Info().Msg("✅ Dependency Verified")
		}
	}

	if output != "" {
		log.Info().Msgf("☸ Helm output: %s", output)
	}
	return nil
}

func configureHelmPullVerification(client *action.Pull, repo string) {
	switch repo {
	case "https://charts.trueforge.org",
		"https://library-charts.trueforge.org",
		"https://deps.trueforge.org":
		client.Keyring = helper.GpgDir + "/pubring.gpg"
		client.Verify = true
	case "https://charts.jetstack.io":
		client.Keyring = helper.GpgDir + "/certman.gpg"
		client.Verify = true
	}
}

func resolveHelmPullLink(repo, chartName string) (string, string) {
	if strings.HasPrefix(repo, "http") {
		return chartName, repo
	}

	return repo + "/" + chartName, ""
}

func noOpLog(format string, v ...interface{}) {}

func updateHelmRepo(name string, url string, silent bool) error {
	// Create a Helm repository configuration
	repoConfig := &repo.Entry{
		Name: name,
		URL:  url,
	}

	// Initialize Helm settings
	settings := cli.New()

	// Create a repository object
	r, err := repo.NewChartRepository(repoConfig, getter.All(settings))
	if err != nil {
		return fmt.Errorf("failed to create chart repository: %w", err)
	}

	// Ensure the repository cache directory exists
	cacheDir := settings.RepositoryCache
	if err := os.MkdirAll(cacheDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Download the latest index file
	if _, err := r.DownloadIndexFile(); err != nil {
		return fmt.Errorf("failed to download index file: %w", err)
	}

	// Load existing repositories file or create a new one
	repoFile := settings.RepositoryConfig
	repoFileContent := repo.NewFile()
	if _, err := os.Stat(repoFile); err == nil {
		repoFileContent, err = repo.LoadFile(repoFile)
		if err != nil {
			return fmt.Errorf("failed to load repositories file: %w", err)
		}
	}

	// Update the repositories file with the new repository
	if !repoFileContent.Has(name) {
		repoFileContent.Update(repoConfig)
	}

	if err := repoFileContent.WriteFile(repoFile, 0644); err != nil {
		return fmt.Errorf("failed to write repositories file: %w", err)
	}

	if !silent {
		log.Info().Msgf("Successfully updated repository '%s' from %s\n", name, url)
	}
	return nil
}

// cleanRepoURL performs the specified operations on the input URL
func cleanRepoURL(url string) string {
	// Remove http:// or https:// prefix
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")

	// Remove charts. prefix if present
	url = strings.TrimPrefix(url, "charts.")

	// Remove helm. prefix if present
	url = strings.TrimPrefix(url, "helm.")

	// Remove everything after the last dot
	lastDotIndex := strings.LastIndex(url, ".")
	if lastDotIndex != -1 {
		url = url[:lastDotIndex]
	}

	url = repoURL(url)

	return url
}

func repoURL(url string) string {
	parts := strings.SplitN(url, "/", 2) // Split into two parts at the first "/"
	if len(parts) > 0 {
		url = parts[0]
	}

	return url
}

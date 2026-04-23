package helmhandler

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

var (
	helmPullNewEnvSettingsFn   = cli.New
	helmPullActionConfigInitFn = func(actionConfig *action.Configuration, settings *cli.EnvSettings, logger func(string, ...interface{})) error {
		return actionConfig.Init(settings.RESTClientGetter(), "", os.Getenv("HELM_DRIVER"), logger)
	}
	helmPullNewDefaultRegistryClientFn = newDefaultRegistryClient
	helmPullNewPullWithOptsFn          = action.NewPullWithOpts
	helmPullMkdirAllFn                 = os.MkdirAll
	helmPullConfigureVerificationFn    = configureHelmPullVerification
	helmPullResolveLinkFn              = resolveHelmPullLink
	helmPullUpdateHelmRepoFn           = updateHelmRepo
	helmPullClientRunFn                = func(client *action.Pull, link string) (string, error) { return client.Run(link) }
	helmPullRemoveFn                   = os.Remove
	helmPullPathJoinFn                 = path.Join
	helmPullRepoNewChartRepositoryFn   = func(repoConfig *repo.Entry, settings *cli.EnvSettings) (*repo.ChartRepository, error) {
		return repo.NewChartRepository(repoConfig, getter.All(settings))
	}
	helmPullDownloadIndexFileFn = func(r *repo.ChartRepository) error {
		_, err := r.DownloadIndexFile()
		return err
	}
	helmPullRepoNewFileFn   = repo.NewFile
	helmPullRepoLoadFileFn  = repo.LoadFile
	helmPullRepoWriteFileFn = func(repoFileContent *repo.File, repoFile string) error {
		return repoFileContent.WriteFile(repoFile, 0644)
	}
	helmPullRegistryNewClientFn = registry.NewClient
	helmPullStatFn              = os.Stat
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
	registryClient, err := helmPullRegistryNewClientFn(opts...)
	if err != nil {
		return nil, err
	}
	return registryClient, nil
}

// HelmPull downloads a Helm chart from a repository
func HelmPull(repo string, name string, version string, dest string, silent bool) error {
	settings := helmPullNewEnvSettingsFn()
	actionConfig := new(action.Configuration)

	// Define logger based on the silent parameter
	var logger func(string, ...interface{})
	if silent {
		logger = noOpLog
	} else {
		logger = log.Printf
	}

	// Initialize actionConfig with the appropriate logger
	if err := helmPullActionConfigInitFn(actionConfig, settings, logger); err != nil {
		return fmt.Errorf("failed to initialize Helm action config: %w", err)
	}

	registryClient, err := helmPullNewDefaultRegistryClientFn(false, settings)
	if err != nil {
		return err
	}
	actionConfig.RegistryClient = registryClient

	client := helmPullNewPullWithOptsFn(action.WithConfig(actionConfig))
	client.Settings = settings
	client.RepoURL = repo
	client.Version = version
	if dest != "" {
		client.DestDir = dest
	} else {
		client.DestDir = helper.HelmCache
	}

	// Create cache directory
	if err := helmPullMkdirAllFn(client.DestDir, os.ModePerm); err != nil {
		return fmt.Errorf("❌ Failed to create cache directory: %s", err)
	}
	helmPullConfigureVerificationFn(client, repo)

	link, repoURL := helmPullResolveLinkFn(repo, name)
	client.RepoURL = repoURL
	if strings.HasPrefix(repo, "http") {
		repoName := cleanRepoURL(repo)
		helmPullUpdateHelmRepoFn(repoName, repo, silent)
	}

	output, err := helmPullClientRunFn(client, link)

	if err != nil {
		helmPullRemoveFn(helmPullPathJoinFn(dest, fmt.Sprintf("%s-%s.tgz", name, version)))
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

func noOpLog(format string, v ...interface{}) {
	_ = format
	_ = v
}

func updateHelmRepo(name string, url string, silent bool) error {
	// Create a Helm repository configuration
	repoConfig := &repo.Entry{
		Name: name,
		URL:  url,
	}

	// Initialize Helm settings
	settings := helmPullNewEnvSettingsFn()

	// Create a repository object
	r, err := helmPullRepoNewChartRepositoryFn(repoConfig, settings)
	if err != nil {
		return fmt.Errorf("failed to create chart repository: %w", err)
	}

	// Ensure the repository cache directory exists
	cacheDir := settings.RepositoryCache
	if err := helmPullMkdirAllFn(cacheDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Download the latest index file
	if err := helmPullDownloadIndexFileFn(r); err != nil {
		return fmt.Errorf("failed to download index file: %w", err)
	}

	// Load existing repositories file or create a new one
	repoFile := settings.RepositoryConfig
	repoFileContent := helmPullRepoNewFileFn()
	if _, err := helmPullStatFn(repoFile); err == nil {
		repoFileContent, err = helmPullRepoLoadFileFn(repoFile)
		if err != nil {
			return fmt.Errorf("failed to load repositories file: %w", err)
		}
	}

	// Update the repositories file with the new repository
	if !repoFileContent.Has(name) {
		repoFileContent.Update(repoConfig)
	}

	if err := helmPullRepoWriteFileFn(repoFileContent, repoFile); err != nil {
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

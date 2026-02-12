package fluxhandler

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/trueforge-org/forgetool/pkg/helper"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/kustomize/kyaml/yaml"
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

// HelmInstall installs a Helm chart with provided parameters
func HelmInstall(repoURL string, chartName string, releaseName string, namespace string, valuesFile string, version string, dryRun bool, wait bool, silent bool) error {
	if dryRun {
		log.Info().Msg("dryRun not possible...")
		return nil
	}

	settings, actionConfig, err := initHelmActionConfig(namespace, silent)
	if err != nil {
		return err
	}

	if err := ensureNamespace(actionConfig, namespace); err != nil {
		return fmt.Errorf("failed to ensure namespace exists: %w", err)
	}

	chartPath, err := resolveChartPath(repoURL, chartName, version)
	if err != nil {
		return err
	}

	chart, err := loader.Load(chartPath)
	if err != nil {
		return fmt.Errorf("failed to load chart: %w", err)
	}

	client := action.NewInstall(actionConfig)
	client.Namespace = namespace
	client.ReleaseName = releaseName
	client.DryRun = dryRun
	client.Version = version

	valueFiles, err := buildReleaseValueFiles(releaseName, valuesFile, chart.Values, false)
	if err != nil {
		return err
	}

	vals, err := mergeValueFiles(settings, valueFiles)
	if err != nil {
		return fmt.Errorf("failed to merge values: %w", err)
	}

	release, err := runInstallWithTimeoutRetry(client, chart, vals)
	if err != nil {
		return err
	}

	if wait {
		waitForRelease(actionConfig, release.Name, client.Namespace)
	}

	log.Printf("Installed Chart: %s in namespace: %s\n", release.Name, release.Namespace)
	log.Printf("Installed Chart values: %v\n", release.Config)

	return nil
}

func runInstallWithTimeoutRetry(client *action.Install, chart *chart.Chart, vals map[string]interface{}) (*release.Release, error) {
	log.Debug().Msg("Installing chart...")
	rel, err := client.Run(chart, vals)
	if err == nil {
		return rel, nil
	}

	log.Debug().Msg("Chart install returned an error")
	if !strings.Contains(err.Error(), "timed out") {
		return nil, fmt.Errorf("failed to install chart: %w", err)
	}

	log.Warn().Msg("Chart install recieved a timeout, retrying in 15 seconds...")
	time.Sleep(15 * time.Second)
	rel, err = client.Run(chart, vals)
	if err == nil {
		return rel, nil
	}
	if strings.Contains(err.Error(), "timed out") {
		return nil, fmt.Errorf("failed to install chart after retry, with another timeout: %w", err)
	}

	return nil, fmt.Errorf("failed to install chart after retry: %w", err)
}

func ensureNamespace(actionConfig *action.Configuration, namespace string) error {
	// Check if the namespace exists
	exists, err := namespaceExists(actionConfig, namespace)
	if err != nil {
		return fmt.Errorf("failed to check if namespace exists: %w", err)
	}

	if !exists {
		// Create the namespace if it does not exist
		err := createNamespace(actionConfig, namespace)
		if err != nil {
			return fmt.Errorf("failed to create namespace: %w", err)
		}
	}

	return nil
}

// HelmUpgrade upgrades a Helm release with provided parameters
// HelmUpgrade upgrades a Helm release with provided parameters
func HelmUpgrade(repoURL string, chartName string, releaseName string, namespace string, valuesFile string, version string, wait bool, silent bool) error {
	settings, actionConfig, err := initHelmActionConfig(namespace, silent)
	if err != nil {
		return err
	}

	if err := ensureNamespace(actionConfig, namespace); err != nil {
		return fmt.Errorf("failed to ensure namespace exists: %w", err)
	}

	chartPath, err := resolveChartPath(repoURL, chartName, version)
	if err != nil {
		return err
	}

	chart, err := loader.Load(chartPath)
	if err != nil {
		return fmt.Errorf("failed to load chart: %w", err)
	}

	client := action.NewUpgrade(actionConfig)
	client.Namespace = namespace
	client.Version = version

	valueFiles, err := buildReleaseValueFiles(releaseName, valuesFile, chart.Values, true)
	if err != nil {
		return err
	}

	vals, err := mergeValueFiles(settings, valueFiles)
	if err != nil {
		return fmt.Errorf("failed to merge values: %w", err)
	}

	// Perform the upgrade with merged values
	release, err := client.Run(releaseName, chart, vals)
	if err != nil {
		return fmt.Errorf("failed to upgrade chart: %w", err)
	}

	if wait {
		waitForRelease(actionConfig, release.Name, client.Namespace)
	}

	log.Printf("Upgraded Chart: %s in namespace: %s\n", release.Name, release.Namespace)
	log.Printf("Upgraded Chart values: %v\n", release.Config)

	return nil
}

func initHelmActionConfig(namespace string, silent bool) (*cli.EnvSettings, *action.Configuration, error) {
	settings := cli.New()
	settings.SetNamespace(namespace)

	var logger func(string, ...interface{})
	if silent {
		logger = noOpLog
	} else {
		logger = log.Printf
	}

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), logger); err != nil {
		return nil, nil, fmt.Errorf("failed to initialize Helm action config: %w", err)
	}

	return settings, actionConfig, nil
}

func resolveChartPath(repoURL string, chartName string, version string) (string, error) {
	if strings.HasPrefix(repoURL, "http://") || strings.HasPrefix(repoURL, "https://") || strings.HasPrefix(repoURL, "oci://") {
		err := HelmPull(repoURL, chartName, version, "", true)
		if err != nil {
			return "", fmt.Errorf("failed to pull chart %s: %w", chartName, err)
		}
		return path.Join(helper.HelmCache, fmt.Sprintf("%s-%s.tgz", chartName, version)), nil
	}

	return repoURL, nil
}

func buildReleaseValueFiles(releaseName string, valuesFile string, chartValues map[string]interface{}, strictHelmRelease bool) ([]string, error) {
	tempValuesPath := path.Join(helper.HelmCache, releaseName+"tempvalues.yaml")
	err := createValuesYAML(chartValues, tempValuesPath)
	if err != nil {
		return nil, fmt.Errorf("error creating tempvalues.yaml: %w", err)
	}

	valueFiles := []string{tempValuesPath}
	directory := filepath.Dir(valuesFile)
	helmreleasePath := path.Join(directory, "helm-release.yaml")

	helmRelease, err := LoadHelmRelease(helmreleasePath)
	if err != nil {
		if strictHelmRelease {
			return nil, fmt.Errorf("error loading helm-release.yaml: %w", err)
		}
	} else {
		tempHRValuesPath := path.Join(helper.HelmCache, releaseName+"temphrvalues.yaml")
		err = createValuesYAML(helmRelease.Spec.Values, tempHRValuesPath)
		if err != nil {
			return nil, fmt.Errorf("error creating temphrvalues.yaml: %w", err)
		}
		helper.EnvSubst(tempHRValuesPath, helper.TalEnv)
		valueFiles = append(valueFiles, tempHRValuesPath)
	}

	if _, err = os.Stat(valuesFile); err == nil {
		valueFiles = append(valueFiles, valuesFile)
	}

	overrideValuesPath := path.Join(directory, "bootstrap-values.yaml.ct")
	if _, err = os.Stat(overrideValuesPath); err == nil {
		valueFiles = append(valueFiles, overrideValuesPath)
	}

	return valueFiles, nil
}

func mergeValueFiles(settings *cli.EnvSettings, valueFiles []string) (map[string]interface{}, error) {
	valOpts := &values.Options{ValueFiles: valueFiles}
	valProviders := getter.All(settings)
	return valOpts.MergeValues(valProviders)
}

func namespaceExists(actionConfig *action.Configuration, namespace string) (bool, error) {
	// Retrieve Kubernetes client set from actionConfig
	clientset, err := actionConfig.KubernetesClientSet()
	if err != nil {
		return false, fmt.Errorf("failed to get Kubernetes client set: %w", err)
	}

	// Use clientset to check if the namespace exists
	_, err = clientset.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
	if err != nil {
		return false, nil // Namespace doesn't exist or other error occurred
	}
	return true, nil // Namespace exists
}

func createNamespace(actionConfig *action.Configuration, namespace string) error {
	// Retrieve Kubernetes client set from actionConfig
	clientset, err := actionConfig.KubernetesClientSet()
	if err != nil {
		return fmt.Errorf("failed to get Kubernetes client set: %w", err)
	}

	// Create the namespace using clientset
	_, err = clientset.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {

		} else {
			return fmt.Errorf("failed to create namespace: %w", err)
		}
	}
	return nil
}

func createValuesYAML(values map[string]interface{}, fileName string) error {
	removeFileIfExists(fileName)
	// Marshal values map into YAML format
	data, err := yaml.Marshal(values)
	if err != nil {
		return err
	}

	// Write YAML data into the file
	err = ioutil.WriteFile(fileName, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

func removeFileIfExists(fileName string) error {
	// Check if the file exists
	_, err := os.Stat(fileName)
	if err == nil {
		// Delete the file
		err = os.Remove(fileName)
		if err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		// Return other errors if the file check failed for a reason other than not existing
		return err
	}

	return nil
}

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

func waitForRelease(actionConfig *action.Configuration, releaseName, namespace string) {
	statusClient := action.NewStatus(actionConfig)
	for {
		rel, err := statusClient.Run(releaseName)
		if err != nil {
			log.Info().Msgf("failed to get release status: %v", err)
		}
		if rel.Info.Status == release.StatusDeployed {
			log.Info().Msgf("Release %s is now deployed\n", releaseName)
			break
		}
		log.Info().Msgf("Waiting for release %s to be deployed (current status: %s)\n", releaseName, rel.Info.Status)
		time.Sleep(5 * time.Second)
	}
}

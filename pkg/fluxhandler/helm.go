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
	"helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	helmInitHelmActionConfigFn       = initHelmActionConfig
	helmEnsureNamespaceFn            = ensureNamespace
	helmResolveChartPathFn           = resolveChartPath
	helmLoaderLoadFn                 = loader.Load
	helmBuildReleaseValueFilesFn     = buildReleaseValueFiles
	helmMergeValueFilesFn            = mergeValueFiles
	helmRunInstallWithTimeoutRetryFn = runInstallWithTimeoutRetry
	helmWaitForReleaseFn             = waitForRelease
	helmNewInstallFn                 = action.NewInstall
	helmNewUpgradeFn                 = action.NewUpgrade
	helmUpgradeRunFn                 = func(client *action.Upgrade, releaseName string, ch *chart.Chart, vals map[string]interface{}) (*release.Release, error) {
		return client.Run(releaseName, ch, vals)
	}
	helmActionConfigInitFn = func(actionConfig *action.Configuration, settings *cli.EnvSettings, namespace string, logger func(string, ...interface{})) error {
		return actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), logger)
	}
	helmHelmPullFn         = HelmPull
	helmCreateValuesYAMLFn = createValuesYAML
	helmLoadHelmReleaseFn  = LoadHelmRelease
	helmEnvSubstFn         = helper.EnvSubst
	helmOSStatFn           = os.Stat
	helmValuesMergeFn      = func(valOpts *values.Options, settings *cli.EnvSettings) (map[string]interface{}, error) {
		return valOpts.MergeValues(getter.All(settings))
	}
	helmGetKubernetesClientSetFn = func(actionConfig *action.Configuration) (kubernetes.Interface, error) {
		return actionConfig.KubernetesClientSet()
	}
	helmNamespaceGetFn = func(clientset kubernetes.Interface, namespace string) error {
		_, err := clientset.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
		return err
	}
	helmNamespaceCreateFn = func(clientset kubernetes.Interface, namespace string) error {
		_, err := clientset.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}, metav1.CreateOptions{})
		return err
	}
	helmYAMLMarshalFn  = yaml.Marshal
	helmWriteFileFn    = ioutil.WriteFile
	helmNewStatusRunFn = func(actionConfig *action.Configuration, releaseName string) (*release.Release, error) {
		return action.NewStatus(actionConfig).Run(releaseName)
	}
	helmSleepFn      = time.Sleep
	helmInstallRunFn = func(client *action.Install, ch *chart.Chart, vals map[string]interface{}) (*release.Release, error) {
		return client.Run(ch, vals)
	}
)

// HelmInstall installs a Helm chart with provided parameters
func HelmInstall(repoURL string, chartName string, releaseName string, namespace string, valuesFile string, version string, dryRun bool, wait bool, silent bool) error {
	if dryRun {
		log.Info().Msg("dryRun not possible...")
		return nil
	}

	settings, actionConfig, err := helmInitHelmActionConfigFn(namespace, silent)
	if err != nil {
		return err
	}

	if err := helmEnsureNamespaceFn(actionConfig, namespace); err != nil {
		return fmt.Errorf("failed to ensure namespace exists: %w", err)
	}

	chartPath, err := helmResolveChartPathFn(repoURL, chartName, version)
	if err != nil {
		return err
	}

	chart, err := helmLoaderLoadFn(chartPath)
	if err != nil {
		return fmt.Errorf("failed to load chart: %w", err)
	}

	client := helmNewInstallFn(actionConfig)
	client.Namespace = namespace
	client.ReleaseName = releaseName
	client.DryRun = dryRun
	client.Version = version

	valueFiles, err := helmBuildReleaseValueFilesFn(releaseName, valuesFile, chart.Values, false)
	if err != nil {
		return err
	}

	vals, err := helmMergeValueFilesFn(settings, valueFiles)
	if err != nil {
		return fmt.Errorf("failed to merge values: %w", err)
	}

	release, err := helmRunInstallWithTimeoutRetryFn(client, chart, vals)
	if err != nil {
		return err
	}

	if wait {
		helmWaitForReleaseFn(actionConfig, release.Name, client.Namespace)
	}

	log.Printf("Installed Chart: %s in namespace: %s\n", release.Name, release.Namespace)
	log.Printf("Installed Chart values: %v\n", release.Config)

	return nil
}

func runInstallWithTimeoutRetry(client *action.Install, chart *chart.Chart, vals map[string]interface{}) (*release.Release, error) {
	log.Debug().Msg("Installing chart...")
	rel, err := helmInstallRunFn(client, chart, vals)
	if err == nil {
		return rel, nil
	}

	log.Debug().Msg("Chart install returned an error")
	if !strings.Contains(err.Error(), "timed out") {
		return nil, fmt.Errorf("failed to install chart: %w", err)
	}

	log.Warn().Msg("Chart install recieved a timeout, retrying in 15 seconds...")
	helmSleepFn(15 * time.Second)
	rel, err = helmInstallRunFn(client, chart, vals)
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
	settings, actionConfig, err := helmInitHelmActionConfigFn(namespace, silent)
	if err != nil {
		return err
	}

	if err := helmEnsureNamespaceFn(actionConfig, namespace); err != nil {
		return fmt.Errorf("failed to ensure namespace exists: %w", err)
	}

	chartPath, err := helmResolveChartPathFn(repoURL, chartName, version)
	if err != nil {
		return err
	}

	chart, err := helmLoaderLoadFn(chartPath)
	if err != nil {
		return fmt.Errorf("failed to load chart: %w", err)
	}

	client := helmNewUpgradeFn(actionConfig)
	client.Namespace = namespace
	client.Version = version

	valueFiles, err := helmBuildReleaseValueFilesFn(releaseName, valuesFile, chart.Values, true)
	if err != nil {
		return err
	}

	vals, err := helmMergeValueFilesFn(settings, valueFiles)
	if err != nil {
		return fmt.Errorf("failed to merge values: %w", err)
	}

	// Perform the upgrade with merged values
	release, err := helmUpgradeRunFn(client, releaseName, chart, vals)
	if err != nil {
		return fmt.Errorf("failed to upgrade chart: %w", err)
	}

	if wait {
		helmWaitForReleaseFn(actionConfig, release.Name, client.Namespace)
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
	if err := helmActionConfigInitFn(actionConfig, settings, namespace, logger); err != nil {
		return nil, nil, fmt.Errorf("failed to initialize Helm action config: %w", err)
	}

	return settings, actionConfig, nil
}

func resolveChartPath(repoURL string, chartName string, version string) (string, error) {
	if strings.HasPrefix(repoURL, "http://") || strings.HasPrefix(repoURL, "https://") || strings.HasPrefix(repoURL, "oci://") {
		err := helmHelmPullFn(repoURL, chartName, version, "", true)
		if err != nil {
			return "", fmt.Errorf("failed to pull chart %s: %w", chartName, err)
		}
		return path.Join(helper.HelmCache, fmt.Sprintf("%s-%s.tgz", chartName, version)), nil
	}

	return repoURL, nil
}

func buildReleaseValueFiles(releaseName string, valuesFile string, chartValues map[string]interface{}, strictHelmRelease bool) ([]string, error) {
	tempValuesPath := path.Join(helper.HelmCache, releaseName+"tempvalues.yaml")
	err := helmCreateValuesYAMLFn(chartValues, tempValuesPath)
	if err != nil {
		return nil, fmt.Errorf("error creating tempvalues.yaml: %w", err)
	}

	valueFiles := []string{tempValuesPath}
	directory := filepath.Dir(valuesFile)
	helmreleasePath := path.Join(directory, "helm-release.yaml")

	helmRelease, err := helmLoadHelmReleaseFn(helmreleasePath)
	if err != nil {
		if strictHelmRelease {
			return nil, fmt.Errorf("error loading helm-release.yaml: %w", err)
		}
	} else {
		tempHRValuesPath := path.Join(helper.HelmCache, releaseName+"temphrvalues.yaml")
		err = helmCreateValuesYAMLFn(helmRelease.Spec.Values, tempHRValuesPath)
		if err != nil {
			return nil, fmt.Errorf("error creating temphrvalues.yaml: %w", err)
		}
		helmEnvSubstFn(tempHRValuesPath, helper.TalEnv)
		valueFiles = append(valueFiles, tempHRValuesPath)
	}

	if _, err = helmOSStatFn(valuesFile); err == nil {
		valueFiles = append(valueFiles, valuesFile)
	}

	overrideValuesPath := path.Join(directory, "bootstrap-values.yaml.ct")
	if _, err = helmOSStatFn(overrideValuesPath); err == nil {
		valueFiles = append(valueFiles, overrideValuesPath)
	}

	return valueFiles, nil
}

func mergeValueFiles(settings *cli.EnvSettings, valueFiles []string) (map[string]interface{}, error) {
	valOpts := &values.Options{ValueFiles: valueFiles}
	return helmValuesMergeFn(valOpts, settings)
}

func namespaceExists(actionConfig *action.Configuration, namespace string) (bool, error) {
	// Retrieve Kubernetes client set from actionConfig
	clientset, err := helmGetKubernetesClientSetFn(actionConfig)
	if err != nil {
		return false, fmt.Errorf("failed to get Kubernetes client set: %w", err)
	}

	// Use clientset to check if the namespace exists
	err = helmNamespaceGetFn(clientset, namespace)
	if err != nil {
		return false, nil // Namespace doesn't exist or other error occurred
	}
	return true, nil // Namespace exists
}

func createNamespace(actionConfig *action.Configuration, namespace string) error {
	// Retrieve Kubernetes client set from actionConfig
	clientset, err := helmGetKubernetesClientSetFn(actionConfig)
	if err != nil {
		return fmt.Errorf("failed to get Kubernetes client set: %w", err)
	}

	// Create the namespace using clientset
	err = helmNamespaceCreateFn(clientset, namespace)
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
	data, err := helmYAMLMarshalFn(values)
	if err != nil {
		return err
	}

	// Write YAML data into the file
	err = helmWriteFileFn(fileName, data, 0644)
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

func waitForRelease(actionConfig *action.Configuration, releaseName, namespace string) {
	for {
		rel, err := helmNewStatusRunFn(actionConfig, releaseName)
		if err != nil {
			log.Info().Msgf("failed to get release status: %v", err)
		}
		if rel.Info.Status == release.StatusDeployed {
			log.Info().Msgf("Release %s is now deployed\n", releaseName)
			break
		}
		log.Info().Msgf("Waiting for release %s to be deployed (current status: %s)\n", releaseName, rel.Info.Status)
		helmSleepFn(5 * time.Second)
	}
}

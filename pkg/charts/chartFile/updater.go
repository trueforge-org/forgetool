package chartFile

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/trueforge-org/forgetool/v4/pkg/charts/helmignore"
	"github.com/trueforge-org/forgetool/v4/pkg/charts/image"
	"github.com/trueforge-org/forgetool/v4/pkg/readme"
	"github.com/trueforge-org/forgetool/v4/pkg/version"
)

var updateSourcesFunc = updateSources
var saveChartFunc = func(chart *HelmChart, chartPath string) error {
	return chart.SaveToFile(chartPath)
}

// UpdateChartFile updates the specified Chart.yaml file with an optional bump parameter.
func UpdateChartFile(chartPathOrFolder, bump string) error {
	chartPath, err := resolveChartPath(chartPathOrFolder)
	if err != nil {
		return err
	}

	log.Info().Msgf("🏃 Processing chart [%s]", chartPath)
	chart := NewHelmChart()
	if err := chart.LoadFromFile(chartPath); err != nil {
		return err
	}

	initializeAnnotations(chart)
	train := GetTrain(chartPath, chart)
	setMetadata(chart, train)

	values, imageLinks, err := loadChartImages(chartPath)
	if err != nil {
		return err
	}
	setAppVersionFromImage(chart, values, "image")

	// Attempt to update sources
	if err := updateSourcesFunc(chart, train, imageLinks); err != nil {
		return err
	}

	// Update appVersion, icon, and home URLs
	bumpChartVersion(chart, bump)

	// Save the modified metadata back to the file
	if err := saveChartFunc(chart, chartPath); err != nil {
		return fmt.Errorf("error saving Chart.yaml: %s", err)
	}

	log.Info().Msgf("Chart file updated and saved to [%s]", chartPath)

	if err := generateChartArtifacts(chartPath, chart.Metadata.Name, train); err != nil {
		return err
	}
	return nil
}

func initializeAnnotations(chart *HelmChart) {
	if chart.Metadata.Annotations == nil {
		chart.Metadata.Annotations = make(map[string]string)
	}
}

func loadChartImages(chartPath string) (*image.Images, []string, error) {
	var values image.Images
	if err := values.LoadValuesFile(filepath.Join(filepath.Dir(chartPath), "values.yaml")); err != nil {
		return nil, nil, err
	}

	var imageLinks []string
	for _, details := range values.ImagesMap {
		imageLinks = append(imageLinks, details.Link)
	}

	return &values, imageLinks, nil
}

func bumpChartVersion(chart *HelmChart, bump string) {
	if bump != version.Major && bump != version.Minor && bump != version.Patch {
		return
	}

	newVersion, err := version.IncrementVersion(chart.Metadata.Version, bump)
	log.Info().Msgf("🆚 Bumping [%s], from [%s] to [%s]", chart.Metadata.Name, chart.Metadata.Version, newVersion)
	if err != nil {
		log.Error().Err(err).Msg("Error bumping version")
	}
	chart.Metadata.Version = newVersion
}

func resolveChartPath(chartPathOrFolder string) (string, error) {
	fileInfo, err := os.Stat(chartPathOrFolder)
	if err != nil {
		return "", err
	}

	if fileInfo.IsDir() {
		return filepath.Join(chartPathOrFolder, "Chart.yaml"), nil
	}

	return chartPathOrFolder, nil
}

func generateChartArtifacts(chartPath, chartName, train string) error {

	templateDir := chartPath
	for i := 0; i < 4; i++ {
		templateDir = filepath.Dir(templateDir)
	}

	// Generate README.md for the specified train and chart
	readmeErr := readme.GenerateReadme(templateDir, chartPath, chartName, train)
	if readmeErr != nil {
		log.Info().Msgf("Error Generating readme for %v: %v\n", chartName, readmeErr)
		return fmt.Errorf("error generating readme for %s: %w", chartName, readmeErr)
	}

	// Generate .helmignore for the specified train and chart
	helmignoreErr := helmignore.GenerateHelmIgnore(templateDir, chartPath)
	if helmignoreErr != nil {
		log.Info().Msgf("Error Generating helmignore for %v: %v\n", chartName, helmignoreErr)
		return fmt.Errorf("error generating helmignore for %s: %w", chartName, helmignoreErr)
	}

	return nil
}

func setAppVersionFromImage(chart *HelmChart, imageMap *image.Images, key string) {
	imageDetails, exists := imageMap.ImagesMap[key]
	if !exists {
		log.Warn().Msgf("Details for image key [%s] not found in values.yaml, skipping setting appVersion", key)
		return
	}

	log.Info().Msgf("Detected - Tag [%s], Image Version [%s]", imageDetails.Tag, imageDetails.Version)
	chart.Metadata.AppVersion = imageDetails.Version
}

// detectTrainFromFile detects the train name based on the path of Chart.yaml.
func detectTrainFromFile(chartFilename string) string {
	parts := strings.Split(
		// Remove the filename from the path
		filepath.Dir(chartFilename),
		string(os.PathSeparator),
	)

	if len(parts) >= 2 {
		return parts[len(parts)-2]
	}

	// One case to reach here is when the tool is run from inside the train directory
	// But we can't safely assume the current directory is the train directory
	log.Error().Msgf("Unable to detect train from path [%s]", chartFilename)
	return ""
}

func GetTrain(chartPath string, chart *HelmChart) string {
	// Detect the train from the path of Chart.yaml
	// Do not rely on the annotations in the chart, as they may be outdated
	train := detectTrainFromFile(chartPath)
	if train == "" {
		// If the train cannot be detected from the path, fallback to detect it from the annotations
		if val, exists := chart.Metadata.Annotations["trueforge.org/train"]; exists {
			train = val
		} else {
			log.Error().Msgf("Unable to detect train for chart [%s]. Setting as [unknown]", chart.Metadata.Name)
			train = "unknown"
		}
	}

	return train
}

func setMetadata(chart *HelmChart, train string) {
	chart.Metadata.Annotations["truecharts.org/train"] = train
	chart.Metadata.Icon = fmt.Sprintf("https://truecharts.org/img/hotlink-ok/chart-icons/%s.webp", chart.Metadata.Name)
	chart.Metadata.Home = fmt.Sprintf("https://truecharts.org/charts/%s/%s", train, chart.Metadata.Name)
}

// UpdateSources updates the sources in Chart.yaml using Go.
func updateSources(chart *HelmChart, train string, imageLinks []string) error {
	var updatedSources []string

	for _, source := range chart.Metadata.Sources {
		if shouldKeepSource(source) {
			log.Info().Msgf("🔗 Keeping source [%s]", source)
			updatedSources = append(updatedSources, source)
		}
	}

	// Add the GitHub source for the chart
	ghSource := fmt.Sprintf("https://github.com/trueforge-org/truecharts/tree/master/charts/%s/%s", train, chart.Metadata.Name)
	updatedSources = append(updatedSources, ghSource)

	// Add new sources for each image
	updatedSources = append(updatedSources, imageLinks...)

	finalSources := deduplicateSources(updatedSources)

	// Sort the sources, so subsequent commits will only include actual changes
	slices.Sort(finalSources)

	// Update the chart's sources
	chart.Metadata.Sources = finalSources

	return nil
}

func shouldKeepSource(source string) bool {
	if source == "" {
		return false
	}

	if strings.HasPrefix(source, "https://ghcr") ||
		strings.HasPrefix(source, "https://docker.io") ||
		strings.HasPrefix(source, "https://hub.docker") ||
		strings.HasPrefix(source, "https://fleet.linuxserver") ||
		strings.HasPrefix(source, "https://mcr.microsoft") ||
		strings.HasPrefix(source, "https://cr.hotio.dev") ||
		strings.HasPrefix(source, "https://github.com/trueforge-org") ||
		strings.HasPrefix(source, "https://github.com/truecharts") ||
		strings.HasPrefix(source, "https://gallery.ecr.aws") ||
		strings.HasPrefix(source, "https://gcr") ||
		strings.HasPrefix(source, "https://quay") ||
		strings.HasPrefix(source, "http://") ||
		strings.Contains(source, ".azurecr.io") ||
		strings.Contains(source, ".ocir.io") {
		return false
	}

	return true
}

func deduplicateSources(updatedSources []string) []string {
	deduplicatedSources := make(map[string]bool)
	var finalSources []string
	for _, source := range updatedSources {
		if source == "" {
			continue
		}
		if _, exists := deduplicatedSources[source]; exists {
			continue
		}

		deduplicatedSources[source] = true
		finalSources = append(finalSources, source)
	}

	return finalSources
}

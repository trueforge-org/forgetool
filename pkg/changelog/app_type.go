package changelog

import "fmt"

// AppType identifies whether the changelog is being generated for charts or containers.
type AppType string

const (
	AppTypeChart     AppType = "chart"
	AppTypeContainer AppType = "container"
)

// Function variables that vary by app type. Defaults are set to chart implementations.
var (
	getAppNameFunc            = chartGetAppName
	getAppTrainFunc           = chartGetAppTrain
	getVersionFromContentFunc = chartGetVersion
	isPreferredFileFunc       = chartIsPreferredFile
	renderOutputPathFunc      = chartRenderOutputPath
	activeAppsManifestFile    = chartManifestFile
	parseActiveAppFunc        = chartParseActiveApp
)

func configureForAppType(appType AppType) error {
	switch appType {
	case AppTypeContainer:
		getAppNameFunc = containerGetAppName
		getAppTrainFunc = containerGetAppTrain
		getVersionFromContentFunc = containerGetVersion
		isPreferredFileFunc = containerIsPreferredFile
		renderOutputPathFunc = containerRenderOutputPath
		activeAppsManifestFile = containerManifestFile
		parseActiveAppFunc = containerParseActiveApp
		getAppPathFunc = containerGetAppPath
	case AppTypeChart, "":
		getAppNameFunc = chartGetAppName
		getAppTrainFunc = chartGetAppTrain
		getVersionFromContentFunc = chartGetVersion
		isPreferredFileFunc = chartIsPreferredFile
		renderOutputPathFunc = chartRenderOutputPath
		activeAppsManifestFile = chartManifestFile
		parseActiveAppFunc = chartParseActiveApp
		getAppPathFunc = chartGetAppPath
	default:
		return fmt.Errorf("unknown app type: %s", appType)
	}
	return nil
}

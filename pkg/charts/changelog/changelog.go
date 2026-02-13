package changelog

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/go-git/go-git/v5"
	gitobject "github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rs/zerolog/log"
	"github.com/trueforge-org/forgetool/pkg/helper"
)

type ChangelogOptions struct {
	RepoPath                  string // Path to the repository (eg "./charts")
	TemplatePath              string // Path to the template file (eg "./changelog.tmpl")
	ChangelogFileName         string // Name of the changelog file eg "CHANGELOG.md"
	JSONOutputPath            string // Path to the JSON output file
	PrettyJSON                bool   // If true, the JSON output will be pretty-printed
	ChartsDir                 string // Dir where the charts are located (eg "./charts/")
	StatusUpdateInterval      int    // Interval in seconds between status updates
	SkipCommitsWithBadMessage bool   // If true, commits with bad messages will be skipped
}

func checkPath(path string, createIfNotExist bool) error {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			if createIfNotExist {
				_, err := os.Create(path)
				if err != nil {
					return fmt.Errorf("cannot create path %s: %w", path, err)
				}
				return nil
			}
			return nil
		}
		return fmt.Errorf("path %s cannot be used: %w", path, err)
	}
	return nil
}

func (o *ChangelogOptions) validate() error {
	if o.RepoPath == "" {
		return fmt.Errorf("repo path is empty")
	}
	if o.TemplatePath == "" {
		return fmt.Errorf("template path is empty")
	}
	if o.ChangelogFileName == "" {
		return fmt.Errorf("changelog file name is empty")
	}
	if o.ChartsDir == "" {
		return fmt.Errorf("charts dir is empty")
	}
	if o.JSONOutputPath == "" {
		return fmt.Errorf("json output path is empty")
	}
	if o.StatusUpdateInterval <= 0 {
		return fmt.Errorf("status update interval is zero")
	}

	paths := map[string]bool{
		o.TemplatePath:   false,
		o.RepoPath:       false,
		o.ChartsDir:      false,
		o.JSONOutputPath: false,
	}

	for path, create := range paths {
		if err := checkPath(path, create); err != nil {
			return err
		}
	}

	return nil
}

var changedData ChangedData = ChangedData{mu: &sync.RWMutex{}, Charts: make(map[string]*Chart)}
var stagingData ChangedData = ChangedData{mu: &sync.RWMutex{}, Charts: make(map[string]*Chart)}
var activeCharts ActiveCharts = ActiveCharts{items: make(map[string]ActiveChart), mu: &sync.RWMutex{}}
var currentStatus status = status{processedCount: 0, totalCount: 0, skippedCount: 0, avgTime: 0, totalProcessingTime: 0, mu: &sync.RWMutex{}}
var skipCommitsWithBadMessage bool
var dateFormat = "2006-01-02"
var walkChartsFunc = helper.WalkCharts2
var openRepoFunc = git.PlainOpen
var processCommitFunc = processCommit
var mergeStagingToCurrentFunc = mergeStagingToCurrent
var writeChangedDataFunc = func(path string) error { return changedData.WriteToFile(path) }
var prepareGenerateFunc = func(o *ChangelogOptions, start time.Time) error { return o.prepareGenerate(start) }
var loadCommitsForGenerateFunc = func(o *ChangelogOptions) ([]*gitobject.Commit, error) { return o.loadCommitsForGenerate() }
var repoLogFunc = func(repo *git.Repository) (gitobject.CommitIter, error) {
	return repo.Log(&git.LogOptions{Order: git.LogOrderCommitterTime})
}

func (o *ChangelogOptions) Generate() error {
	start := time.Now()
	skipCommitsWithBadMessage = o.SkipCommitsWithBadMessage
	log.Info().Msgf("Starting changelog generation at %s", start)
	if err := prepareGenerateFunc(o, start); err != nil {
		return err
	}

	commits, err := loadCommitsForGenerateFunc(o)
	if err != nil {
		return err
	}
	if len(commits) == 0 {
		log.Info().Msgf("No commits to process in %s", o.RepoPath)
		return nil
	}
	log.Info().Msgf("Found [%d] commits to process in %s", len(commits), o.RepoPath)

	stop := make(chan struct{}) // Stop channel
	defer close(stop)
	go o.statusPrinter(stop)
	o.processCommits(commits)

	stop <- struct{}{}

	if err := mergeStagingToCurrentFunc(); err != nil {
		return err
	}
	log.Info().Msgf("Writhing json to %s", o.JSONOutputPath)
	if err := writeChangedDataFunc(o.JSONOutputPath); err != nil {
		return fmt.Errorf("error writing json new file: %s", err)
	}
	log.Info().Msgf("Finished in %s", time.Since(start))
	o.printStatus(start, false)
	return nil
}

func (o *ChangelogOptions) prepareGenerate(start time.Time) error {
	if err := o.validate(); err != nil {
		return err
	}

	if err := walkChartsFunc([]string{o.RepoPath}, activeCharts.getActiveChartsWalker, helper.AsyncMode); err != nil {
		return err
	}
	log.Info().Msgf("Found [%d] active charts in [%s]", len(activeCharts.items), time.Since(start))

	log.Info().Msgf("Loading json %s", o.JSONOutputPath)
	if err := changedData.LoadFromFile(o.JSONOutputPath); err != nil {
		return fmt.Errorf("failed to load existing json file, maybe it is not matching the current structure: %w", err)
	}
	if changedData.LastCommit == "" {
		log.Info().Msgf("No last commit found in [%s], starting from the beginning", o.JSONOutputPath)
	} else {
		log.Info().Msgf("Last commit found in [%s], will start from [%s]", o.JSONOutputPath, changedData.LastCommit)
	}

	return nil
}

func (o *ChangelogOptions) loadCommitsForGenerate() ([]*gitobject.Commit, error) {
	repo, err := openRepoFunc(o.RepoPath)
	if err != nil {
		return nil, err
	}

	cIter, err := repoLogFunc(repo)
	if err != nil {
		return nil, err
	}

	return o.reverseCommits(cIter, changedData.LastCommit)
}

func (o *ChangelogOptions) processCommits(commits []*gitobject.Commit) {
	for _, c := range commits {
		changedData.mu.Lock()
		changedData.LastCommit = c.Hash.String()
		changedData.mu.Unlock()

		commitStart := time.Now()
		if err := processCommitFunc(c); err != nil {
			log.Error().Err(err).Msgf("Error processing commit: %s", c.Hash.String())
		}

		currentStatus.mu.Lock()
		currentStatus.processedCount++
		currentStatus.totalProcessingTime += time.Since(commitStart)
		currentStatus.avgTime = currentStatus.totalProcessingTime / time.Duration(currentStatus.processedCount+currentStatus.skippedCount)
		currentStatus.mu.Unlock()
	}
}

// We have to go over the stagingData, for each chart,
// we sort the versions from the changelogData
// and we add the commits from stagingData to the nearest next version in changelogData
func mergeStagingToCurrent() error {
	start := time.Now()
	log.Info().Msgf("Merging staging to current")
	changedData.mu.Lock()
	defer changedData.mu.Unlock()

	stagingData.mu.Lock()
	defer stagingData.mu.Unlock()
	for chart, stagingChartItem := range stagingData.Charts {
		if err := mergeChartStaging(chart, stagingChartItem); err != nil {
			return err
		}
	}

	log.Info().Msgf("Finished merging in %s", time.Since(start))
	return nil
}

func mergeChartStaging(chart string, stagingChartItem *Chart) error {
	chartItem, ok := changedData.Charts[chart]
	if !ok {
		changedData.Charts[chart] = stagingChartItem
		return nil
	}

	if chartItem.Versions == nil || len(chartItem.Versions) == 0 {
		chartItem.Versions = stagingChartItem.Versions
		return nil
	}

	chartVersions, err := chartItem.SortVersions(false)
	if err != nil {
		return err
	}

	for versionKey := range stagingData.Charts[chart].Versions {
		if err = mergeVersionStaging(chart, versionKey, chartItem, stagingChartItem, chartVersions); err != nil {
			return err
		}
	}

	return nil
}

func mergeVersionStaging(chart string, versionKey string, chartItem *Chart, stagingChartItem *Chart, chartVersions []*semver.Version) error {
	stagingVer, err := semver.NewVersion(versionKey)
	if err != nil {
		return err
	}

	for _, chartVer := range chartVersions {
		if !chartVer.GreaterThan(stagingVer) {
			continue
		}
		chartVerItem := ensureChartVersion(chartItem, versionKey, stagingChartItem)
		mergeVersionCommits(chart, versionKey, chartVerItem, stagingChartItem.Versions[versionKey].Commits)
		return nil
	}

	mergeVersionCommits(chart, versionKey, chartItem.Versions[versionKey], stagingChartItem.Versions[versionKey].Commits)
	return nil
}

func ensureChartVersion(chartItem *Chart, versionKey string, stagingChartItem *Chart) *Version {
	chartVerItem, ok := chartItem.Versions[versionKey]
	if !ok {
		chartItem.AddVersion(versionKey, stagingChartItem.Versions[versionKey].Train)
		chartVerItem = chartItem.Versions[versionKey]
	}
	return chartVerItem
}

func mergeVersionCommits(chart string, versionKey string, chartVerItem *Version, commits map[string]*Commit) {
	if chartVerItem.Commits == nil {
		log.Warn().Msgf("Commits were nil for version [%s] in chart [%s]", versionKey, chart)
		chartVerItem.Commits = make(map[string]*Commit)
	}

	for commitKey, commit := range commits {
		if _, ok := chartVerItem.Commits[commitKey]; ok {
			log.Warn().Msgf("Commit [%s] already exists in version [%s]", commitKey, versionKey)
			continue
		}
		chartVerItem.Commits[commitKey] = commit
	}
}

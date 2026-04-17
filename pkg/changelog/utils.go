package changelog

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rs/zerolog/log"
)

type status struct {
	processedCount      int
	totalCount          int
	skippedCount        int
	avgTime             time.Duration
	totalProcessingTime time.Duration
	mu                  *sync.RWMutex
}

func (s *status) incSkippedCount() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.skippedCount++
	s.processedCount--
}

func (o *ChangelogOptions) printStatus(start time.Time, eta bool) {
	currentStatus.mu.RLock()
	defer currentStatus.mu.RUnlock()
	if eta {
		log.Info().Msgf("Processed [%d + (%d skipped) / %d] commits in %s, ETA: %s (avg commit processing time: %s)", currentStatus.processedCount, currentStatus.skippedCount, currentStatus.totalCount, time.Since(start), time.Duration(currentStatus.totalCount-currentStatus.processedCount-currentStatus.skippedCount)*currentStatus.avgTime, currentStatus.avgTime)
	} else {
		log.Info().Msgf("Processed [%d + (%d skipped) / %d] commits in %s", currentStatus.processedCount, currentStatus.skippedCount, currentStatus.totalCount, time.Since(start))
	}
}

func (o *ChangelogOptions) statusPrinter(stop <-chan struct{}) {
	log.Info().Msgf("Printing status every [%d] seconds", o.StatusUpdateInterval)
	start := time.Now()
	ticker := time.NewTicker(time.Second * time.Duration(o.StatusUpdateInterval))
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			o.printStatus(start, true)
		case <-stop:
			return
		}
	}
}

func (o *ChangelogOptions) reverseCommits(cIter object.CommitIter, lastCommit string) ([]*object.Commit, error) {
	start := time.Now()
	var commits []*object.Commit
	var errDoneReversing = errors.New("done reversing commits")
	log.Info().Msgf("Reversing commits order")
	defer cIter.Close()
	if err := cIter.ForEach(func(c *object.Commit) error {
		// We go from newer to oldest, if we hit the last commit, we stop
		if c.Hash.String() == lastCommit {
			return errDoneReversing
		}

		currentStatus.totalCount++
		// Reverse the order of the commits to get the oldest first
		commits = append([]*object.Commit{c}, commits...)
		return nil
	}); err != nil {
		if !errors.Is(err, errDoneReversing) {
			return nil, err
		}
	}

	log.Info().Msgf("Finished reversing commits in %s", time.Since(start))
	return commits, nil
}

// Just some random text to avoid any app name conflicts
var invalidName = "5fdad45c8f5b954e5643c314"

func getAppName(path string) string {
	// path = apps/<train>/<app>/...
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		log.Debug().Msgf("failed to get app name from path [%s]", path)
		return invalidName
	}
	return parts[2]
}

var appFilePathRegex = regexp.MustCompile(`^charts/([\w-_]+)/([\w-_]+)/Chart.yaml$`)
var commitTreeFunc = func(c *object.Commit) (*object.Tree, error) { return c.Tree() }
var treeFileFunc = func(t *object.Tree, path string) (*object.File, error) { return t.File(path) }
var fileContentsFunc = func(f *object.File) (string, error) { return f.Contents() }

func getAppPath(path string) (string, error) {
	original := path
	for {
		if path == "." {
			return "", fmt.Errorf("path too short [%s], or could not construct app path", original)
		}
		if appFilePathRegex.MatchString(filepath.Join(path, "Chart.yaml")) {
			return filepath.Join(path, "Chart.yaml"), nil
		}
		// Remove the last part of the path and try again
		path = filepath.Dir(path)
	}
}

func getAppVersion(c *object.Commit, path string) (version string, retErr error) {
	defer func() {
		if r := recover(); r != nil {
			retErr = fmt.Errorf("failed to get tree: %v", r)
		}
	}()

	tree, err := commitTreeFunc(c)
	if err != nil {
		return "", fmt.Errorf("failed to get tree: %w", err)
	}
	file, err := treeFileFunc(tree, path)
	if err != nil {
		return "", fmt.Errorf("failed to get file: %w", err)
	}
	strData, err := fileContentsFunc(file)
	if err != nil {
		return "", fmt.Errorf("failed to get file contents: %w", err)
	}
	return getVersion(strData)
}

var charsToRemove = []string{"-"}

// We use this instead of NewHelmChart.Load(), because
// this will work even if the Chart.yaml is malformed
func getVersion(strData string) (string, error) {
	lines := strings.Split(strData, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "version:") {
			ver := strings.TrimSpace(strings.Split(line, ":")[1])
			// In some cases there was a type in the version (eg "1.0.-2")
			for _, c := range charsToRemove {
				ver = strings.ReplaceAll(ver, c, "")
			}
			return ver, nil
		}
	}
	return "", fmt.Errorf("could not find version in file")
}

func getAppTrain(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		log.Error().Msgf("Could not get app train from path [%s]", path)
		return ""
	}
	return parts[1]
}

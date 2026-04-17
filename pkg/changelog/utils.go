package changelog

import (
	"errors"
	"fmt"
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

var commitTreeFunc = func(c *object.Commit) (*object.Tree, error) { return c.Tree() }
var treeFileFunc = func(t *object.Tree, path string) (*object.File, error) { return t.File(path) }
var fileContentsFunc = func(f *object.File) (string, error) { return f.Contents() }

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
	return getVersionFromContentFunc(strData)
}

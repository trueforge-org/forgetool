package changelog

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rs/zerolog/log"
)

// getCommitMessage returns the first line of the commit message
func getCommitMessage(c *object.Commit) string {
	return strings.TrimSpace(strings.Split(c.Message, "\n")[0])
}

// TODO: update regex
var commitMessageRegex = regexp.MustCompile(`^(chore|feat|fix|docs)\((.+)\)?: (.+)`)

func getCommitKind(c *object.Commit) string {
	match := commitMessageRegex.FindStringSubmatch(getCommitMessage(c))
	if match == nil {
		return ""
	}
	return match[1]
}

func isValidCommit(c *object.Commit) bool {
	if c.Message == "" {
		currentStatus.incSkippedCount()
		log.Debug().Msgf("Skipping commit [%s]. Reason: the commit message is empty", c.Hash.String())
		return false
	}
	if c.ParentHashes == nil || len(c.ParentHashes) == 0 {
		currentStatus.incSkippedCount()
		log.Debug().Msgf("Skipping commit [%s]. Reason: the commit is Batman (has no parent)", c.Hash.String())
		return false
	}

	if skipCommitsWithBadMessage && !commitMessageRegex.MatchString(getCommitMessage(c)) {
		currentStatus.incSkippedCount()
		log.Debug().Msgf("Skipping commit [%s]. Reason: the commit message does not match the pattern", c.Hash.String())
		return false
	}

	return true
}

type oldNewPaths struct {
	old diff.File
	new diff.File
}
type chartsWithChangedFiles map[string][]oldNewPaths
type chartsWithChangedFile map[string]oldNewPaths

var getParentCommitFunc = func(c *object.Commit) (*object.Commit, error) { return c.Parent(0) }
var getPatchFunc = func(par, c *object.Commit) (*object.Patch, error) { return par.Patch(c) }
var getChartsWithMultipleChangedFilesFunc = getChartsWithMultipleChangedFiles
var processChartsWithSingleChangedFileFunc = processChartsWithSingleChangedFile

func processCommit(c *object.Commit) error {
	var err error
	if !isValidCommit(c) {
		return nil
	}

	parCommit, err := getParentCommitFunc(c)
	if err != nil {
		return fmt.Errorf("failed to get parent commit: %w", err)
	}
	patch, err := getPatchFunc(parCommit, c)
	if err != nil {
		return fmt.Errorf("failed to get patch: %w", err)
	}

	// Go over the filePatches (old/new pairs) and get create a
	// map of charts with an slice of all the old/new fileDiffs
	chartsWithMultipleFiles, err := getChartsWithMultipleChangedFilesFunc(patch)
	if err != nil {
		return fmt.Errorf("failed to get changed files: %w", err)
	}

	// For each chart, keep a single old/new pair, preferably the chart.yaml
	// otherwise the first file in the list, doesn't matter
	chartsWithSingleFile := getChartsWithSingleChangedFile(chartsWithMultipleFiles)

	// Populate the changedData and stagingData
	if err := processChartsWithSingleChangedFileFunc(c, parCommit, chartsWithSingleFile); err != nil {
		return fmt.Errorf("failed to process changed file: %w", err)
	}

	return nil
}

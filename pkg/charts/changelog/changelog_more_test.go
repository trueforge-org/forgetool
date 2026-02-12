package changelog

import (
	"io"
	"sync"
	"testing"

	"github.com/go-git/go-git/v5/plumbing/object"
)

// sliceCommitIter is a simple CommitIter backed by a slice.
type sliceCommitIter struct {
	commits []*object.Commit
	pos     int
}

func (it *sliceCommitIter) Next() (*object.Commit, error) {
	if it.pos >= len(it.commits) {
		return nil, io.EOF
	}
	c := it.commits[it.pos]
	it.pos++
	return c, nil
}

func (it *sliceCommitIter) ForEach(cb func(*object.Commit) error) error {
	for {
		c, err := it.Next()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if err := cb(c); err != nil {
			return err
		}
	}
}

func (it *sliceCommitIter) Close() {}

func TestReverseCommitsEmpty(t *testing.T) {
	resetChangelogGlobals()
	opts := &ChangelogOptions{StatusUpdateInterval: 1}
	iter := &sliceCommitIter{}
	commits, err := opts.reverseCommits(iter, "")
	if err != nil {
		t.Fatalf("reverseCommits returned error: %v", err)
	}
	if len(commits) != 0 {
		t.Fatalf("expected 0 commits, got %d", len(commits))
	}
}

func TestReverseCommitsSingle(t *testing.T) {
	resetChangelogGlobals()
	opts := &ChangelogOptions{StatusUpdateInterval: 1}
	c := &object.Commit{Message: "feat: single"}
	iter := &sliceCommitIter{commits: []*object.Commit{c}}
	commits, err := opts.reverseCommits(iter, "")
	if err != nil {
		t.Fatalf("reverseCommits returned error: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(commits))
	}
	if commits[0] != c {
		t.Fatal("expected the same commit object")
	}
}

func TestReverseCommitsMultiple(t *testing.T) {
	resetChangelogGlobals()
	opts := &ChangelogOptions{StatusUpdateInterval: 1}
	c1 := &object.Commit{Message: "feat: first"}
	c2 := &object.Commit{Message: "fix: second"}
	c3 := &object.Commit{Message: "chore: third"}
	// Iterator yields newest first (c3, c2, c1); reverseCommits should return oldest first (c1, c2, c3).
	iter := &sliceCommitIter{commits: []*object.Commit{c3, c2, c1}}
	commits, err := opts.reverseCommits(iter, "")
	if err != nil {
		t.Fatalf("reverseCommits returned error: %v", err)
	}
	if len(commits) != 3 {
		t.Fatalf("expected 3 commits, got %d", len(commits))
	}
	if commits[0] != c1 || commits[1] != c2 || commits[2] != c3 {
		t.Fatal("commits are not in reversed order")
	}
}

func TestStatusPrinterDoesNotPanic(t *testing.T) {
	currentStatus = status{
		processedCount:      5,
		totalCount:          10,
		skippedCount:        1,
		avgTime:             0,
		totalProcessingTime: 0,
		mu:                  &sync.RWMutex{},
	}
	opts := &ChangelogOptions{StatusUpdateInterval: 1}
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		opts.statusPrinter(stop)
	}()
	// Signal stop immediately; the test just verifies no panic occurs.
	stop <- struct{}{}
	<-done
}

func TestValidateMissingRepoPath(t *testing.T) {
	opts := ChangelogOptions{
		TemplatePath:         "/tmp/tmpl",
		ChangelogFileName:    "CHANGELOG.md",
		ChartsDir:            "/tmp/charts",
		JSONOutputPath:       "/tmp/out.json",
		StatusUpdateInterval: 1,
	}
	if err := opts.validate(); err == nil {
		t.Fatal("expected error for missing RepoPath")
	}
}

func TestValidateMissingTemplatePath(t *testing.T) {
	opts := ChangelogOptions{
		RepoPath:             "/tmp/repo",
		ChangelogFileName:    "CHANGELOG.md",
		ChartsDir:            "/tmp/charts",
		JSONOutputPath:       "/tmp/out.json",
		StatusUpdateInterval: 1,
	}
	if err := opts.validate(); err == nil {
		t.Fatal("expected error for missing TemplatePath")
	}
}

func TestValidateMissingChangelogFileName(t *testing.T) {
	opts := ChangelogOptions{
		RepoPath:             "/tmp/repo",
		TemplatePath:         "/tmp/tmpl",
		ChartsDir:            "/tmp/charts",
		JSONOutputPath:       "/tmp/out.json",
		StatusUpdateInterval: 1,
	}
	if err := opts.validate(); err == nil {
		t.Fatal("expected error for missing ChangelogFileName")
	}
}

func TestValidateMissingChartsDir(t *testing.T) {
	opts := ChangelogOptions{
		RepoPath:             "/tmp/repo",
		TemplatePath:         "/tmp/tmpl",
		ChangelogFileName:    "CHANGELOG.md",
		JSONOutputPath:       "/tmp/out.json",
		StatusUpdateInterval: 1,
	}
	if err := opts.validate(); err == nil {
		t.Fatal("expected error for missing ChartsDir")
	}
}

func TestValidateMissingJSONOutputPath(t *testing.T) {
	opts := ChangelogOptions{
		RepoPath:             "/tmp/repo",
		TemplatePath:         "/tmp/tmpl",
		ChangelogFileName:    "CHANGELOG.md",
		ChartsDir:            "/tmp/charts",
		StatusUpdateInterval: 1,
	}
	if err := opts.validate(); err == nil {
		t.Fatal("expected error for missing JSONOutputPath")
	}
}

func TestValidateZeroStatusUpdateInterval(t *testing.T) {
	opts := ChangelogOptions{
		RepoPath:          "/tmp/repo",
		TemplatePath:      "/tmp/tmpl",
		ChangelogFileName: "CHANGELOG.md",
		ChartsDir:         "/tmp/charts",
		JSONOutputPath:    "/tmp/out.json",
	}
	if err := opts.validate(); err == nil {
		t.Fatal("expected error for zero StatusUpdateInterval")
	}
}

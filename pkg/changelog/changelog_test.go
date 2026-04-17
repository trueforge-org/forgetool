package changelog

import (
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestCommitMessageAndKind(t *testing.T) {
	c := &object.Commit{Message: "feat(scope): add feature\nmore details"}
	msg := getCommitMessage(c)
	if msg != "feat(scope): add feature" {
		t.Fatalf("unexpected commit message: %s", msg)
	}

	kind := getCommitKind(c)
	if kind != "feat" {
		t.Fatalf("unexpected commit kind: %s", kind)
	}

	c2 := &object.Commit{Message: "no conventional message"}
	if getCommitKind(c2) != "" {
		t.Fatalf("expected empty kind for non-matching message")
	}
}

func TestIsValidCommit(t *testing.T) {
	// reset status and globals to stable defaults
	currentStatus = status{processedCount: 0, totalCount: 0, skippedCount: 0, avgTime: 0, totalProcessingTime: 0, mu: &sync.RWMutex{}}

	// empty message
	cEmpty := &object.Commit{Message: ""}
	skipCommitsWithBadMessage = false
	if isValidCommit(cEmpty) {
		t.Fatalf("expected empty message commit to be invalid")
	}

	// no parent
	cNoParent := &object.Commit{Message: "feat(x): y", ParentHashes: nil}
	if isValidCommit(cNoParent) {
		t.Fatalf("expected commit with no parents to be invalid")
	}

	// skip commits with bad message enabled
	cBad := &object.Commit{Message: "random message", ParentHashes: []plumbing.Hash{{}}}
	skipCommitsWithBadMessage = true
	if isValidCommit(cBad) {
		t.Fatalf("expected bad message commit to be invalid when skip enabled")
	}

	// valid commit
	cValid := &object.Commit{Message: "fix(scope): bugfix", ParentHashes: []plumbing.Hash{{}}}
	skipCommitsWithBadMessage = false
	if !isValidCommit(cValid) {
		t.Fatalf("expected valid commit to be valid")
	}
}

func TestAppSortVersions(t *testing.T) {
	c := &App{Versions: map[string]*Version{
		"2.0.0": {Version: "2.0.0"},
		"1.0.0": {Version: "1.0.0"},
	}}

	vers, err := c.SortVersions(false)
	if err != nil {
		t.Fatalf("SortVersions failed: %v", err)
	}
	if len(vers) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(vers))
	}
	if vers[0].String() != "1.0.0" || vers[1].String() != "2.0.0" {
		t.Fatalf("unexpected order: %+v", vers)
	}
}

func TestAddOrUpdateAppAndCommits(t *testing.T) {
	var cd ChangedData

	// construct fake commits
	parentHash := plumbing.NewHash("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	commit := &object.Commit{
		Hash:         plumbing.NewHash("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
		ParentHashes: []plumbing.Hash{parentHash},
		Author: object.Signature{
			Name: "Tester",
			When: time.Date(2021, 1, 2, 3, 4, 5, 0, time.UTC),
		},
		Message: "feat(core): add feature\n\ndetails",
	}

	cd.AddOrUpdateApp("myapp", "1.2.3", "trainA", commit)

	if cd.Apps == nil {
		t.Fatalf("Apps map not initialized")
	}
	ch, ok := cd.Apps["myapp"]
	if !ok {
		t.Fatalf("app missing")
	}
	if _, ok := ch.Versions["1.2.3"]; !ok {
		t.Fatalf("version missing")
	}
	v := ch.Versions["1.2.3"]
	if len(v.Commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(v.Commits))
	}

	// test sorting commits
	_, err := v.SortCommits(false)
	if err != nil {
		t.Fatalf("SortCommits failed: %v", err)
	}
	if len(v.SortedCommits) != 1 {
		t.Fatalf("expected 1 sorted commit, got %d", len(v.SortedCommits))
	}
}

func TestLoadAndWriteToFile(t *testing.T) {
	td := t.TempDir()
	p := filepath.Join(td, "changed.json")

	cd := &ChangedData{}
	// populate a small structure
	cd.Apps = map[string]*App{"c": {Versions: map[string]*Version{"0.1.0": {Version: "0.1.0"}}}}

	if err := cd.WriteToFile(p); err != nil {
		t.Fatalf("WriteToFile failed: %v", err)
	}

	var cd2 ChangedData
	if err := cd2.LoadFromFile(p); err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}
	if cd2.Apps == nil {
		t.Fatalf("expected apps loaded")
	}
	if _, ok := cd2.Apps["c"]; !ok {
		t.Fatalf("expected app 'c' in loaded data")
	}

	// invalid path tests
	if err := cd2.LoadFromFile(filepath.Join(td, "nonexistent", "file")); err != nil {
		// should return nil when file doesn't exist
		t.Fatalf("LoadFromFile on missing file returned error: %v", err)
	}
	// directory path returns error
	if err := cd2.LoadFromFile(td); err == nil {
		t.Fatalf("LoadFromFile should error when given directory")
	}
	// cleanup
	_ = os.Remove(p)
}

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
	stop <- struct{}{}
	<-done
}

func TestValidateMissingRepoPath(t *testing.T) {
	opts := ChangelogOptions{
		TemplatePath:         "/tmp/tmpl",
		ChangelogFileName:    "CHANGELOG.md",
		AppsDir:              "/tmp/charts",
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
		AppsDir:              "/tmp/charts",
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
		AppsDir:              "/tmp/charts",
		JSONOutputPath:       "/tmp/out.json",
		StatusUpdateInterval: 1,
	}
	if err := opts.validate(); err == nil {
		t.Fatal("expected error for missing ChangelogFileName")
	}
}

func TestValidateMissingAppsDir(t *testing.T) {
	opts := ChangelogOptions{
		RepoPath:             "/tmp/repo",
		TemplatePath:         "/tmp/tmpl",
		ChangelogFileName:    "CHANGELOG.md",
		JSONOutputPath:       "/tmp/out.json",
		StatusUpdateInterval: 1,
	}
	if err := opts.validate(); err == nil {
		t.Fatal("expected error for missing AppsDir")
	}
}

func TestValidateMissingJSONOutputPath(t *testing.T) {
	opts := ChangelogOptions{
		RepoPath:             "/tmp/repo",
		TemplatePath:         "/tmp/tmpl",
		ChangelogFileName:    "CHANGELOG.md",
		AppsDir:              "/tmp/charts",
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
		AppsDir:           "/tmp/charts",
		JSONOutputPath:    "/tmp/out.json",
	}
	if err := opts.validate(); err == nil {
		t.Fatal("expected error for zero StatusUpdateInterval")
	}
}

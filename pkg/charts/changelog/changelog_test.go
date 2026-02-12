package changelog

import (
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

func TestChartSortVersions(t *testing.T) {
	c := &Chart{Versions: map[string]*Version{
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

func TestAddOrUpdateChartAndCommits(t *testing.T) {
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

	cd.AddOrUpdateChart("mychart", "1.2.3", "trainA", commit)

	if cd.Charts == nil {
		t.Fatalf("Charts map not initialized")
	}
	ch, ok := cd.Charts["mychart"]
	if !ok {
		t.Fatalf("chart missing")
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
	cd.Charts = map[string]*Chart{"c": {Versions: map[string]*Version{"0.1.0": {Version: "0.1.0"}}}}

	if err := cd.WriteToFile(p); err != nil {
		t.Fatalf("WriteToFile failed: %v", err)
	}

	var cd2 ChangedData
	if err := cd2.LoadFromFile(p); err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}
	if cd2.Charts == nil {
		t.Fatalf("expected charts loaded")
	}
	if _, ok := cd2.Charts["c"]; !ok {
		t.Fatalf("expected chart 'c' in loaded data")
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

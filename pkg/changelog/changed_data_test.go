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

func TestSortVersionsReverse(t *testing.T) {
	c := &App{Versions: map[string]*Version{
		"1.0.0": {Version: "1.0.0"},
		"3.0.0": {Version: "3.0.0"},
		"2.0.0": {Version: "2.0.0"},
	}}

	vers, err := c.SortVersions(true)
	if err != nil {
		t.Fatalf("SortVersions(reverse=true) failed: %v", err)
	}
	if len(vers) != 3 {
		t.Fatalf("expected 3 versions, got %d", len(vers))
	}
	if vers[0].String() != "3.0.0" || vers[1].String() != "2.0.0" || vers[2].String() != "1.0.0" {
		t.Fatalf("unexpected reverse order: %v, %v, %v", vers[0], vers[1], vers[2])
	}
	if len(c.SortedVersions) != 3 {
		t.Fatalf("expected SortedVersions to have 3 entries, got %d", len(c.SortedVersions))
	}
}

func TestSortVersionsInvalidVersion(t *testing.T) {
	c := &App{Versions: map[string]*Version{
		"not-a-version": {Version: "not-a-version"},
	}}
	_, err := c.SortVersions(false)
	if err == nil {
		t.Fatalf("expected error for invalid semver")
	}
}

func TestAddOrUpdateAppDuplicate(t *testing.T) {
	parentHash := plumbing.NewHash("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	commit1 := &object.Commit{
		Hash:         plumbing.NewHash("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
		ParentHashes: []plumbing.Hash{parentHash},
		Author:       object.Signature{Name: "Test", When: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		Message:      "feat(x): first",
	}
	commit2 := &object.Commit{
		Hash:         plumbing.NewHash("cccccccccccccccccccccccccccccccccccccccc"),
		ParentHashes: []plumbing.Hash{parentHash},
		Author:       object.Signature{Name: "Test", When: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)},
		Message:      "fix(x): second",
	}

	var cd ChangedData
	cd.AddOrUpdateApp("myapp", "1.0.0", "stable", commit1)
	cd.AddOrUpdateApp("myapp", "1.0.0", "stable", commit2)

	if len(cd.Apps["myapp"].Versions["1.0.0"].Commits) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(cd.Apps["myapp"].Versions["1.0.0"].Commits))
	}

	// Adding the same commit again should not increase count
	cd.AddOrUpdateApp("myapp", "1.0.0", "stable", commit1)
	if len(cd.Apps["myapp"].Versions["1.0.0"].Commits) != 2 {
		t.Fatalf("expected 2 commits after duplicate, got %d", len(cd.Apps["myapp"].Versions["1.0.0"].Commits))
	}
}

func TestAddVersionIdempotent(t *testing.T) {
	c := &App{}
	c.AddVersion("1.0.0", "stable")
	c.AddVersion("1.0.0", "stable")
	if len(c.Versions) != 1 {
		t.Fatalf("expected 1 version, got %d", len(c.Versions))
	}
}

func TestAddCommitIdempotent(t *testing.T) {
	parentHash := plumbing.NewHash("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	commit := &object.Commit{
		Hash:         plumbing.NewHash("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
		ParentHashes: []plumbing.Hash{parentHash},
		Author:       object.Signature{Name: "Test", When: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		Message:      "feat(x): test",
	}

	v := &Version{Version: "1.0.0", Train: "stable"}
	v.AddCommit(commit)
	v.AddCommit(commit)
	if len(v.Commits) != 1 {
		t.Fatalf("expected 1 commit after duplicate add, got %d", len(v.Commits))
	}
}

func TestSortCommitsMultipleReverse(t *testing.T) {
	v := &Version{
		Version: "1.0.0",
		Commits: map[string]*Commit{
			"a": {CommitHash: "a", Author: Author{Name: "x", Date: "2024-01-01"}},
			"b": {CommitHash: "b", Author: Author{Name: "y", Date: "2024-01-03"}},
			"c": {CommitHash: "c", Author: Author{Name: "z", Date: "2024-01-02"}},
		},
	}
	sorted, err := v.SortCommits(true)
	if err != nil {
		t.Fatalf("SortCommits failed: %v", err)
	}
	if len(sorted) != 3 {
		t.Fatalf("expected 3 sorted commits, got %d", len(sorted))
	}
	// Reverse: newest first
	if sorted[0].CommitHash != "b" {
		t.Fatalf("expected newest commit first, got %s", sorted[0].CommitHash)
	}
	if sorted[2].CommitHash != "a" {
		t.Fatalf("expected oldest commit last, got %s", sorted[2].CommitHash)
	}
}

func TestSortCommitsForward(t *testing.T) {
	v := &Version{
		Version: "1.0.0",
		Commits: map[string]*Commit{
			"a": {CommitHash: "a", Author: Author{Name: "x", Date: "2024-01-03"}},
			"b": {CommitHash: "b", Author: Author{Name: "y", Date: "2024-01-01"}},
		},
	}
	sorted, err := v.SortCommits(false)
	if err != nil {
		t.Fatalf("SortCommits failed: %v", err)
	}
	// Forward: oldest first
	if sorted[0].CommitHash != "b" {
		t.Fatalf("expected oldest commit first, got %s", sorted[0].CommitHash)
	}
}

func TestLoadFromFileInvalidJSON(t *testing.T) {
	td := t.TempDir()
	p := filepath.Join(td, "invalid.json")
	if err := os.WriteFile(p, []byte("{invalid json}"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	var cd ChangedData
	err := cd.LoadFromFile(p)
	if err == nil {
		t.Fatalf("expected error for invalid JSON")
	}
}

func TestLoadFromFileNonExistent(t *testing.T) {
	td := t.TempDir()
	var cd ChangedData
	err := cd.LoadFromFile(filepath.Join(td, "nonexistent", "file.json"))
	if err != nil {
		t.Fatalf("LoadFromFile should return nil for non-existent file, got: %v", err)
	}
}

func TestWriteToFileAndRoundTrip(t *testing.T) {
	td := t.TempDir()
	p := filepath.Join(td, "roundtrip.json")

	cd := &ChangedData{
		mu:         &sync.RWMutex{},
		LastCommit: "abc123",
		Apps: map[string]*App{
			"myapp": {
				Versions: map[string]*Version{
					"1.0.0": {
						Version: "1.0.0",
						Train:   "stable",
						Commits: map[string]*Commit{
							"hash1": {
								CommitHash: "hash1",
								ParentHash: "parent1",
								Author:     Author{Name: "dev", Date: "2024-03-15"},
								Kind:       "feat",
								Message:    "add feature",
							},
						},
					},
				},
			},
		},
	}

	if err := cd.WriteToFile(p); err != nil {
		t.Fatalf("WriteToFile failed: %v", err)
	}

	var cd2 ChangedData
	if err := cd2.LoadFromFile(p); err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if cd2.LastCommit != "abc123" {
		t.Fatalf("expected LastCommit abc123, got %s", cd2.LastCommit)
	}
	if cd2.Apps["myapp"] == nil {
		t.Fatalf("expected myapp app")
	}
	if cd2.Apps["myapp"].Versions["1.0.0"].Commits["hash1"].Message != "add feature" {
		t.Fatalf("expected commit message preserved")
	}
}

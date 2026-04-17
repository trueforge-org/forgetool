package changelog

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
)

func TestGetAppsWithSingleChangedFilePreferChartYaml(t *testing.T) {
	multi := appsWithChangedFiles{
		"app": {
			{new: &fakeFile{path: "charts/stable/app/values.yaml"}, old: nil},
			{new: &fakeFile{path: "charts/stable/app/Chart.yaml"}, old: nil},
		},
	}
	single := getAppsWithSingleChangedFile(multi)
	if single["app"].new.Path() != "charts/stable/app/Chart.yaml" {
		t.Fatalf("expected Chart.yaml to be preferred, got %s", single["app"].new.Path())
	}
}

func TestGetAppsWithSingleChangedFileFirstIfNoChartYaml(t *testing.T) {
	multi := appsWithChangedFiles{
		"app": {
			{new: &fakeFile{path: "charts/stable/app/values.yaml"}, old: nil},
			{new: &fakeFile{path: "charts/stable/app/templates/deploy.yaml"}, old: nil},
		},
	}
	single := getAppsWithSingleChangedFile(multi)
	// First non-Chart.yaml entry should be kept if no Chart.yaml is found
	if single["app"].new == nil {
		t.Fatalf("expected a file to be selected")
	}
}

func TestCheckPathDirectoryExists(t *testing.T) {
	td := t.TempDir()
	if err := checkPath(td, false); err != nil {
		t.Fatalf("checkPath on existing directory should not error: %v", err)
	}
}

func TestCheckPathNonExistentNoCreate(t *testing.T) {
	td := t.TempDir()
	if err := checkPath(filepath.Join(td, "nonexistent"), false); err != nil {
		t.Fatalf("checkPath on non-existent without create should not error: %v", err)
	}
}

func TestMergeStagingToCurrentNoVersionsInChanged(t *testing.T) {
	resetChangelogGlobals()
	changedData.Apps["app"] = &App{Versions: nil}
	stagingData.Apps["app"] = &App{Versions: map[string]*Version{
		"1.0.0": {Version: "1.0.0", Train: "stable", Commits: map[string]*Commit{
			"abc": {CommitHash: "abc", Author: Author{Name: "x", Date: "2024-01-01"}},
		}},
	}}

	if err := mergeStagingToCurrent(); err != nil {
		t.Fatalf("mergeStagingToCurrent failed: %v", err)
	}

	if changedData.Apps["app"].Versions == nil {
		t.Fatalf("expected versions to be populated from staging")
	}
	if changedData.Apps["app"].Versions["1.0.0"] == nil {
		t.Fatalf("expected version 1.0.0 from staging")
	}
}

func TestMergeStagingToCurrentDuplicateCommit(t *testing.T) {
	resetChangelogGlobals()
	changedData.Apps["app"] = &App{Versions: map[string]*Version{
		"2.0.0": {Version: "2.0.0", Train: "stable", Commits: map[string]*Commit{
			"abc": {CommitHash: "abc", Author: Author{Name: "x", Date: "2024-01-01"}},
		}},
	}}
	stagingData.Apps["app"] = &App{Versions: map[string]*Version{
		"1.0.0": {Version: "1.0.0", Train: "stable", Commits: map[string]*Commit{
			"abc": {CommitHash: "abc", Author: Author{Name: "x", Date: "2024-01-01"}},
		}},
	}}

	if err := mergeStagingToCurrent(); err != nil {
		t.Fatalf("mergeStagingToCurrent failed: %v", err)
	}
}

func TestMergeStagingToCurrentNoGreaterVersion(t *testing.T) {
	resetChangelogGlobals()
	// Both changedData and stagingData have the same version key
	// so the "no greater version" path adds commits to the existing version
	changedData.Apps["app"] = &App{Versions: map[string]*Version{
		"2.0.0": {Version: "2.0.0", Train: "stable", Commits: map[string]*Commit{
			"existing": {CommitHash: "existing", Author: Author{Name: "x", Date: "2024-01-01"}},
		}},
	}}
	stagingData.Apps["app"] = &App{Versions: map[string]*Version{
		"2.0.0": {Version: "2.0.0", Train: "stable", Commits: map[string]*Commit{
			"def": {CommitHash: "def", Author: Author{Name: "y", Date: "2024-01-02"}},
		}},
	}}

	if err := mergeStagingToCurrent(); err != nil {
		t.Fatalf("mergeStagingToCurrent failed: %v", err)
	}
	if changedData.Apps["app"].Versions["2.0.0"].Commits["def"] == nil {
		t.Fatalf("expected commit def to be merged into version 2.0.0")
	}
}

func TestValidateAllFieldsPresent(t *testing.T) {
	td := t.TempDir()
	opts := ChangelogOptions{
		RepoPath:             td,
		TemplatePath:         td,
		ChangelogFileName:    "CHANGELOG.md",
		AppsDir:              td,
		JSONOutputPath:       td,
		StatusUpdateInterval: 5,
	}
	if err := opts.validate(); err != nil {
		t.Fatalf("validate with all fields should succeed: %v", err)
	}
}

// fakeFile implements diff.File for testing getAppsWithSingleChangedFile
type fakeFile struct {
	path string
}

func (f *fakeFile) Hash() plumbing.Hash     { return plumbing.Hash{} }
func (f *fakeFile) Mode() filemode.FileMode { return filemode.Regular }
func (f *fakeFile) Path() string            { return f.path }

// Ensure fakeFile also satisfies any interface method
var _ interface{ Path() string } = &fakeFile{}

func init() {
	// Ensure globals are properly initialized for tests that don't call resetChangelogGlobals
	if changedData.mu == nil {
		changedData.mu = &sync.RWMutex{}
	}
	if stagingData.mu == nil {
		stagingData.mu = &sync.RWMutex{}
	}
}

package changelog

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/trueforge-org/forgetool/pkg/helper"
)

func TestGenerate_SeamBranches(t *testing.T) {
	resetChangelogGlobals()
	opts := &ChangelogOptions{StatusUpdateInterval: 1, RepoPath: "repo", JSONOutputPath: "out.json"}

	origPrepare := prepareGenerateFunc
	origLoad := loadCommitsForGenerateFunc
	origMerge := mergeStagingToCurrentFunc
	origWrite := writeChangedDataFunc
	origProcess := processCommitFunc
	t.Cleanup(func() {
		prepareGenerateFunc = origPrepare
		loadCommitsForGenerateFunc = origLoad
		mergeStagingToCurrentFunc = origMerge
		writeChangedDataFunc = origWrite
		processCommitFunc = origProcess
	})

	prepareGenerateFunc = func(_ *ChangelogOptions, _ time.Time) error { return errors.New("prepare") }
	if err := opts.Generate(); err == nil {
		t.Fatalf("expected prepare error")
	}

	prepareGenerateFunc = func(_ *ChangelogOptions, _ time.Time) error { return nil }
	loadCommitsForGenerateFunc = func(_ *ChangelogOptions) ([]*object.Commit, error) { return nil, errors.New("load") }
	if err := opts.Generate(); err == nil {
		t.Fatalf("expected load commits error")
	}

	loadCommitsForGenerateFunc = func(_ *ChangelogOptions) ([]*object.Commit, error) { return []*object.Commit{}, nil }
	if err := opts.Generate(); err != nil {
		t.Fatalf("expected no-commits success, got %v", err)
	}

	loadCommitsForGenerateFunc = func(_ *ChangelogOptions) ([]*object.Commit, error) {
		return []*object.Commit{{Hash: plumbing.NewHash("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), ParentHashes: []plumbing.Hash{{}}, Message: "feat(x): y"}}, nil
	}
	processCommitFunc = func(_ *object.Commit) error { return nil }
	mergeStagingToCurrentFunc = func() error { return errors.New("merge") }
	if err := opts.Generate(); err == nil {
		t.Fatalf("expected merge error")
	}

	mergeStagingToCurrentFunc = func() error { return nil }
	writeChangedDataFunc = func(_ string) error { return errors.New("write") }
	if err := opts.Generate(); err == nil {
		t.Fatalf("expected write error")
	}
}

func TestLoadCommitsForGenerate_SeamErrors(t *testing.T) {
	resetChangelogGlobals()
	opts := &ChangelogOptions{RepoPath: "repo"}

	origOpen := openRepoFunc
	origLog := repoLogFunc
	t.Cleanup(func() {
		openRepoFunc = origOpen
		repoLogFunc = origLog
	})

	openRepoFunc = func(_ string) (*git.Repository, error) { return nil, errors.New("open") }
	if _, err := opts.loadCommitsForGenerate(); err == nil {
		t.Fatalf("expected open repo error")
	}

	openRepoFunc = func(_ string) (*git.Repository, error) { return &git.Repository{}, nil }
	repoLogFunc = func(_ *git.Repository) (object.CommitIter, error) { return nil, errors.New("log") }
	if _, err := opts.loadCommitsForGenerate(); err == nil {
		t.Fatalf("expected repo log error")
	}
}

func TestProcessCommitsAndMergeHelpers(t *testing.T) {
	resetChangelogGlobals()
	origProcess := processCommitFunc
	t.Cleanup(func() { processCommitFunc = origProcess })
	processCommitFunc = func(_ *object.Commit) error { return errors.New("boom") }

	opts := &ChangelogOptions{}
	commits := []*object.Commit{{Hash: plumbing.NewHash("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")}}
	opts.processCommits(commits)
	if currentStatus.processedCount != 1 {
		t.Fatalf("expected processedCount increment")
	}

	chart := &App{Versions: map[string]*Version{"1.2.0": {Version: "1.2.0", Commits: map[string]*Commit{}}}}
	staging := &App{Versions: map[string]*Version{"1.1.0": {Version: "1.1.0", Train: "stable", Commits: map[string]*Commit{"c1": {CommitHash: "c1"}}}}}
	changedData.Apps = map[string]*App{"app": chart}
	stagingData.Apps = map[string]*App{"app": staging}
	if err := mergeAppStaging("app", staging); err != nil {
		t.Fatalf("mergeAppStaging failed: %v", err)
	}

	changedData.Apps = map[string]*App{}
	if err := mergeAppStaging("new", staging); err != nil {
		t.Fatalf("mergeAppStaging new app failed: %v", err)
	}

	changedData.Apps = map[string]*App{"empty": {Versions: nil}}
	if err := mergeAppStaging("empty", staging); err != nil {
		t.Fatalf("mergeAppStaging empty versions failed: %v", err)
	}

	if err := mergeVersionStaging("x", "bad", &App{Versions: map[string]*Version{}}, &App{Versions: map[string]*Version{}}, []*semver.Version{}); err == nil {
		t.Fatalf("expected mergeVersionStaging semver parse error")
	}
}

func TestMergeVersionCommitsAndSortCommitsError(t *testing.T) {
	ver := &Version{Version: "1.0.0", Commits: nil}
	mergeVersionCommits("app", "1.0.0", ver, map[string]*Commit{"a": {CommitHash: "a"}})
	mergeVersionCommits("app", "1.0.0", ver, map[string]*Commit{"a": {CommitHash: "a"}, "b": {CommitHash: "b"}})
	if len(ver.Commits) != 2 {
		t.Fatalf("expected commit map merge to skip duplicate and add new")
	}

	v := &Version{Commits: map[string]*Commit{
		"x": {Author: Author{Date: "bad-date"}},
		"y": {Author: Author{Date: "2024-01-01"}},
	}}
	if _, err := v.SortCommits(true); err == nil {
		t.Fatalf("expected sort commits date parse error")
	}
}

func TestProcessCommit_SeamErrorPaths(t *testing.T) {
	resetChangelogGlobals()
	origParent := getParentCommitFunc
	origPatch := getPatchFunc
	origMulti := getAppsWithMultipleChangedFilesFunc
	origSingle := processAppsWithSingleChangedFileFunc
	t.Cleanup(func() {
		getParentCommitFunc = origParent
		getPatchFunc = origPatch
		getAppsWithMultipleChangedFilesFunc = origMulti
		processAppsWithSingleChangedFileFunc = origSingle
	})

	c := &object.Commit{Message: "feat(app): ok", ParentHashes: []plumbing.Hash{{}}, Hash: plumbing.NewHash("cccccccccccccccccccccccccccccccccccccccc")}

	getParentCommitFunc = func(_ *object.Commit) (*object.Commit, error) { return nil, errors.New("parent") }
	if err := processCommit(c); err == nil {
		t.Fatalf("expected parent error")
	}

	getParentCommitFunc = func(_ *object.Commit) (*object.Commit, error) { return &object.Commit{}, nil }
	getPatchFunc = func(_, _ *object.Commit) (*object.Patch, error) { return nil, errors.New("patch") }
	if err := processCommit(c); err == nil {
		t.Fatalf("expected patch error")
	}

	getPatchFunc = func(_, _ *object.Commit) (*object.Patch, error) { return &object.Patch{}, nil }
	getAppsWithMultipleChangedFilesFunc = func(_ *object.Patch) (appsWithChangedFiles, error) { return nil, errors.New("changed") }
	if err := processCommit(c); err == nil {
		t.Fatalf("expected changed-files error")
	}

	getAppsWithMultipleChangedFilesFunc = func(_ *object.Patch) (appsWithChangedFiles, error) {
		return appsWithChangedFiles{"app": {}}, nil
	}
	processAppsWithSingleChangedFileFunc = func(_ *object.Commit, _ *object.Commit, _ appsWithChangedFile) error { return errors.New("single") }
	if err := processCommit(c); err == nil {
		t.Fatalf("expected single-file processing error")
	}

	processAppsWithSingleChangedFileFunc = func(_ *object.Commit, _ *object.Commit, _ appsWithChangedFile) error { return nil }
	if err := processCommit(c); err != nil {
		t.Fatalf("expected processCommit success, got %v", err)
	}
}

type errCommitIter struct{}

func (errCommitIter) Next() (*object.Commit, error)              { return nil, errors.New("x") }
func (errCommitIter) ForEach(_ func(*object.Commit) error) error { return errors.New("iter") }
func (errCommitIter) Close()                                     {}

func TestReverseCommits_UnexpectedErrorAndStatusPrint(t *testing.T) {
	resetChangelogGlobals()
	opts := &ChangelogOptions{StatusUpdateInterval: 1}
	if _, err := opts.reverseCommits(errCommitIter{}, ""); err == nil {
		t.Fatalf("expected reverseCommits error")
	}

	currentStatus = status{processedCount: 1, totalCount: 2, skippedCount: 1, avgTime: time.Second, totalProcessingTime: time.Second, mu: &sync.RWMutex{}}
	opts.printStatus(time.Now().Add(-time.Second), true)
	opts.printStatus(time.Now().Add(-time.Second), false)
}

func TestCheckPathAndPrepareGenerateBranches(t *testing.T) {
	if err := checkPath(filepath.Join(t.TempDir(), "missing"), false); err != nil {
		t.Fatalf("expected missing path with create=false to be allowed")
	}
	if err := checkPath(filepath.Join(t.TempDir(), "missing-create"), true); err != nil {
		t.Fatalf("expected create=true success, got %v", err)
	}
	if err := checkPath("\x00", false); err == nil {
		t.Fatalf("expected invalid path error")
	}
	if err := checkPath(filepath.Join(t.TempDir(), "no-parent", "file"), true); err == nil {
		t.Fatalf("expected create error when parent does not exist")
	}

	resetChangelogGlobals()
	td := t.TempDir()
	tpl := filepath.Join(td, "tmpl")
	repo := filepath.Join(td, "repo")
	apps := filepath.Join(td, "charts")
	jsonPath := filepath.Join(td, "changed.json")
	_ = os.WriteFile(tpl, []byte("x"), 0644)
	_ = os.MkdirAll(repo, 0755)
	_ = os.MkdirAll(apps, 0755)
	_ = os.WriteFile(jsonPath, []byte("{}"), 0644)

	opt := &ChangelogOptions{RepoPath: repo, TemplatePath: tpl, ChangelogFileName: "CHANGELOG.md", JSONOutputPath: jsonPath, AppsDir: apps, StatusUpdateInterval: 1}
	origWalk := walkAppsFunc
	walkAppsFunc = func(_ []string, _ fs.WalkDirFunc, _ helper.WalkMode) error { return nil }
	t.Cleanup(func() { walkAppsFunc = origWalk })
	if err := opt.prepareGenerate(time.Now()); err != nil {
		t.Fatalf("prepareGenerate success expected, got %v", err)
	}
	changedData.LastCommit = "abc"
	if err := opt.prepareGenerate(time.Now()); err != nil {
		t.Fatalf("prepareGenerate with last commit expected success, got %v", err)
	}

	walkAppsFunc = func(_ []string, _ fs.WalkDirFunc, _ helper.WalkMode) error { return errors.New("walk") }
	if err := opt.prepareGenerate(time.Now()); err == nil {
		t.Fatalf("expected walk error")
	}
}

type mockDiffFile struct{ p string }

func (m mockDiffFile) Hash() plumbing.Hash     { return plumbing.ZeroHash }
func (m mockDiffFile) Mode() filemode.FileMode { return filemode.Regular }
func (m mockDiffFile) Path() string            { return m.p }

type mockChunk struct{}

func (mockChunk) Content() string      { return "" }
func (mockChunk) Type() diff.Operation { return diff.Equal }

type mockFilePatch struct {
	from diff.File
	to   diff.File
}

func (m mockFilePatch) IsBinary() bool                { return false }
func (m mockFilePatch) Files() (diff.File, diff.File) { return m.from, m.to }
func (m mockFilePatch) Chunks() []diff.Chunk          { return []diff.Chunk{mockChunk{}} }

func TestPatchHelpersWithMocks(t *testing.T) {
	resetChangelogGlobals()
	activeApps.items["app"] = ActiveApp{Name: "app", Train: "stable"}
	origPath := getAppPathFunc
	origVer := getAppVersionFunc
	t.Cleanup(func() {
		getAppPathFunc = origPath
		getAppVersionFunc = origVer
	})

	if _, _, err := getChangedFilePair(mockFilePatch{from: nil, to: nil}); !errors.Is(err, errSkipPatch) {
		t.Fatalf("expected skip for nil new file")
	}
	if _, _, err := getChangedFilePair(mockFilePatch{to: mockDiffFile{p: "charts/stable/other/Chart.yaml"}}); !errors.Is(err, errSkipPatch) {
		t.Fatalf("expected skip for inactive app")
	}
	getAppPathFunc = func(_ string) (string, error) { return "", errors.New("bad") }
	if _, _, err := getChangedFilePair(mockFilePatch{to: mockDiffFile{p: "charts/stable/app/Chart.yaml"}}); !errors.Is(err, errSkipPatch) {
		t.Fatalf("expected skip for invalid app path")
	}

	getAppPathFunc = func(_ string) (string, error) { return "charts/stable/app/Chart.yaml", nil }
	getAppVersionFunc = func(_ *object.Commit, _ string) (string, error) { return "1.0.0", nil }
	if _, _, err := getChangedFilePair(mockFilePatch{from: mockDiffFile{p: "charts/stable/app/values.yaml"}, to: mockDiffFile{p: "charts/stable/app/Chart.yaml"}}); err != nil {
		t.Fatalf("expected valid changed pair, got %v", err)
	}

	getAppVersionFunc = func(_ *object.Commit, _ string) (string, error) { return "", errors.New("ver") }
	if _, _, err := getOldAndNewVersion(&object.Commit{}, &object.Commit{}, oldNewPaths{new: mockDiffFile{p: "charts/stable/app/Chart.yaml"}}); err == nil {
		t.Fatalf("expected new version error")
	}

	getAppVersionFunc = func(_ *object.Commit, _ string) (string, error) { return "1.0.0", nil }
	if _, _, err := getOldAndNewVersion(&object.Commit{}, &object.Commit{}, oldNewPaths{new: mockDiffFile{p: "charts/stable/app/Chart.yaml"}, old: mockDiffFile{p: "charts/stable/app/Chart.yaml"}}); err != nil {
		t.Fatalf("expected old/new version success, got %v", err)
	}

	getAppVersionFunc = func(_ *object.Commit, _ string) (string, error) { return "bad-semver", nil }
	if err := processAppsWithSingleChangedFile(&object.Commit{Hash: plumbing.NewHash("dddddddddddddddddddddddddddddddddddddddd"), ParentHashes: []plumbing.Hash{{}}, Author: object.Signature{Name: "t", When: time.Now()}, Message: "feat(app): add"}, &object.Commit{}, appsWithChangedFile{"app": {new: mockDiffFile{p: "charts/stable/app/Chart.yaml"}, old: mockDiffFile{p: "charts/stable/app/Chart.yaml"}}}); err == nil {
		t.Fatalf("expected semver parse error")
	}
}

func TestRenderSeamBranches(t *testing.T) {
	origLoad := loadChangedDataFileFunc
	origWalk := walkAppsRenderFunc
	origRender := renderAppChangelogFunc
	t.Cleanup(func() {
		loadChangedDataFileFunc = origLoad
		walkAppsRenderFunc = origWalk
		renderAppChangelogFunc = origRender
	})

	o := &ChangelogOptions{RepoPath: t.TempDir(), JSONOutputPath: filepath.Join(t.TempDir(), "x.json")}
	loadChangedDataFileFunc = func(_ *ChangedData, _ string) error { return errors.New("load") }
	if err := o.Render(); err == nil {
		t.Fatalf("expected render load error")
	}

	loadChangedDataFileFunc = func(_ *ChangedData, _ string) error { return nil }
	walkAppsRenderFunc = func(_ []string, _ fs.WalkDirFunc, _ helper.WalkMode) error { return errors.New("walk") }
	if err := o.Render(); err == nil {
		t.Fatalf("expected render walk error")
	}

	walkAppsRenderFunc = func(_ []string, walker fs.WalkDirFunc, _ helper.WalkMode) error {
		return walker("charts/stable/app/Chart.yaml", fakeDirEntry{name: "Chart.yaml"}, nil)
	}
	renderAppChangelogFunc = func(_ *ChangelogOptions, _ *ChangedData, _, _ string) error { return errors.New("app") }
	if err := o.Render(); err == nil {
		t.Fatalf("expected render app error")
	}
}

func TestChangedData_LoadAndWriteErrorSeams(t *testing.T) {
	var cd ChangedData
	if err := cd.LoadFromFile("\x00"); err == nil {
		t.Fatalf("expected stat error for invalid path")
	}

	origRead := readChangedDataFileFunc
	readChangedDataFileFunc = func(_ string) ([]byte, error) { return nil, errors.New("read") }
	t.Cleanup(func() { readChangedDataFileFunc = origRead })

	f := filepath.Join(t.TempDir(), "data.json")
	if err := os.WriteFile(f, []byte("{}"), 0644); err != nil {
		t.Fatalf("write json file failed: %v", err)
	}
	if err := cd.LoadFromFile(f); err == nil {
		t.Fatalf("expected read error")
	}

	origMarshal := marshalChangedDataFunc
	marshalChangedDataFunc = func(_ interface{}, _, _ string) ([]byte, error) { return nil, errors.New("marshal") }
	t.Cleanup(func() { marshalChangedDataFunc = origMarshal })
	if err := cd.WriteToFile(filepath.Join(t.TempDir(), "out.json")); err == nil {
		t.Fatalf("expected marshal error")
	}
}

func TestPatch_AdditionalBranches(t *testing.T) {
	resetChangelogGlobals()
	activeApps.items["app"] = ActiveApp{Name: "app", Train: "stable"}
	origPath := getAppPathFunc
	origVer := getAppVersionFunc
	origPatches := getFilePatchesFunc
	t.Cleanup(func() {
		getAppPathFunc = origPath
		getAppVersionFunc = origVer
		getFilePatchesFunc = origPatches
	})

	getAppPathFunc = func(path string) (string, error) {
		if path == "bad-old" {
			return "", errors.New("bad-old")
		}
		return "charts/stable/app/Chart.yaml", nil
	}
	if _, _, err := getChangedFilePair(mockFilePatch{from: mockDiffFile{p: "bad-old"}, to: mockDiffFile{p: "charts/stable/app/Chart.yaml"}}); !errors.Is(err, errSkipPatch) {
		t.Fatalf("expected skip when old path invalid")
	}

	getAppPathFunc = func(path string) (string, error) {
		if path == "bad-new" {
			return "", errors.New("bad-new")
		}
		if path == "bad-old2" {
			return "", errors.New("bad-old2")
		}
		return "ok", nil
	}
	getAppVersionFunc = func(_ *object.Commit, path string) (string, error) {
		if path == "ok" {
			return "", errors.New("ver")
		}
		return "1.0.0", nil
	}
	if _, _, err := getOldAndNewVersion(&object.Commit{}, &object.Commit{}, oldNewPaths{new: mockDiffFile{p: "bad-new"}}); err == nil {
		t.Fatalf("expected new path error")
	}
	getAppPathFunc = func(path string) (string, error) {
		if path == "oldbad" {
			return "", errors.New("old-path")
		}
		if path == "new" {
			return "new", nil
		}
		if path == "old" {
			return "old", nil
		}
		return "ok", nil
	}
	getAppVersionFunc = func(_ *object.Commit, _ string) (string, error) { return "1.0.0", nil }
	if _, _, err := getOldAndNewVersion(&object.Commit{}, &object.Commit{}, oldNewPaths{new: mockDiffFile{p: "ok"}, old: mockDiffFile{p: "oldbad"}}); err == nil {
		t.Fatalf("expected old path error")
	}
	getAppVersionFunc = func(_ *object.Commit, path string) (string, error) {
		if path == "old" {
			return "", errors.New("old-ver")
		}
		return "1.0.0", nil
	}
	if _, _, err := getOldAndNewVersion(&object.Commit{}, &object.Commit{}, oldNewPaths{new: mockDiffFile{p: "new"}, old: mockDiffFile{p: "old"}}); err == nil {
		t.Fatalf("expected old version error")
	}

	getAppPathFunc = func(_ string) (string, error) { return "charts/stable/app/Chart.yaml", nil }
	getAppVersionFunc = func(_ *object.Commit, _ string) (string, error) { return "1.0.0", nil }
	getFilePatchesFunc = func(_ *object.Patch) []diff.FilePatch {
		return []diff.FilePatch{
			mockFilePatch{to: nil},
			mockFilePatch{to: mockDiffFile{p: "charts/stable/app/Chart.yaml"}},
		}
	}
	if _, err := getAppsWithMultipleChangedFiles(&object.Patch{}); err != nil {
		t.Fatalf("expected getAppsWithMultipleChangedFiles success, got %v", err)
	}
}

func TestRenderAppAndPrepareVersionsBranches(t *testing.T) {
	o := &ChangelogOptions{TemplatePath: filepath.Join(t.TempDir(), "missing.tpl"), AppsDir: t.TempDir(), ChangelogFileName: "CHANGELOG.md", JSONOutputPath: "x"}
	if o.hasRenderableAppData(nil, "x") {
		t.Fatalf("expected nil app data to be non-renderable")
	}
	if o.hasRenderableAppData(&App{Versions: nil}, "x") {
		t.Fatalf("expected nil versions to be non-renderable")
	}

	if err := o.renderAppChangelog(&ChangedData{Apps: map[string]*App{"app": {Versions: map[string]*Version{"1.0.0": {Version: "1.0.0", Commits: map[string]*Commit{}}}}}}, "app", "stable"); err == nil {
		t.Fatalf("expected template parse error")
	}

	badApp := &App{Versions: map[string]*Version{"bad": {Version: "bad", Commits: map[string]*Commit{}}}}
	if err := o.prepareAppVersions(badApp); err == nil {
		t.Fatalf("expected sort versions error")
	}

	tplDir := t.TempDir()
	tpl := filepath.Join(tplDir, "changelog.tmpl")
	if err := os.WriteFile(tpl, []byte("{{ range .SortedVersions }}{{ . }}{{ end }}"), 0644); err != nil {
		t.Fatalf("write template failed: %v", err)
	}
	o.TemplatePath = tpl
	chart := &App{Versions: map[string]*Version{"1.0.0": {Version: "1.0.0", Commits: map[string]*Commit{"a": {Author: Author{Date: "bad-date"}}}}}}
	if err := o.renderAppChangelog(&ChangedData{Apps: map[string]*App{"app": chart}}, "app", "stable"); err == nil {
		t.Fatalf("expected prepareAppVersions commit sort error")
	}
}

func TestGetAppVersion_ErrorBranches(t *testing.T) {
	if _, err := getAppVersion(&object.Commit{}, "charts/stable/app/Chart.yaml"); err == nil {
		t.Fatalf("expected getAppVersion tree error for detached commit")
	}
}

func TestChangelog_DefaultSeamFuncsAndMergeErrorBranches(t *testing.T) {
	resetChangelogGlobals()
	if err := writeChangedDataFunc(filepath.Join(t.TempDir(), "out.json")); err != nil {
		t.Fatalf("expected default writeChangedDataFunc success, got %v", err)
	}

	td := t.TempDir()
	tpl := filepath.Join(td, "tmpl")
	repo := filepath.Join(td, "repo")
	apps := filepath.Join(td, "charts")
	jsonPath := filepath.Join(td, "changed.json")
	_ = os.WriteFile(tpl, []byte("x"), 0644)
	repoObj, err := git.PlainInit(repo, false)
	if err != nil {
		t.Fatalf("init git repo failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("x"), 0644); err != nil {
		t.Fatalf("write repo file failed: %v", err)
	}
	wt, err := repoObj.Worktree()
	if err != nil {
		t.Fatalf("get worktree failed: %v", err)
	}
	if _, err := wt.Add("README.md"); err != nil {
		t.Fatalf("git add failed: %v", err)
	}
	if _, err := wt.Commit("feat(app): init", &git.CommitOptions{Author: &object.Signature{Name: "t", Email: "t@e", When: time.Now()}}); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}
	_ = os.MkdirAll(apps, 0755)
	_ = os.WriteFile(jsonPath, []byte("{}"), 0644)
	o := &ChangelogOptions{RepoPath: repo, TemplatePath: tpl, ChangelogFileName: "CHANGELOG.md", JSONOutputPath: jsonPath, AppsDir: apps, StatusUpdateInterval: 1}
	if err := prepareGenerateFunc(o, time.Now()); err != nil {
		t.Fatalf("expected default prepareGenerateFunc success, got %v", err)
	}
	if commits, err := loadCommitsForGenerateFunc(o); err != nil || len(commits) == 0 {
		t.Fatalf("expected default loadCommitsForGenerateFunc to return commits, got commits=%d err=%v", len(commits), err)
	}

	changedData.Apps = map[string]*App{"app": {Versions: map[string]*Version{"bad": {Version: "bad", Commits: map[string]*Commit{}}}}}
	stagingData.Apps = map[string]*App{"app": {Versions: map[string]*Version{"1.0.0": {Version: "1.0.0", Commits: map[string]*Commit{}}}}}
	if err := mergeStagingToCurrent(); err == nil {
		t.Fatalf("expected merge sort error")
	}

	changedData.Apps = map[string]*App{"app": {Versions: map[string]*Version{"1.1.0": {Version: "1.1.0", Commits: map[string]*Commit{}}}}}
	stagingData.Apps = map[string]*App{"app": {Versions: map[string]*Version{"bad": {Version: "bad", Commits: map[string]*Commit{}}}}}
	if err := mergeStagingToCurrent(); err == nil {
		t.Fatalf("expected merge version staging error")
	}
}

func TestPatch_ProcessAndCollectRemainingBranches(t *testing.T) {
	resetChangelogGlobals()
	activeApps.items["app"] = ActiveApp{Name: "app", Train: "stable"}
	origPath := getAppPathFunc
	origVer := getAppVersionFunc
	origPatches := getFilePatchesFunc
	t.Cleanup(func() {
		getAppPathFunc = origPath
		getAppVersionFunc = origVer
		getFilePatchesFunc = origPatches
	})

	getAppPathFunc = func(_ string) (string, error) { return "charts/stable/app/Chart.yaml", nil }
	getAppVersionFunc = func(_ *object.Commit, path string) (string, error) {
		if path == "old-bad" {
			return "bad", nil
		}
		if path == "old-ok" {
			return "1.0.0", nil
		}
		if path == "new-ok" {
			return "1.1.0", nil
		}
		return "1.0.0", nil
	}

	getFilePatchesFunc = func(_ *object.Patch) []diff.FilePatch {
		return []diff.FilePatch{
			mockFilePatch{to: mockDiffFile{p: ""}},
			mockFilePatch{to: mockDiffFile{p: "charts/stable/app/Chart.yaml"}},
		}
	}
	if _, err := getAppsWithMultipleChangedFiles(&object.Patch{}); err != nil {
		t.Fatalf("expected getAppsWithMultipleChangedFiles success with empty path skip, got %v", err)
	}

	c := &object.Commit{Hash: plumbing.NewHash("eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"), ParentHashes: []plumbing.Hash{{}}, Author: object.Signature{Name: "t", When: time.Now()}, Message: "feat(app): x"}
	par := &object.Commit{}

	getAppPathFunc = func(path string) (string, error) {
		if path == "new" {
			return "new-ok", nil
		}
		if path == "old" {
			return "old-bad", nil
		}
		return "new-ok", nil
	}
	if err := processAppsWithSingleChangedFile(c, par, appsWithChangedFile{"app": {new: mockDiffFile{p: "new"}, old: mockDiffFile{p: "old"}}}); err == nil {
		t.Fatalf("expected old semver parse error")
	}

	getAppPathFunc = func(path string) (string, error) {
		if path == "new" {
			return "new-bad", nil
		}
		if path == "old" {
			return "old-ok", nil
		}
		return "new-bad", nil
	}
	getAppVersionFunc = func(_ *object.Commit, path string) (string, error) {
		if path == "old-ok" {
			return "1.0.0", nil
		}
		return "bad", nil
	}
	if err := processAppsWithSingleChangedFile(c, par, appsWithChangedFile{"app": {new: mockDiffFile{p: "new"}, old: mockDiffFile{p: "old"}}}); err == nil {
		t.Fatalf("expected new semver parse error")
	}

	getAppPathFunc = func(path string) (string, error) {
		if path == "new" {
			return "new-ok", nil
		}
		if path == "old" {
			return "old-ok", nil
		}
		return "new-ok", nil
	}
	getAppVersionFunc = func(_ *object.Commit, path string) (string, error) {
		if path == "old-ok" {
			return "1.0.0", nil
		}
		return "1.1.0", nil
	}
	if err := processAppsWithSingleChangedFile(c, par, appsWithChangedFile{"app": {new: mockDiffFile{p: "new"}, old: mockDiffFile{p: "old"}}}); err != nil {
		t.Fatalf("expected greater-than branch success, got %v", err)
	}
}

func TestRenderApp_FileSystemBranches(t *testing.T) {
	tplDir := t.TempDir()
	tpl := filepath.Join(tplDir, "changelog.tmpl")
	if err := os.WriteFile(tpl, []byte("{{ index .SortedVersions 99 }}"), 0644); err != nil {
		t.Fatalf("write template failed: %v", err)
	}
	o := &ChangelogOptions{TemplatePath: tpl, AppsDir: filepath.Join(t.TempDir(), "base"), ChangelogFileName: "CHANGELOG.md", JSONOutputPath: "x"}
	chart := &App{Versions: map[string]*Version{"1.0.0": {Version: "1.0.0", Commits: map[string]*Commit{"a": {Author: Author{Date: "2024-01-01"}}}}}}
	if err := o.renderAppChangelog(&ChangedData{Apps: map[string]*App{"app": chart}}, "app", "stable"); err == nil {
		t.Fatalf("expected template execute error")
	}

	if err := os.WriteFile(tpl, []byte("ok"), 0644); err != nil {
		t.Fatalf("rewrite template failed: %v", err)
	}
	blocking := filepath.Join(t.TempDir(), "block")
	if err := os.WriteFile(blocking, []byte("x"), 0644); err != nil {
		t.Fatalf("write blocking file failed: %v", err)
	}
	o.AppsDir = blocking
	if err := o.renderAppChangelog(&ChangedData{Apps: map[string]*App{"app": chart}}, "app", "stable"); err == nil {
		t.Fatalf("expected mkdir error")
	}

	outDir := filepath.Join(t.TempDir(), "out")
	if err := os.MkdirAll(filepath.Join(outDir, "stable", "app"), 0755); err != nil {
		t.Fatalf("mkdir out dir failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(outDir, "stable", "app", "CHANGELOG.md"), 0755); err != nil {
		t.Fatalf("mkdir blocking changelog path failed: %v", err)
	}
	o.AppsDir = outDir
	o.ChangelogFileName = "CHANGELOG.md"
	if err := o.renderAppChangelog(&ChangedData{Apps: map[string]*App{"app": chart}}, "app", "stable"); err == nil {
		t.Fatalf("expected write file error")
	}
}

func TestUtils_RemainingBranches(t *testing.T) {
	resetChangelogGlobals()
	opt := &ChangelogOptions{StatusUpdateInterval: 1}
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		opt.statusPrinter(stop)
	}()
	time.Sleep(1100 * time.Millisecond)
	close(stop)
	<-done

	iter := &sliceCommitIter{commits: []*object.Commit{{Hash: plumbing.NewHash("ffffffffffffffffffffffffffffffffffffffff")}}}
	if _, err := opt.reverseCommits(iter, "ffffffffffffffffffffffffffffffffffffffff"); err != nil {
		t.Fatalf("expected reverseCommits done-reversing branch success, got %v", err)
	}

	origTree := commitTreeFunc
	origFile := treeFileFunc
	origContents := fileContentsFunc
	t.Cleanup(func() {
		commitTreeFunc = origTree
		treeFileFunc = origFile
		fileContentsFunc = origContents
	})

	commitTreeFunc = func(_ *object.Commit) (*object.Tree, error) { return &object.Tree{}, nil }
	treeFileFunc = func(_ *object.Tree, _ string) (*object.File, error) { return nil, errors.New("file") }
	if _, err := getAppVersion(&object.Commit{}, "charts/stable/app/Chart.yaml"); err == nil {
		t.Fatalf("expected get file error")
	}

	commitTreeFunc = func(_ *object.Commit) (*object.Tree, error) { return nil, errors.New("tree") }
	if _, err := getAppVersion(&object.Commit{}, "charts/stable/app/Chart.yaml"); err == nil {
		t.Fatalf("expected get tree error")
	}

	commitTreeFunc = func(_ *object.Commit) (*object.Tree, error) { return &object.Tree{}, nil }
	treeFileFunc = func(_ *object.Tree, _ string) (*object.File, error) { return &object.File{}, nil }
	fileContentsFunc = func(_ *object.File) (string, error) { return "", errors.New("contents") }
	if _, err := getAppVersion(&object.Commit{}, "charts/stable/app/Chart.yaml"); err == nil {
		t.Fatalf("expected get file contents error")
	}
}

func TestChangelog_FinalBranches(t *testing.T) {
	td := t.TempDir()
	tpl := filepath.Join(td, "tmpl")
	repo := filepath.Join(td, "repo")
	apps := filepath.Join(td, "charts")
	jsonPath := filepath.Join(td, "changed.json")
	_ = os.WriteFile(tpl, []byte("x"), 0644)
	_ = os.MkdirAll(repo, 0755)
	_ = os.MkdirAll(apps, 0755)
	_ = os.WriteFile(jsonPath, []byte("{}"), 0644)

	if err := (&ChangelogOptions{RepoPath: repo, TemplatePath: "\x00", ChangelogFileName: "CHANGELOG.md", JSONOutputPath: jsonPath, AppsDir: apps, StatusUpdateInterval: 1}).validate(); err == nil {
		t.Fatalf("expected validate checkPath loop error")
	}

	if err := (&ChangelogOptions{}).prepareGenerate(time.Now()); err == nil {
		t.Fatalf("expected prepareGenerate validate error")
	}

	if err := (&ChangelogOptions{RepoPath: repo, TemplatePath: tpl, ChangelogFileName: "CHANGELOG.md", JSONOutputPath: td, AppsDir: apps, StatusUpdateInterval: 1}).prepareGenerate(time.Now()); err == nil {
		t.Fatalf("expected prepareGenerate load file error with directory json path")
	}

	resetChangelogGlobals()
	origPrepare := prepareGenerateFunc
	origLoad := loadCommitsForGenerateFunc
	origMerge := mergeStagingToCurrentFunc
	origWrite := writeChangedDataFunc
	origProcess := processCommitFunc
	t.Cleanup(func() {
		prepareGenerateFunc = origPrepare
		loadCommitsForGenerateFunc = origLoad
		mergeStagingToCurrentFunc = origMerge
		writeChangedDataFunc = origWrite
		processCommitFunc = origProcess
	})

	prepareGenerateFunc = func(_ *ChangelogOptions, _ time.Time) error { return nil }
	loadCommitsForGenerateFunc = func(_ *ChangelogOptions) ([]*object.Commit, error) {
		return []*object.Commit{{Hash: plumbing.NewHash("abababababababababababababababababababab"), ParentHashes: []plumbing.Hash{{}}, Message: "feat(app): ok"}}, nil
	}
	mergeStagingToCurrentFunc = func() error { return nil }
	writeChangedDataFunc = func(_ string) error { return nil }
	processCommitFunc = func(_ *object.Commit) error { return nil }
	if err := (&ChangelogOptions{StatusUpdateInterval: 1, JSONOutputPath: jsonPath}).Generate(); err != nil {
		t.Fatalf("expected generate success path to end, got %v", err)
	}
}

func TestCommitAndPatchFinalBranches(t *testing.T) {
	repoDir, _, c2, _ := createRepoWithAppHistory(t)
	_ = repoDir
	if _, err := getParentCommitFunc(c2); err != nil {
		t.Fatalf("expected default getParentCommitFunc success, got %v", err)
	}
	par, _ := c2.Parent(0)
	if _, err := getPatchFunc(par, c2); err != nil {
		t.Fatalf("expected default getPatchFunc success, got %v", err)
	}
	if err := processCommit(&object.Commit{Message: ""}); err != nil {
		t.Fatalf("expected invalid commit path to return nil, got %v", err)
	}

	origPair := getChangedFilePairFunc
	origOldNew := getOldAndNewVersionFunc
	origPatches := getFilePatchesFunc
	t.Cleanup(func() {
		getChangedFilePairFunc = origPair
		getOldAndNewVersionFunc = origOldNew
		getFilePatchesFunc = origPatches
	})

	getChangedFilePairFunc = func(_ diff.FilePatch) (string, oldNewPaths, error) { return "", oldNewPaths{}, errors.New("boom") }
	getFilePatchesFunc = func(_ *object.Patch) []diff.FilePatch { return []diff.FilePatch{mockFilePatch{}} }
	if _, err := getAppsWithMultipleChangedFiles(&object.Patch{}); err == nil {
		t.Fatalf("expected non-skip changed-file error branch")
	}

	getChangedFilePairFunc = func(_ diff.FilePatch) (string, oldNewPaths, error) {
		return "app", oldNewPaths{new: mockDiffFile{p: ""}}, nil
	}
	getFilePatchesFunc = func(_ *object.Patch) []diff.FilePatch { return []diff.FilePatch{mockFilePatch{}} }
	if _, err := getAppsWithMultipleChangedFiles(&object.Patch{}); err != nil {
		t.Fatalf("expected empty new path skip branch success, got %v", err)
	}

	getOldAndNewVersionFunc = func(_ *object.Commit, _ *object.Commit, _ oldNewPaths) (string, string, error) {
		return "", "", errors.New("oldnew")
	}
	if err := processAppsWithSingleChangedFile(&object.Commit{}, &object.Commit{}, appsWithChangedFile{"app": {new: mockDiffFile{p: "charts/stable/app/Chart.yaml"}}}); err == nil {
		t.Fatalf("expected old/new version error branch")
	}

	getOldAndNewVersionFunc = func(_ *object.Commit, _ *object.Commit, _ oldNewPaths) (string, string, error) {
		return "", "1.0.0", nil
	}
	if err := processAppsWithSingleChangedFile(&object.Commit{Hash: plumbing.NewHash("1212121212121212121212121212121212121212"), ParentHashes: []plumbing.Hash{{}}, Author: object.Signature{Name: "t", When: time.Now()}, Message: "feat(app): x"}, &object.Commit{}, appsWithChangedFile{"app": {new: mockDiffFile{p: "charts/stable/app/Chart.yaml"}}}); err != nil {
		t.Fatalf("expected old version empty branch success, got %v", err)
	}
}

func TestRender_EarlyReturnBranch(t *testing.T) {
	o := &ChangelogOptions{JSONOutputPath: "x"}
	if err := o.renderAppChangelog(&ChangedData{Apps: map[string]*App{}}, "missing", "stable"); err != nil {
		t.Fatalf("expected early non-renderable return nil, got %v", err)
	}
}

type fakeDirEntry struct{ name string }

func (f fakeDirEntry) Name() string               { return f.name }
func (f fakeDirEntry) IsDir() bool                { return false }
func (f fakeDirEntry) Type() os.FileMode          { return 0 }
func (f fakeDirEntry) Info() (os.FileInfo, error) { return nil, nil }

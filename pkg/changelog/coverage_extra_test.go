package changelog

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func resetChangelogGlobals() {
	changedData = ChangedData{mu: &sync.RWMutex{}, Charts: make(map[string]*Chart)}
	stagingData = ChangedData{mu: &sync.RWMutex{}, Charts: make(map[string]*Chart)}
	activeCharts = ActiveCharts{items: make(map[string]ActiveChart), mu: &sync.RWMutex{}}
	currentStatus = status{processedCount: 0, totalCount: 0, skippedCount: 0, avgTime: 0, totalProcessingTime: 0, mu: &sync.RWMutex{}}
	skipCommitsWithBadMessage = false
}

func writeFile(t *testing.T, path string, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
}

func commitAll(t *testing.T, repo *git.Repository, message string, when time.Time) plumbing.Hash {
	t.Helper()
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree failed: %v", err)
	}
	if err := wt.AddGlob("."); err != nil {
		t.Fatalf("add glob failed: %v", err)
	}
	h, err := wt.Commit(message, &git.CommitOptions{Author: &object.Signature{Name: "tester", Email: "t@example.com", When: when}})
	if err != nil {
		t.Fatalf("commit failed: %v", err)
	}
	return h
}

func createRepoWithChartHistory(t *testing.T) (string, *object.Commit, *object.Commit, *object.Commit) {
	t.Helper()
	repoDir := t.TempDir()
	repo, err := git.PlainInit(repoDir, false)
	if err != nil {
		t.Fatalf("init repo failed: %v", err)
	}

	chartPath := filepath.Join(repoDir, "charts", "stable", "app", "Chart.yaml")
	valuesPath := filepath.Join(repoDir, "charts", "stable", "app", "values.yaml")

	writeFile(t, chartPath, "name: app\nversion: 1.0.0\n")
	writeFile(t, valuesPath, "foo: bar\n")
	h1 := commitAll(t, repo, "feat(app): initial", time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC))

	writeFile(t, chartPath, "name: app\nversion: 1.1.0\n")
	writeFile(t, valuesPath, "foo: baz\n")
	h2 := commitAll(t, repo, "feat(app): bump", time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC))

	writeFile(t, valuesPath, "foo: qux\n")
	h3 := commitAll(t, repo, "fix(app): values", time.Date(2024, 1, 3, 10, 0, 0, 0, time.UTC))

	c1, _ := repo.CommitObject(h1)
	c2, _ := repo.CommitObject(h2)
	c3, _ := repo.CommitObject(h3)
	return repoDir, c1, c2, c3
}

func TestActiveChartsWalkerAndLookup(t *testing.T) {
	ac := ActiveCharts{items: make(map[string]ActiveChart), mu: &sync.RWMutex{}}

	d := t.TempDir()
	chartPath := filepath.Join(d, "charts", "stable", "app", "Chart.yaml")
	writeFile(t, chartPath, "name: app\nversion: 1.0.0\n")

	entries, err := os.ReadDir(filepath.Dir(chartPath))
	if err != nil {
		t.Fatalf("readdir failed: %v", err)
	}
	var chartEntry os.DirEntry
	for _, e := range entries {
		if e.Name() == "Chart.yaml" {
			chartEntry = e
			break
		}
	}
	if chartEntry == nil {
		t.Fatalf("missing chart entry")
	}

	if err := ac.getActiveChartsWalker(filepath.ToSlash(chartPath), chartEntry, nil); err != nil {
		t.Fatalf("walker failed: %v", err)
	}
	if !ac.isActiveChart("app") {
		t.Fatalf("expected app to be active")
	}
	if err := ac.getActiveChartsWalker("a/Chart.yaml", chartEntry, nil); err == nil {
		t.Fatalf("expected invalid short path to fail")
	}
}

func TestCheckPathAndValidate(t *testing.T) {
	td := t.TempDir()
	p := filepath.Join(td, "out.json")
	if err := checkPath(p, true); err != nil {
		t.Fatalf("checkPath(create=true) failed: %v", err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("expected created file, stat err: %v", err)
	}

	opts := ChangelogOptions{}
	if err := opts.validate(); err == nil {
		t.Fatalf("expected validate to fail for empty fields")
	}

	tpl := filepath.Join(td, "changelog.tmpl")
	repoPath := filepath.Join(td, "repo")
	chartsDir := filepath.Join(repoPath, "charts")
	jsonPath := filepath.Join(td, "changed.json")
	writeFile(t, tpl, "{{ .Name }}")
	if err := os.MkdirAll(chartsDir, os.ModePerm); err != nil {
		t.Fatalf("mkdir charts failed: %v", err)
	}
	writeFile(t, jsonPath, "{}")

	opts = ChangelogOptions{
		RepoPath:             repoPath,
		TemplatePath:         tpl,
		ChangelogFileName:    "CHANGELOG.md",
		JSONOutputPath:       jsonPath,
		ChartsDir:            chartsDir,
		StatusUpdateInterval: 1,
	}
	if err := opts.validate(); err != nil {
		t.Fatalf("expected validate success, got: %v", err)
	}
}

func TestPatchProcessingAndVersionRouting(t *testing.T) {
	resetChangelogGlobals()
	repoDir, _, c2, c3 := createRepoWithChartHistory(t)

	activeCharts.items["app"] = ActiveChart{Name: "app", Train: "stable"}

	par2, err := c2.Parent(0)
	if err != nil {
		t.Fatalf("parent c2 failed: %v", err)
	}
	patch2, err := par2.Patch(c2)
	if err != nil {
		t.Fatalf("patch c2 failed: %v", err)
	}

	multi, err := getChartsWithMultipleChangedFiles(patch2)
	if err != nil {
		t.Fatalf("getChartsWithMultipleChangedFiles failed: %v", err)
	}
	if len(multi["app"]) == 0 {
		t.Fatalf("expected changed files for app in repo %s", repoDir)
	}

	single := getChartsWithSingleChangedFile(multi)
	if !strings.HasSuffix(single["app"].new.Path(), "Chart.yaml") {
		t.Fatalf("expected Chart.yaml to be preferred, got %s", single["app"].new.Path())
	}

	if err := processChartsWithSingleChangedFile(c2, par2, single); err != nil {
		t.Fatalf("processChartsWithSingleChangedFile for c2 failed: %v", err)
	}
	if changedData.Charts["app"] == nil || changedData.Charts["app"].Versions["1.1.0"] == nil {
		t.Fatalf("expected changedData to contain app@1.1.0")
	}

	par3, err := c3.Parent(0)
	if err != nil {
		t.Fatalf("parent c3 failed: %v", err)
	}
	patch3, err := par3.Patch(c3)
	if err != nil {
		t.Fatalf("patch c3 failed: %v", err)
	}
	multi3, err := getChartsWithMultipleChangedFiles(patch3)
	if err != nil {
		t.Fatalf("getChartsWithMultipleChangedFiles for c3 failed: %v", err)
	}
	single3 := getChartsWithSingleChangedFile(multi3)
	if err := processChartsWithSingleChangedFile(c3, par3, single3); err != nil {
		t.Fatalf("processChartsWithSingleChangedFile for c3 failed: %v", err)
	}
	if stagingData.Charts["app"] == nil || stagingData.Charts["app"].Versions["1.1.0"] == nil {
		t.Fatalf("expected stagingData to contain app@1.1.0 for unreleased changes")
	}

}

func TestMergeStagingToCurrent(t *testing.T) {
	resetChangelogGlobals()

	changedData.Charts["app"] = &Chart{Versions: map[string]*Version{
		"1.2.0": {Version: "1.2.0", Train: "stable", Commits: map[string]*Commit{}},
	}}
	stagingData.Charts["app"] = &Chart{Versions: map[string]*Version{
		"1.1.0": {Version: "1.1.0", Train: "stable", Commits: map[string]*Commit{"abc": {CommitHash: "abc", Author: Author{Name: "x", Date: "2024-01-02"}}}},
	}}
	stagingData.Charts["newchart"] = &Chart{Versions: map[string]*Version{
		"0.1.0": {Version: "0.1.0", Train: "stable", Commits: map[string]*Commit{"def": {CommitHash: "def", Author: Author{Name: "y", Date: "2024-01-02"}}}},
	}}

	if err := mergeStagingToCurrent(); err != nil {
		t.Fatalf("mergeStagingToCurrent failed: %v", err)
	}

	if changedData.Charts["app"].Versions["1.1.0"] == nil {
		t.Fatalf("expected app@1.1.0 to be present after merge")
	}
	if changedData.Charts["newchart"] == nil {
		t.Fatalf("expected newchart to be copied from staging")
	}
}

func TestRenderWritesChangelog(t *testing.T) {
	td := t.TempDir()
	repoPath := filepath.Join(td, "repo")
	chartDir := filepath.Join(repoPath, "charts", "stable", "app")
	if err := os.MkdirAll(chartDir, os.ModePerm); err != nil {
		t.Fatalf("mkdir chart dir failed: %v", err)
	}
	writeFile(t, filepath.Join(chartDir, "Chart.yaml"), "name: app\nversion: 1.0.0\n")

	tplPath := filepath.Join(td, "changelog.tmpl")
	writeFile(t, tplPath, "# {{ .Name }}\n{{ .Train }}\n{{ range .SortedVersions }}- {{ . }}\n{{ end }}")

	jsonPath := filepath.Join(td, "changed.json")
	data := ChangedData{mu: &sync.RWMutex{}, Charts: map[string]*Chart{
		"app": {
			Versions: map[string]*Version{
				"1.0.0": {Version: "1.0.0", Train: "stable", Commits: map[string]*Commit{"a": {CommitHash: "a", Author: Author{Name: "t", Date: "2024-01-01"}}}},
			},
		},
	}}
	if err := data.WriteToFile(jsonPath); err != nil {
		t.Fatalf("write json failed: %v", err)
	}

	opts := ChangelogOptions{
		RepoPath:             filepath.Join(repoPath, "charts"),
		TemplatePath:         tplPath,
		ChangelogFileName:    "CHANGELOG.md",
		JSONOutputPath:       jsonPath,
		ChartsDir:            filepath.Join(repoPath, "charts"),
		StatusUpdateInterval: 1,
	}

	if err := opts.Render(); err != nil {
		t.Fatalf("render failed: %v", err)
	}

	outPath := filepath.Join(repoPath, "charts", "stable", "app", "CHANGELOG.md")
	out, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("expected changelog output, got err: %v", err)
	}
	if !strings.Contains(string(out), "app") {
		t.Fatalf("expected output to contain chart name, got: %s", string(out))
	}
}

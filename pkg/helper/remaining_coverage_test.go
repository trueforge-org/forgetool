package helper

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

type errMarshaler struct{}

func (errMarshaler) MarshalYAML() (interface{}, error) { return nil, errors.New("marshal fail") }

func chdirForTest(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v (%s)", args, err, string(out))
	}
	return string(out)
}

func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "a@b.c")
	runGit(t, dir, "config", "user.name", "tester")
	return dir
}

func TestCoverage_GetYesOrNoReadErrorBranch(t *testing.T) {
	resetHelperHooks(t)
	defer resetHelperHooks(t)

	calls := 0
	promptReadStringFn = func(_ *bufio.Reader) (string, error) {
		calls++
		if calls == 1 {
			return "", errors.New("read failed")
		}
		return "y\n", nil
	}
	if !GetYesOrNo("continue?") {
		t.Fatalf("expected true after transient read error")
	}
}

func TestCoverage_CheckSystemTimeBranches(t *testing.T) {
	resetHelperHooks(t)
	defer resetHelperHooks(t)

	checkSystemTimeNTPTimeFn = func(string) (time.Time, error) {
		return time.Time{}, errors.New("ntp fail")
	}
	if !CheckSystemTime() {
		t.Fatalf("expected true on ntp failure path")
	}

	now := time.Now()
	checkSystemTimeNowFn = func() time.Time { return now }
	checkSystemTimeNTPTimeFn = func(string) (time.Time, error) { return now.Add(-1 * time.Second), nil }
	if !CheckSystemTime() {
		t.Fatalf("expected true for within-threshold path")
	}

	exitCode := -1
	checkSystemTimeExitFn = func(code int) { exitCode = code }
	checkSystemTimeNTPTimeFn = func(string) (time.Time, error) { return now.Add(-30 * time.Second), nil }
	if CheckSystemTime() {
		t.Fatalf("expected false for out-of-threshold path")
	}
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
}

func TestCoverage_GitGo(t *testing.T) {
	resetHelperHooks(t)
	defer resetHelperHooks(t)

	repo := initGitRepo(t)
	chdirForTest(t, repo)

	if files, err := GetStagedFiles(); err != nil || len(files) != 0 {
		t.Fatalf("expected empty staged files, got %v err=%v", files, err)
	}

	file := filepath.Join(repo, "a.txt")
	if err := os.WriteFile(file, []byte("one"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := StageFile("a.txt"); err != nil {
		t.Fatalf("StageFile: %v", err)
	}
	if err := StageFiles([]string{"a.txt"}); err != nil {
		t.Fatalf("StageFiles: %v", err)
	}

	staged, err := GetStagedFiles()
	if err != nil || len(staged) == 0 {
		t.Fatalf("GetStagedFiles expected entries, got %v err=%v", staged, err)
	}
	staged2, err := GetGitStagedFiles()
	if err != nil || len(staged2) == 0 {
		t.Fatalf("GetGitStagedFiles expected entries, got %v err=%v", staged2, err)
	}

	if err := StageFile("missing.txt"); err == nil {
		t.Fatalf("expected StageFile error for missing file")
	}
	if err := StageFiles([]string{"missing.txt"}); err == nil {
		t.Fatalf("expected StageFiles error for missing file")
	}

	if err := os.MkdirAll("DEVTRIGGER", 0o755); err != nil {
		t.Fatalf("mkdir DEVTRIGGER: %v", err)
	}
	ignored, err := IsFileIgnored("repositories/abc")
	if err != nil || !ignored {
		t.Fatalf("expected DEVTRIGGER prefix shortcut, ignored=%v err=%v", ignored, err)
	}
	_ = os.RemoveAll("DEVTRIGGER")

	if err := os.WriteFile(".gitignore", []byte("ignored.txt\n"), 0o644); err != nil {
		t.Fatalf("write gitignore: %v", err)
	}
	ignored2, err := IsFileIgnored("ignored.txt")
	if err != nil {
		t.Fatalf("IsFileIgnored ignored.txt err: %v", err)
	}
	_ = ignored2
	if _, err := checkIgnore("not-ignored.txt"); err != nil {
		t.Fatalf("expected checkIgnore exit-code-1 path without error, got: %v", err)
	}

	if err := os.WriteFile("tracked.txt", []byte("v1"), 0o644); err != nil {
		t.Fatalf("write tracked: %v", err)
	}
	if err := StageFile("tracked.txt"); err != nil {
		t.Fatalf("stage tracked: %v", err)
	}
	ok, err := IsFileFullyStaged("tracked.txt")
	if err != nil || !ok {
		t.Fatalf("expected fully staged true, got %v err=%v", ok, err)
	}

	if err := os.MkdirAll("forgetool", 0o755); err != nil {
		t.Fatalf("mkdir forgetool: %v", err)
	}
	if err := os.WriteFile(filepath.Join("forgetool", "tracked.txt"), []byte("v1"), 0o644); err != nil {
		t.Fatalf("write forgetool tracked: %v", err)
	}
	ok, err = IsFileFullyStaged("tracked.txt")
	if err != nil {
		t.Fatalf("expected no error in forgetoolExists branch: %v", err)
	}
	_ = ok

	if err := os.WriteFile("ignored-stage.txt", []byte("x"), 0o644); err != nil {
		t.Fatalf("write ignored-stage: %v", err)
	}
	if err := os.WriteFile(".gitignore", []byte("ignored-stage.txt\n"), 0o644); err != nil {
		t.Fatalf("rewrite gitignore: %v", err)
	}
	ok, err = IsFileFullyStaged("ignored-stage.txt")
	if err != nil {
		t.Fatalf("IsFileFullyStaged ignored path err: %v", err)
	}
	if !ok {
		t.Fatalf("expected true when ignored path is skipped")
	}

	if err := os.WriteFile("tracked.txt", []byte("v2"), 0o644); err != nil {
		t.Fatalf("modify tracked: %v", err)
	}
	ok, err = IsFileFullyStaged("tracked.txt")
	if err != nil {
		t.Fatalf("IsFileFullyStaged changed err: %v", err)
	}
	if ok {
		t.Fatalf("expected fully staged false after unstaged change")
	}

	if _, _, err := getGitRootAndForgetoolExists(); err != nil {
		t.Fatalf("getGitRootAndForgetoolExists: %v", err)
	}
	if _, err := hasUnstagedChanges("tracked.txt"); err != nil {
		t.Fatalf("hasUnstagedChanges: %v", err)
	}
}

func TestCoverage_GitGoErrorPathsOutsideRepo(t *testing.T) {
	resetHelperHooks(t)
	defer resetHelperHooks(t)

	nonRepo := t.TempDir()
	chdirForTest(t, nonRepo)

	if _, err := GetStagedFiles(); err == nil {
		t.Fatalf("expected GetStagedFiles error outside git repo")
	}
	if _, err := GetGitStagedFiles(); err == nil {
		t.Fatalf("expected GetGitStagedFiles error outside git repo")
	}
	if _, err := checkIgnore("file.txt"); err == nil {
		t.Fatalf("expected checkIgnore error outside git repo")
	}
	if _, err := IsFileIgnored("file.txt"); err == nil {
		t.Fatalf("expected IsFileIgnored error outside git repo")
	}
	if _, err := IsFileFullyStaged("file.txt"); err == nil {
		t.Fatalf("expected IsFileFullyStaged error outside git repo")
	}
	if _, _, err := getGitRootAndForgetoolExists(); err == nil {
		t.Fatalf("expected getGitRootAndForgetoolExists error outside git repo")
	}
	if _, err := hasUnstagedChanges("file.txt"); err == nil {
		t.Fatalf("expected hasUnstagedChanges error outside git repo")
	}
}

func TestCoverage_GitHookedErrorBranches(t *testing.T) {
	resetHelperHooks(t)
	defer resetHelperHooks(t)

	checkIgnoreCalls := 0
	checkIgnoreFn = func(file string) (bool, error) {
		checkIgnoreCalls++
		if checkIgnoreCalls == 1 {
			return false, nil
		}
		return false, errors.New("second check failed")
	}
	if _, err := IsFileIgnored("x"); err == nil {
		t.Fatalf("expected IsFileIgnored error from second check")
	}

	repo := initGitRepo(t)
	chdirForTest(t, repo)
	if err := os.WriteFile("f.txt", []byte("x"), 0o644); err != nil {
		t.Fatalf("write f: %v", err)
	}
	if err := StageFile("f.txt"); err != nil {
		t.Fatalf("stage f: %v", err)
	}
	hasUnstagedChangesInGitFn = func(string) (bool, error) { return false, errors.New("diff failed") }
	if _, err := IsFileFullyStaged("f.txt"); err == nil {
		t.Fatalf("expected IsFileFullyStaged error from hasUnstagedChanges hook")
	}
}

func TestCoverage_GitPrecommitExtraBranches(t *testing.T) {
	resetHelperHooks(t)
	defer resetHelperHooks(t)

	tmp := t.TempDir()
	hookGetwdFn = func() (string, error) { return "", errors.New("wd fail") }
	if _, err := IsCurrentDirGitRepo(); err == nil {
		t.Fatalf("expected IsCurrentDirGitRepo getwd error")
	}
	if err := CreateEncrPreCommitHook(); err == nil {
		t.Fatalf("expected CreateEncrPreCommitHook getwd error")
	}

	hookGOOS = "windows"
	path := getPreCommitHookPath(tmp)
	if !strings.HasSuffix(path, "pre-commit.bat") {
		t.Fatalf("expected windows hook path, got %s", path)
	}
	script := buildExecutableHookScript(filepath.Join(tmp, "precommit"))
	if !strings.Contains(script, "@echo off") {
		t.Fatalf("expected windows script")
	}

	hookGOOS = runtime.GOOS
	hookCreateFn = func(string) (*os.File, error) { return nil, errors.New("create fail") }
	if err := writeHookScript(filepath.Join(tmp, "x"), "data"); err == nil {
		t.Fatalf("expected writeHookScript create error")
	}

	hookCreateFn = os.Create
	hookChmodFn = func(string, os.FileMode) error { return errors.New("chmod fail") }
	repo := initGitRepo(t)
	chdirForTest(t, repo)
	if err := os.MkdirAll(filepath.Join(repo, ".git", "hooks"), 0o755); err != nil {
		t.Fatalf("mkdir hooks: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "go.mod"), []byte("module m\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := CreateEncrPreCommitHook(); err == nil {
		t.Fatalf("expected chmod error")
	}
}

func TestCoverage_OtherHelperBranches(t *testing.T) {
	resetHelperHooks(t)
	defer resetHelperHooks(t)

	// helper.go invalid mode branch
	root := t.TempDir()
	chartDir := filepath.Join(root, "x")
	if err := os.MkdirAll(chartDir, 0o755); err != nil {
		t.Fatalf("mkdir chart dir: %v", err)
	}
	chartPath := filepath.Join(chartDir, "Chart.yaml")
	if err := os.WriteFile(chartPath, []byte("apiVersion: v2\n"), 0o644); err != nil {
		t.Fatalf("write chart: %v", err)
	}
	entries, err := os.ReadDir(chartDir)
	if err != nil || len(entries) == 0 {
		t.Fatalf("readdir: %v", err)
	}
	f := getWalkDirFunc(func(string, string) error { return nil }, "b", WalkMode(99), nil)
	if err := f(chartPath, entries[0], nil); err == nil {
		t.Fatalf("expected invalid mode error")
	}

	rootCharts := t.TempDir()
	chdirForTest(t, rootCharts)
	if err := os.MkdirAll(filepath.Join(rootCharts, "charts", "c1"), 0o755); err != nil {
		t.Fatalf("mkdir default charts: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rootCharts, "charts", "c1", "Chart.yaml"), []byte("apiVersion: v2\n"), 0o644); err != nil {
		t.Fatalf("write default chart: %v", err)
	}
	called := 0
	if err := WalkCharts(nil, func(string, string) error { called++; return nil }, "x", SyncMode); err != nil {
		t.Fatalf("WalkCharts default path: %v", err)
	}
	if called != 1 {
		t.Fatalf("expected one default chart action, got %d", called)
	}

	// marshaller encode error
	var buf bytes.Buffer
	if err := MarshalYaml(&buf, errMarshaler{}); err == nil {
		t.Fatalf("expected MarshalYaml error")
	}

	// netvalidate invalid endpoints branch
	if ok, err := IPInRange("10.0.0.2", "10.0.0.1-bad"); err != nil || ok {
		t.Fatalf("expected false,nil for invalid range endpoints, got %v,%v", ok, err)
	}

	// replace write error branches
	td := t.TempDir()
	replaceFile := filepath.Join(td, "replace.txt")
	if err := os.WriteFile(replaceFile, []byte("x"), 0o444); err != nil {
		t.Fatalf("write replace file: %v", err)
	}
	if err := ReplaceInFile(replaceFile, "x", "y"); err == nil {
		t.Fatalf("expected ReplaceInFile write error on readonly file")
	}

	target := filepath.Join(td, "target.txt")
	source := filepath.Join(td, "source.txt")
	if err := os.WriteFile(target, []byte("FROM\nold\nTILL\n"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	if err := os.WriteFile(source, []byte("FROM\nnew\nTILL\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.Chmod(target, 0o444); err != nil {
		t.Fatalf("chmod target: %v", err)
	}
	if err := ReplaceContentBetweenLines(target, source, "FROM", "TILL"); err == nil {
		t.Fatalf("expected ReplaceContentBetweenLines write error on readonly target")
	}

	if err := ReplaceContentBetweenLines(filepath.Join(td, "missing-target.txt"), source, "FROM", "TILL"); err == nil {
		t.Fatalf("expected ReplaceContentBetweenLines open target error")
	}
	replaceBlockContentFn = func(*os.File, string, string, string) ([]string, error) {
		return nil, errors.New("replace block failed")
	}
	if err := ReplaceContentBetweenLines(target, source, "FROM", "TILL"); err == nil {
		t.Fatalf("expected ReplaceContentBetweenLines replaceBlockContent error")
	}

	closedFile, err := os.Open(source)
	if err != nil {
		t.Fatalf("open source: %v", err)
	}
	_ = closedFile.Close()
	if _, err := replaceBlockContent(closedFile, "x", "FROM", "TILL"); err == nil {
		t.Fatalf("expected scanner error for closed file")
	}

	// runcmd non-silent branch
	out, err := RunCommand([]string{"echo", "ok"}, false)
	if err != nil || !strings.Contains(out, "ok") {
		t.Fatalf("RunCommand non-silent expected output, got %q err=%v", out, err)
	}

	// talhelperextract error branches
	oldClusterPath := ClusterPath
	oldTalEnv := TalEnv
	ClusterPath = t.TempDir()
	TalEnv = map[string]string{}
	t.Cleanup(func() {
		ClusterPath = oldClusterPath
		TalEnv = oldTalEnv
	})

	if _, err := CreateIPHostnameMap(); err == nil {
		t.Fatalf("expected CreateIPHostnameMap error when talconfig missing")
	}

	cfg := filepath.Join(ClusterPath, "talos", "talconfig.yaml")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o755); err != nil {
		t.Fatalf("mkdir talos: %v", err)
	}
	if err := os.WriteFile(cfg, []byte("nodes: ["), 0o644); err != nil {
		t.Fatalf("write invalid yaml: %v", err)
	}
	if _, err := CreateIPHostnameMap(); err == nil {
		t.Fatalf("expected CreateIPHostnameMap yaml unmarshal error")
	}

	// tooldocs error branches
	ToolDocs(filepath.Join(td, "missing"), filepath.Join(td, "out"))
	toolDocsProcessFilesFn = func(string, string) error { return nil }
	toolDocsMoveMatchingFilesFn = func(string) error { return errors.New("move failed") }
	ToolDocs(td, filepath.Join(td, "out-hooked"))

	tmpIn := filepath.Join(td, "tmpin")
	outDir := filepath.Join(td, "out2")
	if err := os.MkdirAll(tmpIn, 0o755); err != nil {
		t.Fatalf("mkdir tmpIn: %v", err)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("mkdir outDir: %v", err)
	}
	unreadable := filepath.Join(tmpIn, "forgetool_cmd.md")
	if err := os.WriteFile(unreadable, []byte("## forgetool cmd\n"), 0o000); err != nil {
		t.Fatalf("write unreadable: %v", err)
	}
	readable := filepath.Join(tmpIn, "forgetool_readable.md")
	if err := os.WriteFile(readable, []byte("## forgetool readable\n"), 0o644); err != nil {
		t.Fatalf("write readable: %v", err)
	}
	processFilesWriteToFileFn = func(string, []byte, os.DirEntry) error { return errors.New("write fail") }
	processFilesRemoveFn = func(string) error { return errors.New("remove fail") }
	if err := processFiles(tmpIn, outDir); err != nil {
		t.Fatalf("processFiles should continue on read errors: %v", err)
	}
	processFilesWriteToFileFn = writeToFile
	processFilesRemoveFn = os.Remove
	if err := os.Chmod(unreadable, 0o644); err != nil {
		t.Fatalf("restore unreadable perms: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpIn, "forgetool_remove.md"), []byte("## forgetool remove\n"), 0o644); err != nil {
		t.Fatalf("write remove test file: %v", err)
	}
	processFilesRemoveFn = func(string) error { return errors.New("remove fail") }
	if err := processFiles(tmpIn, outDir); err != nil {
		t.Fatalf("processFiles with remove errors should continue: %v", err)
	}
	processFilesRemoveFn = os.Remove

	badParent := filepath.Join(td, "notdir")
	if err := os.WriteFile(badParent, []byte("x"), 0o644); err != nil {
		t.Fatalf("write bad parent: %v", err)
	}
	entrys, err := os.ReadDir(tmpIn)
	if err == nil && len(entrys) > 0 {
		if err := writeToFile(filepath.Join(badParent, "child", "a.md"), []byte("x"), entrys[0]); err == nil {
			t.Fatalf("expected writeToFile mkdir error")
		}
	}

	if err := os.MkdirAll(filepath.Join(td, "dirpath"), 0o755); err != nil {
		t.Fatalf("mkdir dirpath: %v", err)
	}
	entry := writeDirEntry(t, td, "seed2.md", "x")
	writeToFileMkdirAllFn = func(string, os.FileMode) error { return errors.New("mkdir fail") }
	if err := writeToFile(filepath.Join(td, "mk", "f.md"), []byte("x"), entry); err == nil {
		t.Fatalf("expected writeToFile mkdir error from hook")
	}
	writeToFileMkdirAllFn = os.MkdirAll
	if err := writeToFile(filepath.Join(td, "dirpath"), []byte("x"), entry); err == nil {
		t.Fatalf("expected writeToFile write error when target is dir")
	}

	writeToFileChmodFn = func(string, os.FileMode) error { return errors.New("chmod fail") }
	if err := writeToFile(filepath.Join(td, "okfile.md"), []byte("ok"), entry); err == nil {
		t.Fatalf("expected writeToFile chmod error")
	}

	if err := os.WriteFile(filepath.Join(td, "forgetool.md"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write forgetool.md: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(td, "index.md"), 0o755); err != nil {
		t.Fatalf("mkdir index.md dir: %v", err)
	}
	if err := renameForgetoolToIndex(td); err == nil {
		t.Fatalf("expected renameForgetoolToIndex error when destination is dir")
	}

	mv := filepath.Join(td, "move")
	if err := os.MkdirAll(filepath.Join(mv, "sub"), 0o755); err != nil {
		t.Fatalf("mkdir move sub: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mv, "sub.md"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write move file: %v", err)
	}
	moveMatchingRenameFn = func(string, string) error { return errors.New("rename fail") }
	if err := moveMatchingFilesToSubdirs(mv); err == nil {
		t.Fatalf("expected moveMatchingFilesToSubdirs rename error")
	}

	// yamlutil encode error branch
	bad := YamlNewEncoder(&bytes.Buffer{})
	if err := bad.Encode(errMarshaler{}); err == nil {
		t.Fatalf("expected Yaml encoder error for unsupported type")
	}
}

func TestCoverage_CopyAndEnvsubstHookedBranches(t *testing.T) {
	resetHelperHooks(t)
	defer resetHelperHooks(t)

	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	dst := filepath.Join(tmp, "dst")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatalf("write src file: %v", err)
	}

	copyRelFn = func(string, string) (string, error) { return "", errors.New("rel fail") }
	if err := copyDirInternal(src, dst, false, ""); err == nil {
		t.Fatalf("expected copyDirInternal rel error")
	}

	copyRelFn = filepath.Rel
	copyShouldSkipByFilterFn = func(os.FileInfo, string, *regexp.Regexp) (bool, error) {
		return false, errors.New("skip fail")
	}
	if err := copyDirInternal(src, dst, false, ""); err == nil {
		t.Fatalf("expected copyDirInternal skip error")
	}

	copyShouldSkipByFilterFn = shouldSkipByFilter
	copyPathEntryFn = func(string, string, os.FileInfo, bool) error { return errors.New("copy entry fail") }
	if err := copyDirInternal(src, dst, false, ""); err == nil {
		t.Fatalf("expected copyDirInternal copyPathEntry error")
	}

	if err := os.MkdirAll(filepath.Join(tmp, "deny"), 0o000); err != nil {
		t.Fatalf("mkdir deny: %v", err)
	}
	defer os.Chmod(filepath.Join(tmp, "deny"), 0o755)
	if err := CopyFile(filepath.Join(src, "a.txt"), filepath.Join(tmp, "deny", "x.txt"), false); err == nil {
		t.Fatalf("expected CopyFile stat error for inaccessible destination dir")
	}

	envFile := filepath.Join(tmp, "env.txt")
	if err := os.WriteFile(envFile, []byte("KEY='unterminated"), 0o644); err != nil {
		t.Fatalf("write env bad: %v", err)
	}
	out := map[string]string{}
	if err := LoadEnvFromFile(tmp, out); err == nil {
		t.Fatalf("expected LoadEnvFromFile read error when path is a directory")
	}
	if err := LoadEnvFromFile(envFile, out); err == nil {
		t.Fatalf("expected LoadEnvFromFile LoadEnv error")
	}

	if err := os.MkdirAll(filepath.Join(tmp, "blocked"), 0o000); err != nil {
		t.Fatalf("mkdir blocked: %v", err)
	}
	defer os.Chmod(filepath.Join(tmp, "blocked"), 0o755)
	if err := LoadEnvFromFile(filepath.Join(tmp, "blocked", "x"), out); err == nil {
		t.Fatalf("expected LoadEnvFromFile stat error")
	}

	if err := LoadEnv([]byte("KEY='unterminated"), out); err == nil {
		t.Fatalf("expected LoadEnv error")
	}

	stripped := string(StripYamlComment([]byte("# comment only\nKEY=VALUE\n")))
	if strings.Contains(stripped, "comment") {
		t.Fatalf("expected comment-only line to be removed")
	}
	quoted := string(StripYamlComment([]byte("KEY#value\n")))
	if !strings.Contains(quoted, "KEY") {
		t.Fatalf("expected key-like line to be preserved")
	}

	if err := os.Chmod(envFile, 0o444); err != nil {
		t.Fatalf("chmod envFile readonly: %v", err)
	}
	if _, err := EnvSubst(envFile, map[string]string{"KEY": "V"}); err == nil {
		t.Fatalf("expected EnvSubst write error")
	}

	root := filepath.Join(tmp, "root")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir root: %v", err)
	}
	b := filepath.Join(root, "bad.yaml")
	if err := os.WriteFile(b, []byte("KEY=${A}"), 0o444); err != nil {
		t.Fatalf("write recursive bad file: %v", err)
	}
	if err := EnvSubstRecursive(root, `\.yaml$`, map[string]string{"A": "x"}); err == nil {
		t.Fatalf("expected EnvSubstRecursive processing error")
	}
}

func TestCoverage_GitPrecommitRemainingBranches(t *testing.T) {
	resetHelperHooks(t)
	defer resetHelperHooks(t)

	base := t.TempDir()
	repo := filepath.Join(base, "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git", "hooks"), 0o755); err != nil {
		t.Fatalf("mkdir repo hooks: %v", err)
	}
	chdirForTest(t, repo)

	hookStatFn = func(string) (os.FileInfo, error) { return nil, errors.New("stat fail") }
	if _, err := IsCurrentDirGitRepo(); err == nil {
		t.Fatalf("expected IsCurrentDirGitRepo stat error")
	}

	resetHelperHooks(t)
	chdirForTest(t, repo)
	calls := 0
	hookGetwdFn = func() (string, error) {
		calls++
		if calls == 1 {
			return repo, nil
		}
		return "", errors.New("getwd fail")
	}
	if err := CreateEncrPreCommitHook(); err == nil {
		t.Fatalf("expected CreateEncrPreCommitHook second getwd error")
	}

	resetHelperHooks(t)
	chdirForTest(t, repo)
	buildPreCommitHookScriptFn = func(string) (string, error) { return "", errors.New("script build fail") }
	if err := CreateEncrPreCommitHook(); err == nil {
		t.Fatalf("expected buildHookFileData error")
	}

	resetHelperHooks(t)
	chdirForTest(t, repo)
	hookCreateFn = func(string) (*os.File, error) { return nil, errors.New("create fail") }
	if err := CreateEncrPreCommitHook(); err == nil {
		t.Fatalf("expected writeHookScript create error")
	}

	resetHelperHooks(t)
	chdirForTest(t, repo)
	hookWriteStringFn = func(*os.File, string) (int, error) { return 0, errors.New("write fail") }
	if err := CreateEncrPreCommitHook(); err == nil {
		t.Fatalf("expected writeHookScript write error")
	}

	resetHelperHooks(t)
	chdirForTest(t, repo)
	hookChmodFn = func(string, os.FileMode) error { return errors.New("chmod fail") }
	if err := CreateEncrPreCommitHook(); err == nil {
		t.Fatalf("expected chmod error")
	}
}

func TestCoverage_HelperFatalBranches_Subprocess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_FATAL_BRANCH") == "1" {
		fn := getWalkDirFunc(func(string, string) error { return errors.New("boom") }, "x", SyncMode, nil)
		tmp := filepath.Join(os.TempDir(), fmt.Sprintf("helper-fatal-%d", time.Now().UnixNano()))
		_ = os.MkdirAll(tmp, 0o755)
		p := filepath.Join(tmp, "Chart.yaml")
		_ = os.WriteFile(p, []byte("apiVersion: v2\n"), 0o644)
		defer os.RemoveAll(tmp)
		d, _ := os.ReadDir(tmp)
		for _, e := range d {
			if e.Name() == filepath.Base(p) {
				_ = fn(p, e, nil)
				break
			}
		}
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestCoverage_HelperFatalBranches_Subprocess")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_FATAL_BRANCH=1")
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected subprocess to exit non-zero due log.Fatal")
	}
}

func TestCoverage_HelperFatalBranchesAsync_Subprocess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_FATAL_BRANCH_ASYNC") == "1" {
		var wg sync.WaitGroup
		fn := getWalkDirFunc(func(string, string) error { return errors.New("boom") }, "x", AsyncMode, &wg)
		tmp := filepath.Join(os.TempDir(), fmt.Sprintf("helper-fatal-async-%d", time.Now().UnixNano()))
		_ = os.MkdirAll(tmp, 0o755)
		p := filepath.Join(tmp, "Chart.yaml")
		_ = os.WriteFile(p, []byte("apiVersion: v2\n"), 0o644)
		defer os.RemoveAll(tmp)
		d, _ := os.ReadDir(tmp)
		for _, e := range d {
			if e.Name() == filepath.Base(p) {
				_ = fn(p, e, nil)
				wg.Wait()
				break
			}
		}
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestCoverage_HelperFatalBranchesAsync_Subprocess")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_FATAL_BRANCH_ASYNC=1")
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected async subprocess to exit non-zero due log.Fatal")
	}
}

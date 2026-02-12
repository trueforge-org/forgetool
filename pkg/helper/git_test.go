package helper

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestIsPathIgnored(t *testing.T) {
	prefixes := []string{"repositories", "forgetool/repositories"}
	if !isPathIgnored("repositories/foo", prefixes) {
		t.Fatalf("expected repositories/foo to be ignored by prefix")
	}
	if !isPathIgnored("forgetool/repositories/bar", prefixes) {
		t.Fatalf("expected forgetool/repositories/bar to be ignored by prefix")
	}
	if isPathIgnored("other/path", prefixes) {
		t.Fatalf("did not expect other/path to be ignored")
	}
}

func TestIsCurrentDirGitRepoAndCreateHook(t *testing.T) {
	// preserve cwd and CacheDir
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	origCache := CacheDir
	defer func() { CacheDir = origCache }()

	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	// no .git -> not a repo
	isRepo, err := IsCurrentDirGitRepo()
	if err != nil {
		t.Fatalf("IsCurrentDirGitRepo error: %v", err)
	}
	if isRepo {
		t.Fatalf("expected not a git repo")
	}

	// calling CreateEncrPreCommitHook in non-repo should return nil and not create hooks
	CacheDir = filepath.Join(dir, "cache")
	if err := CreateEncrPreCommitHook(); err != nil {
		t.Fatalf("CreateEncrPreCommitHook failed in non-repo: %v", err)
	}

	// now simulate a git repo by creating .git/hooks
	hooks := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(hooks, 0755); err != nil {
		t.Fatalf("mkdir hooks: %v", err)
	}

	// create go.mod to exercise the go.mod branch
	if err := ioutil.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	// ensure cache dir exists
	CacheDir = filepath.Join(dir, "cache")
	if err := os.MkdirAll(CacheDir, 0755); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}

	if err := CreateEncrPreCommitHook(); err != nil {
		t.Fatalf("CreateEncrPreCommitHook failed: %v", err)
	}

	// check that the hook file exists
	hookName := "pre-commit"
	if runtime.GOOS == "windows" {
		hookName = "pre-commit.bat"
	}
	hookPath := filepath.Join(hooks, hookName)
	fi, err := os.Stat(hookPath)
	if err != nil {
		t.Fatalf("expected hook file at %s, stat error: %v", hookPath, err)
	}
	if fi.Size() == 0 {
		t.Fatalf("hook file is empty")
	}
}

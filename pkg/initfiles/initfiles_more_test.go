package initfiles

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

func TestRunAgainLifecycleAndUpdateGitRepo(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	if err := removeRunAgainFile(); err != nil {
		t.Fatalf("removeRunAgainFile when missing should not fail: %v", err)
	}
	createRunAgainFile()
	if !CheckRunAgainFileExists() {
		t.Fatalf("expected RUNAGAIN to exist after create")
	}
	if err := removeRunAgainFile(); err != nil {
		t.Fatalf("removeRunAgainFile failed: %v", err)
	}
	if CheckRunAgainFileExists() {
		t.Fatalf("expected RUNAGAIN removed")
	}

	oldEnv := helper.TalEnv
	helper.TalEnv = map[string]string{"GITHUB_REPOSITORY": "github.com/acme/repo.git"}
	defer func() { helper.TalEnv = oldEnv }()

	repoFile := filepath.Join("repositories", "git", "this-repo.yaml")
	if err := os.MkdirAll(filepath.Dir(repoFile), 0755); err != nil {
		t.Fatalf("mkdir repos dir failed: %v", err)
	}
	if err := os.WriteFile(repoFile, []byte("url: ssh://REPLACEWITHGITREPO\n"), 0644); err != nil {
		t.Fatalf("write this-repo.yaml failed: %v", err)
	}

	UpdateGitRepo()
	b, err := os.ReadFile(repoFile)
	if err != nil {
		t.Fatalf("read this-repo.yaml failed: %v", err)
	}
	if string(b) == "url: ssh://REPLACEWITHGITREPO\n" {
		t.Fatalf("expected placeholder replaced, got unchanged content")
	}
}

package helper

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsPathIgnored_MatchesPrefix(t *testing.T) {
	prefixes := []string{"repos", "clusters"}
	tests := []struct {
		path    string
		ignored bool
	}{
		{"repos/charts", true},
		{"clusters/main", true},
		{"other/path", false},
		{"", false},
		{"repos", true},
	}
	for _, tt := range tests {
		got := isPathIgnored(tt.path, prefixes)
		if got != tt.ignored {
			t.Errorf("isPathIgnored(%q, ...) = %v, want %v", tt.path, got, tt.ignored)
		}
	}
}

func TestIsPathIgnored_EmptyPrefixes(t *testing.T) {
	if isPathIgnored("anything", nil) {
		t.Fatal("expected false for nil prefixes")
	}
	if isPathIgnored("anything", []string{}) {
		t.Fatal("expected false for empty prefixes")
	}
}

func TestIsCurrentDirGitRepo_WithGitDir(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(td)

	os.MkdirAll(filepath.Join(td, ".git"), 0755)
	isRepo, err := IsCurrentDirGitRepo()
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !isRepo {
		t.Fatal("expected true with .git directory")
	}
}

func TestIsCurrentDirGitRepo_WithGitFile(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(td)

	os.WriteFile(filepath.Join(td, ".git"), []byte("gitdir: /some/path"), 0644)
	isRepo, err := IsCurrentDirGitRepo()
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if isRepo {
		t.Fatal("expected false when .git is a file (not a directory)")
	}
}

func TestIsCurrentDirGitRepo_NoGit(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(td)

	isRepo, err := IsCurrentDirGitRepo()
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if isRepo {
		t.Fatal("expected false with no .git")
	}
}

func TestCreateEncrPreCommitHook_NonRepo(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(td)

	origCache := CacheDir
	defer func() { CacheDir = origCache }()
	CacheDir = filepath.Join(td, "cache")

	err := CreateEncrPreCommitHook()
	if err != nil {
		t.Fatalf("expected nil for non-repo, got: %v", err)
	}
}

func TestCreateEncrPreCommitHook_NoGoMod(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(td)

	origCache := CacheDir
	defer func() { CacheDir = origCache }()
	CacheDir = filepath.Join(td, "cache")
	os.MkdirAll(CacheDir, 0755)

	// Create .git/hooks directory but no go.mod
	os.MkdirAll(filepath.Join(td, ".git", "hooks"), 0755)

	err := CreateEncrPreCommitHook()
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}

	// Verify hook was created
	hookPath := filepath.Join(td, ".git", "hooks", "pre-commit")
	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		t.Fatal("expected pre-commit hook to be created")
	}
}

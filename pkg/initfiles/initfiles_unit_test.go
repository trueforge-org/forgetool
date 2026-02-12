package initfiles

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateRunAgainFile_CreatesFile(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	createRunAgainFile()

	if _, err := os.Stat("RUNAGAIN"); os.IsNotExist(err) {
		t.Fatalf("expected RUNAGAIN file to be created")
	}
}

func TestCreateRunAgainFile_Idempotent(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	createRunAgainFile()
	createRunAgainFile()

	if !CheckRunAgainFileExists() {
		t.Fatalf("expected RUNAGAIN to still exist after double create")
	}
}

func TestRemoveRunAgainFile_WhenFileExists(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	createRunAgainFile()
	if err := removeRunAgainFile(); err != nil {
		t.Fatalf("removeRunAgainFile failed: %v", err)
	}
	if CheckRunAgainFileExists() {
		t.Fatalf("expected RUNAGAIN to be removed")
	}
}

func TestRemoveRunAgainFile_WhenFileMissing(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	if err := removeRunAgainFile(); err != nil {
		t.Fatalf("removeRunAgainFile should not error when file missing: %v", err)
	}
}

func TestCheckRunAgainFileExists_FalseForDirectory(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	// Create a directory named RUNAGAIN instead of a file
	if err := os.Mkdir("RUNAGAIN", 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// os.Stat succeeds for directories, so CheckRunAgainFileExists returns true
	if !CheckRunAgainFileExists() {
		t.Fatalf("expected CheckRunAgainFileExists to return true even for a directory (os.Stat succeeds)")
	}
}

func TestReadFilenamesInDir_SkipsSubdirectories(t *testing.T) {
	td := t.TempDir()
	if err := os.WriteFile(filepath.Join(td, "file1.txt"), []byte("a"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.Mkdir(filepath.Join(td, "subdir"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	got, err := readFilenamesInDir(td)
	if err != nil {
		t.Fatalf("readFilenamesInDir error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 file, got %d", len(got))
	}
	if got[0] != "file1.txt" {
		t.Fatalf("expected file1.txt, got %s", got[0])
	}
}

func TestReadFilenamesInDir_EmptyDir(t *testing.T) {
	td := t.TempDir()

	got, err := readFilenamesInDir(td)
	if err != nil {
		t.Fatalf("readFilenamesInDir error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 files, got %d", len(got))
	}
}

func TestReadFilenamesInDir_NonExistentDir(t *testing.T) {
	_, err := readFilenamesInDir("/nonexistent-path-abc123")
	if err == nil {
		t.Fatalf("expected error for nonexistent directory")
	}
}

func TestGetPubKey_MissingFile(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	_, err := GetPubKey()
	if err == nil {
		t.Fatalf("expected error when age.agekey is missing")
	}
}

func TestGetSecKey_MissingFile(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	_, err := GetSecKey()
	if err == nil {
		t.Fatalf("expected error when age.agekey is missing")
	}
}

func TestGetPubKey_NoPubKeyLine(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	content := "# created: 2026-01-01T00:00:00Z\nAGE-SECRET-KEY-ABC\n"
	if err := os.WriteFile("age.agekey", []byte(content), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := GetPubKey()
	if err == nil {
		t.Fatalf("expected error when public key line is missing")
	}
}

func TestGetSecKey_NoSecKeyLine(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	content := "# created: 2026-01-01T00:00:00Z\n# public key: age1deadbeef\n"
	if err := os.WriteFile("age.agekey", []byte(content), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := GetSecKey()
	if err == nil {
		t.Fatalf("expected error when secret key line is missing")
	}
}

func TestFormatGitURL_AlreadyCorrect(t *testing.T) {
	input := "ssh://git@github.com/user/repo.git"
	got := FormatGitURL(input)
	if got != input {
		t.Fatalf("FormatGitURL(%q) = %q, want %q", input, got, input)
	}
}

func TestFormatGitURL_NoMatch(t *testing.T) {
	// Input that doesn't match the regex should be returned as-is
	input := "not-a-url"
	got := FormatGitURL(input)
	if got != "ssh://git@not-a-url" {
		t.Fatalf("FormatGitURL(%q) = %q", input, got)
	}
}

func TestFormatGitURL_DoubleGitAt(t *testing.T) {
	input := "ssh://git@git@github.com/user/repo.git"
	got := FormatGitURL(input)
	expected := "ssh://git@github.com/user/repo.git"
	if got != expected {
		t.Fatalf("FormatGitURL(%q) = %q, want %q", input, got, expected)
	}
}

func TestFormatGitURL_ColonSeparator(t *testing.T) {
	input := "git@github.com:user/repo.git"
	got := FormatGitURL(input)
	expected := "ssh://git@github.com/user/repo.git"
	if got != expected {
		t.Fatalf("FormatGitURL(%q) = %q, want %q", input, got, expected)
	}
}

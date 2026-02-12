package initfiles

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFormatGitURL(t *testing.T) {
	cases := []struct {
		in  string
		out string
	}{
		{"https://gitlab.com/user/repo.git", "ssh://git@gitlab.com/user/repo.git"},
		{"gitlab.com/user/repo.git", "ssh://git@gitlab.com/user/repo.git"},
		{"ssh://git@gitlab.com/user/repo.git", "ssh://git@gitlab.com/user/repo.git"},
		{"ssh://gitlab.com/user/repo.git", "ssh://git@gitlab.com/user/repo.git"},
	}

	for _, c := range cases {
		got := FormatGitURL(c.in)
		if got != c.out {
			t.Fatalf("FormatGitURL(%q) = %q, want %q", c.in, got, c.out)
		}
	}
}

func TestCheckRunAgainFileExists(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	// initially not present
	if CheckRunAgainFileExists() {
		t.Fatalf("expected RUNAGAIN to not exist")
	}

	// create file and check
	path := filepath.Join(td, "RUNAGAIN")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if !CheckRunAgainFileExists() {
		t.Fatalf("expected RUNAGAIN to exist")
	}

	// remove and check
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if CheckRunAgainFileExists() {
		t.Fatalf("expected RUNAGAIN to not exist after remove")
	}
}

func TestGetPubKeyAndSecKey_FromAgeFile(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	content := `# created: 2026-01-01T00:00:00Z
# public key: age1deadbeef
AGE-SECRET-KEY-1: some-secret
AGE-SECRET-KEY-2: other-secret
AGE-SECRET-KEY-ABC
AGE-SECRET-KEY-EXACT` + "\n"
	if err := os.WriteFile("age.agekey", []byte(content), 0o600); err != nil {
		t.Fatalf("write age file: %v", err)
	}

	pub, err := GetPubKey()
	if err != nil {
		t.Fatalf("GetPubKey error: %v", err)
	}
	if pub != "age1deadbeef" {
		t.Fatalf("unexpected pub: %s", pub)
	}

	// GetSecKey expects a line starting with AGE-SECRET-KEY-
	sec, err := GetSecKey()
	if err != nil {
		t.Fatalf("GetSecKey error: %v", err)
	}
	if sec == "" {
		t.Fatalf("expected a secret key, got empty")
	}
}

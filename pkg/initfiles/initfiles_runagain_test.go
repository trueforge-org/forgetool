package initfiles

import (
	"os"
	"testing"
)

func TestRunAgainLifecycle(t *testing.T) {
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
}

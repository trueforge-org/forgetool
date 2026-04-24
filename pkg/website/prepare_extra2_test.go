package website

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWipeAndRestore_MkdirAllError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses perms")
	}
	root := t.TempDir()
	parent := filepath.Join(root, "ro")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	// Make parent read-only so MkdirAll for docsDir inside it fails;
	// docsDir does not exist so RemoveAll succeeds with nil.
	if err := os.Chmod(parent, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o755) })
	docsDir := filepath.Join(parent, "docs")
	err := wipeAndRestore(docsDir)
	if err == nil || !strings.Contains(err.Error(), "recreate") {
		t.Fatalf("expected recreate (mkdir) error, got %v", err)
	}
}

func TestWipeAndRestore_RestoreWriteFileError(t *testing.T) {
	// Override preservedIndexNames so a name with a path separator is used:
	// after wipe + recreate, writing to "<docsDir>/sub/index.md" fails because
	// "sub" no longer exists.
	prev := preservedIndexNames
	preservedIndexNames = []string{filepath.Join("sub", "index.md")}
	t.Cleanup(func() { preservedIndexNames = prev })

	root := t.TempDir()
	docsDir := filepath.Join(root, "docs")
	writeFile(t, filepath.Join(docsDir, "sub", "index.md"), "preserved")
	err := wipeAndRestore(docsDir)
	if err == nil || !strings.Contains(err.Error(), "restore") {
		t.Fatalf("expected restore error, got %v", err)
	}
}

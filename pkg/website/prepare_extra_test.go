package website

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareContainerWebsite_MkdirAllError(t *testing.T) {
	root := t.TempDir()
	// Make WebsiteDir a regular file → MkdirAll for sub-paths fails.
	makeFileAsDir(t, filepath.Join(root, "ws"))
	if err := PrepareContainerWebsite(ContainerOptions{WebsiteDir: filepath.Join(root, "ws")}); err == nil {
		t.Fatal("expected mkdir error")
	}
}

func TestPrepareChartWebsite_MkdirAllError(t *testing.T) {
	root := t.TempDir()
	makeFileAsDir(t, filepath.Join(root, "ws"))
	if err := PrepareChartWebsite(ChartOptions{WebsiteDir: filepath.Join(root, "ws")}); err == nil {
		t.Fatal("expected mkdir error")
	}
}

func TestWipeAndRestore_PreserveReadError(t *testing.T) {
	root := t.TempDir()
	docsDir := filepath.Join(root, "docs")
	// index.md is a directory → ReadFile returns non-NotExist error.
	if err := os.MkdirAll(filepath.Join(docsDir, "index.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := wipeAndRestore(docsDir); err == nil {
		t.Fatal("expected read error")
	}
}

func TestWipeAndRestore_RemoveError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses perms")
	}
	root := t.TempDir()
	docsDir := filepath.Join(root, "parent", "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// chmod the parent of docsDir so RemoveAll can't traverse into docsDir.
	parent := filepath.Join(root, "parent")
	if err := os.Chmod(parent, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o755) })
	if err := wipeAndRestore(docsDir); err == nil {
		t.Fatal("expected remove error")
	}
}

func TestCopyChangelogs_OnlyFiles(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "logs")
	writeFile(t, filepath.Join(src, "loose.md"), "x")
	dst := filepath.Join(root, "dst")
	if err := copyChangelogs(src, dst); err != nil {
		t.Fatal(err)
	}
	// Since hasDir is false, dst should NOT have been created/copied.
	if _, err := os.Stat(filepath.Join(dst, "loose.md")); err == nil {
		t.Fatal("expected no copy when no subdirectories")
	}
}

func TestCopyChangelogs_ReadDirError(t *testing.T) {
	root := t.TempDir()
	// changelogsDir is a regular file → ReadDir returns ENOTDIR (not NotExist).
	makeFileAsDir(t, filepath.Join(root, "blocker"))
	if err := copyChangelogs(filepath.Join(root, "blocker", "child"), filepath.Join(root, "dst")); err == nil {
		t.Fatal("expected readdir error")
	}
}

func TestCopyChangelogs_MkdirAllError(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "logs")
	writeFile(t, filepath.Join(src, "sub", "x.md"), "y")
	// dst parent is a file → MkdirAll fails.
	makeFileAsDir(t, filepath.Join(root, "blocker"))
	dst := filepath.Join(root, "blocker", "child")
	if err := copyChangelogs(src, dst); err == nil {
		t.Fatal("expected mkdir error")
	}
}

package website

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeFileAsDir creates a regular file at p so that any subsequent attempt to
// treat it as a directory parent fails with ENOTDIR (not ErrNotExist).
func makeFileAsDir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestKeepDocsSafe_MkdirAllError(t *testing.T) {
	root := t.TempDir()
	docsDir := filepath.Join(root, "docs")
	tmpDirParent := filepath.Join(root, "tmp")
	makeFileAsDir(t, tmpDirParent) // tmp is a regular file
	if err := keepDocsSafe(docsDir, tmpDirParent, "myapp"); err == nil {
		t.Fatal("expected mkdir error")
	}
}

func TestKeepDocsSafe_StatNotExistError(t *testing.T) {
	root := t.TempDir()
	docsDir := filepath.Join(root, "docs")
	tmpDir := filepath.Join(root, "tmp")
	// docsDir/myapp/CHANGELOG.md → make CHANGELOG.md path traverse a file
	// to force a non-NotExist stat error.
	makeFileAsDir(t, filepath.Join(docsDir, "myapp"))
	if err := keepDocsSafe(docsDir, tmpDir, "myapp"); err == nil {
		t.Fatal("expected stat error from invalid parent")
	}
}

func TestKeepDocsSafe_RenameError(t *testing.T) {
	root := t.TempDir()
	docsDir := filepath.Join(root, "docs")
	tmpDir := filepath.Join(root, "tmp")
	item := "myapp"
	writeFile(t, filepath.Join(docsDir, item, "CHANGELOG.md"), "x")
	// Create tmpDir/item as a file blocking the rename target dir creation.
	// Actually we already MkdirAll'd before; force rename to fail by making
	// tmp/item/CHANGELOG.md a directory so rename of file → dir fails on macOS.
	if err := os.MkdirAll(filepath.Join(tmpDir, item, "CHANGELOG.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Put a child inside so rename can't replace it with a regular file.
	writeFile(t, filepath.Join(tmpDir, item, "CHANGELOG.md", "child"), "y")
	if err := keepDocsSafe(docsDir, tmpDir, item); err == nil {
		t.Fatal("expected rename error")
	}
}

func TestRestoreSafeDocs_StatNotExistError(t *testing.T) {
	root := t.TempDir()
	docsDir := filepath.Join(root, "docs")
	tmpDir := filepath.Join(root, "tmp")
	// tmp/myapp/CHANGELOG.md path crosses a file → ENOTDIR on Stat
	makeFileAsDir(t, filepath.Join(tmpDir, "myapp"))
	if err := restoreSafeDocs(docsDir, tmpDir, "myapp"); err == nil {
		t.Fatal("expected stat error")
	}
}

func TestRestoreSafeDocs_MkdirAllError(t *testing.T) {
	root := t.TempDir()
	docsDir := filepath.Join(root, "docs")
	tmpDir := filepath.Join(root, "tmp")
	item := "myapp"
	writeFile(t, filepath.Join(tmpDir, item, "CHANGELOG.md"), "log")
	// Make docsDir/myapp a regular file so MkdirAll(filepath.Dir(dst)) fails.
	makeFileAsDir(t, filepath.Join(docsDir, item))
	if err := restoreSafeDocs(docsDir, tmpDir, item); err == nil {
		t.Fatal("expected mkdir error")
	}
}

func TestRestoreSafeDocs_RenameError(t *testing.T) {
	root := t.TempDir()
	docsDir := filepath.Join(root, "docs")
	tmpDir := filepath.Join(root, "tmp")
	item := "myapp"
	writeFile(t, filepath.Join(tmpDir, item, "CHANGELOG.md"), "log")
	// Pre-existing non-empty directory blocks rename of file → dir.
	if err := os.MkdirAll(filepath.Join(docsDir, item, "CHANGELOG.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(docsDir, item, "CHANGELOG.md", "child"), "x")
	if err := restoreSafeDocs(docsDir, tmpDir, item); err == nil {
		t.Fatal("expected rename error")
	}
}

func TestResetDir_MkdirAllError(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "parent")
	makeFileAsDir(t, parent)
	// reset parent/child where parent is a file → MkdirAll fails.
	if err := resetDir(filepath.Join(parent, "child")); err == nil {
		t.Fatal("expected mkdir error")
	}
}

func TestCopyTreeContents_MkdirAllError(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	writeFile(t, filepath.Join(src, "a.txt"), "A")
	parent := filepath.Join(root, "blocker")
	makeFileAsDir(t, parent)
	if err := copyTreeContents(src, filepath.Join(parent, "dst")); err == nil {
		t.Fatal("expected mkdir error")
	}
}

func TestCopyTreeContents_CopyFileError(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	writeFile(t, filepath.Join(src, "a.txt"), "A")
	// Pre-create dst/a.txt as a directory to make CopyFile's Create fail.
	if err := os.MkdirAll(filepath.Join(dst, "a.txt"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyTreeContents(src, dst); err == nil {
		t.Fatal("expected copy file error")
	}
}

func TestCopyTreeContents_CopyDirError(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	writeFile(t, filepath.Join(src, "sub", "b.txt"), "B")
	// Pre-create dst/sub/b.txt as a directory to make CopyDir's CopyFile fail.
	if err := os.MkdirAll(filepath.Join(dst, "sub", "b.txt"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyTreeContents(src, dst); err == nil {
		t.Fatal("expected copy dir error")
	}
}

func TestCopyFileIfExists_StatNonNotExistError(t *testing.T) {
	root := t.TempDir()
	blocker := filepath.Join(root, "blocker")
	makeFileAsDir(t, blocker)
	// src crosses a file → ENOTDIR
	src := filepath.Join(blocker, "child", "file.txt")
	if err := copyFileIfExists(src, filepath.Join(root, "dst.txt")); err == nil {
		t.Fatal("expected stat error")
	}
}

func TestCopyFileIfExists_MkdirAllError(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src.txt")
	writeFile(t, src, "x")
	blocker := filepath.Join(root, "blocker")
	makeFileAsDir(t, blocker)
	dst := filepath.Join(blocker, "child", "dst.txt")
	if err := copyFileIfExists(src, dst); err == nil {
		t.Fatal("expected mkdir error")
	}
}

func TestDownloadFile_NewRequestError(t *testing.T) {
	// URL with a control character makes http.NewRequest fail.
	if err := downloadFile("http://exa\x7fmple.com", filepath.Join(t.TempDir(), "x")); err == nil {
		t.Fatal("expected request build error")
	}
}

func TestDownloadFile_DoError(t *testing.T) {
	// Invalid scheme path causes httpClient.Do to fail (no transport).
	if err := downloadFile("garbage://nope", filepath.Join(t.TempDir(), "x")); err == nil {
		t.Fatal("expected do error")
	}
}

func TestDownloadFile_MkdirAllError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()
	root := t.TempDir()
	blocker := filepath.Join(root, "blocker")
	makeFileAsDir(t, blocker)
	dst := filepath.Join(blocker, "child", "out")
	if err := downloadFile(srv.URL, dst); err == nil {
		t.Fatal("expected mkdir error")
	}
}

func TestEnsureFrontMatter_ReadError(t *testing.T) {
	if _, err := ensureFrontMatter(filepath.Join(t.TempDir(), "missing.md")); err == nil {
		t.Fatal("expected read error")
	}
}

func TestEnsureFrontMatter_WriteError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses write perms")
	}
	dir := t.TempDir()
	f := filepath.Join(dir, "doc.md")
	writeFile(t, f, "# Title\nbody\n")
	// Make file itself read-only so the rewrite fails.
	if err := os.Chmod(f, 0o400); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(f, 0o644) })
	if _, err := ensureFrontMatter(f); err == nil {
		t.Fatal("expected write error")
	}
}

func TestExtractFrontMatterTitle_OpenError(t *testing.T) {
	if _, err := extractFrontMatterTitle(filepath.Join(t.TempDir(), "missing.md")); err == nil {
		t.Fatal("expected open error")
	}
}

func TestExtractFrontMatterTitle_NoFrontMatter(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "doc.md")
	writeFile(t, f, "hello world\n---\ntitle: x\n---\n")
	got, err := extractFrontMatterTitle(f)
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("expected empty title, got %q", got)
	}
}

func TestExtractFrontMatterTitle_ClosingMarker(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "doc.md")
	// title key absent; loop terminates via the closing --- branch.
	writeFile(t, f, "---\nfoo: bar\n---\nbody\n")
	got, err := extractFrontMatterTitle(f)
	if err != nil || got != "" {
		t.Fatalf("got=%q err=%v", got, err)
	}
}

func TestCollectDocsLinks_MissingDir(t *testing.T) {
	links, err := collectDocsLinks(filepath.Join(t.TempDir(), "nope"))
	if err != nil || links != nil {
		t.Fatalf("expected nil, nil, got %v %v", links, err)
	}
}

func TestCollectDocsLinks_ReadDirError(t *testing.T) {
	root := t.TempDir()
	blocker := filepath.Join(root, "blocker")
	makeFileAsDir(t, blocker)
	if _, err := collectDocsLinks(filepath.Join(blocker, "child")); err == nil {
		// child crosses a file → ENOTDIR which is not NotExist
		t.Fatal("expected readdir error")
	}
}

func TestCollectDocsLinks_FiltersAndSkips(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "skip.txt"), "no")
	if err := os.MkdirAll(filepath.Join(dir, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "index.mdx"), "---\ntitle: x\n---\n")
	writeFile(t, filepath.Join(dir, "notitle.md"), "no front matter\n")
	writeFile(t, filepath.Join(dir, "emptytitle.md"), "---\nfoo: bar\n---\n")
	writeFile(t, filepath.Join(dir, "ok.mdx"), "---\ntitle: OK\n---\n")
	links, err := collectDocsLinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 1 || links[0].Title != "OK" {
		t.Fatalf("unexpected links: %+v", links)
	}
}

func TestReadReadmeBody_SkipBeyondEnd(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "README.md")
	writeFile(t, f, "only-one-line\n")
	body, err := readReadmeBody(f, 100)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(body) != "" {
		t.Fatalf("expected empty body, got %q", body)
	}
}

func TestReadReadmeBody_ReadError(t *testing.T) {
	dir := t.TempDir()
	// dir as path triggers a non-NotExist read error
	if _, err := readReadmeBody(dir, 0); err == nil {
		t.Fatal("expected read error")
	}
}

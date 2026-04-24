package website

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResetDir_MkdirAllError_ChmodParent(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses perms")
	}
	root := t.TempDir()
	parent := filepath.Join(root, "ro")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(parent, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o755) })
	if err := resetDir(filepath.Join(parent, "child")); err == nil {
		t.Fatal("expected mkdir error")
	}
}

func TestCopyTreeContents_ReadDirOtherError(t *testing.T) {
	root := t.TempDir()
	// Make the source path traverse a regular file → ENOTDIR (not NotExist).
	makeFileAsDir(t, filepath.Join(root, "blocker"))
	if err := copyTreeContents(filepath.Join(root, "blocker", "child"), filepath.Join(root, "dst")); err == nil {
		t.Fatal("expected readdir error")
	}
}

func TestDownloadFile_CreateError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	}))
	t.Cleanup(srv.Close)
	root := t.TempDir()
	dst := filepath.Join(root, "out.bin")
	// Pre-create dst as a directory so os.Create fails.
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := downloadFile(srv.URL, dst); err == nil {
		t.Fatal("expected create error")
	}
}

func TestEnsureFrontMatter_ScannerError(t *testing.T) {
	root := t.TempDir()
	f := filepath.Join(root, "doc.md")
	// File starts with no front-matter and contains a single very long line
	// that exceeds the bufio.Scanner buffer (1 MiB) so scanner.Err() returns
	// bufio.ErrTooLong.
	long := strings.Repeat("x", 1024*1024+16)
	writeFile(t, f, long)
	if _, err := ensureFrontMatter(f); err == nil {
		t.Fatal("expected scanner error")
	}
}

func TestExtractFrontMatterTitle_LeadingBlankLine(t *testing.T) {
	root := t.TempDir()
	f := filepath.Join(root, "doc.md")
	// Leading blank line → "continue" branch in extractFrontMatterTitle is
	// taken before the front-matter block opens.
	writeFile(t, f, "\n---\ntitle: Hello\n---\nbody\n")
	title, err := extractFrontMatterTitle(f)
	if err != nil {
		t.Fatal(err)
	}
	if title != "Hello" {
		t.Fatalf("got %q", title)
	}
}

func TestCollectDocsLinks_EnsureFrontMatterError(t *testing.T) {
	root := t.TempDir()
	// Candidate file with a >1 MiB single line so ensureFrontMatter returns
	// a scanner error.
	long := strings.Repeat("y", 1024*1024+16)
	writeFile(t, filepath.Join(root, "huge.md"), long)
	if _, err := collectDocsLinks(root); err == nil {
		t.Fatal("expected ensureFrontMatter error to bubble up")
	}
}

func TestCollectDocsLinks_ExtractTitleError(t *testing.T) {
	root := t.TempDir()
	// File starts with `---` so ensureFrontMatter returns ok=true without
	// modifying the file. extractFrontMatterTitle then scans line by line and
	// fails on the >1 MiB line.
	long := strings.Repeat("z", 1024*1024+16)
	content := "---\n" + long + "\n---\n"
	writeFile(t, filepath.Join(root, "huge.md"), content)
	if _, err := collectDocsLinks(root); err == nil {
		t.Fatal("expected extractFrontMatterTitle error to bubble up")
	}
}

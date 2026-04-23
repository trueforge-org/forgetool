package website

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func TestEnsureFrontMatter_AlreadyHasFrontMatter(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "doc.md")
	writeFile(t, f, "---\ntitle: Existing\n---\nbody\n")
	ok, err := ensureFrontMatter(f)
	if err != nil || !ok {
		t.Fatalf("expected ok, got ok=%v err=%v", ok, err)
	}
	if got := readFile(t, f); !strings.HasPrefix(got, "---\ntitle: Existing") {
		t.Fatalf("file rewritten unexpectedly: %q", got)
	}
}

func TestEnsureFrontMatter_PromoteHeading(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "doc.md")
	writeFile(t, f, "# My Title\nsome body\n")
	ok, err := ensureFrontMatter(f)
	if err != nil || !ok {
		t.Fatalf("expected ok, got ok=%v err=%v", ok, err)
	}
	got := readFile(t, f)
	if !strings.HasPrefix(got, "---\ntitle: My Title\n---\n") {
		t.Fatalf("front matter not added: %q", got)
	}
	if strings.Contains(got, "# My Title") {
		t.Fatalf("original heading still present: %q", got)
	}
}

func TestEnsureFrontMatter_NoTitle(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "doc.md")
	writeFile(t, f, "just some text\n")
	ok, err := ensureFrontMatter(f)
	if err != nil || ok {
		t.Fatalf("expected !ok, got ok=%v err=%v", ok, err)
	}
}

func TestExtractFrontMatterTitle(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "doc.md")
	writeFile(t, f, "---\ntitle: Hello\nfoo: bar\n---\nbody\n")
	title, err := extractFrontMatterTitle(f)
	if err != nil {
		t.Fatal(err)
	}
	if title != "Hello" {
		t.Fatalf("got %q", title)
	}
}

func TestCollectDocsLinks_SortedAndSkipsIndex(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "Beta.md"), "---\ntitle: Beta\n---\n")
	writeFile(t, filepath.Join(dir, "alpha.md"), "---\ntitle: Alpha\n---\n")
	writeFile(t, filepath.Join(dir, "index.md"), "---\ntitle: Index\n---\n")
	writeFile(t, filepath.Join(dir, "notitle.md"), "no front matter\n")

	links, err := collectDocsLinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d (%+v)", len(links), links)
	}
	// Sorted by filename: Beta.md before alpha.md (uppercase < lowercase).
	if links[0].Title != "Beta" || links[0].Slug != "beta" {
		t.Fatalf("unexpected first link: %+v", links[0])
	}
	if links[1].Title != "Alpha" || links[1].Slug != "alpha" {
		t.Fatalf("unexpected second link: %+v", links[1])
	}
}

func TestReadReadmeBody_SkipAndDemote(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "README.md")
	writeFile(t, f, "L1\nL2\nL3\n## Heading\nbody\n")
	body, err := readReadmeBody(f, 3)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "### Heading") {
		t.Fatalf("expected demoted heading, got %q", body)
	}
	if strings.Contains(body, "L1") {
		t.Fatalf("L1 should have been skipped: %q", body)
	}
}

func TestReadReadmeBody_Missing(t *testing.T) {
	body, err := readReadmeBody(filepath.Join(t.TempDir(), "missing.md"), 3)
	if err != nil || body != "" {
		t.Fatalf("expected empty body, got %q err=%v", body, err)
	}
}

func TestKeepAndRestoreSafeDocs(t *testing.T) {
	root := t.TempDir()
	docsDir := filepath.Join(root, "docs")
	tmpDir := filepath.Join(root, "tmp")
	item := "myapp"
	writeFile(t, filepath.Join(docsDir, item, "CHANGELOG.md"), "changelog\n")

	if err := keepDocsSafe(docsDir, tmpDir, item); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(docsDir, item, "CHANGELOG.md")); !os.IsNotExist(err) {
		t.Fatalf("CHANGELOG should have been moved out, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, item, "CHANGELOG.md")); err != nil {
		t.Fatalf("CHANGELOG missing from tmp: %v", err)
	}

	if err := os.RemoveAll(filepath.Join(docsDir, item)); err != nil {
		t.Fatal(err)
	}
	if err := restoreSafeDocs(docsDir, tmpDir, item); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(docsDir, item, "CHANGELOG.md")); got != "changelog\n" {
		t.Fatalf("unexpected restored content: %q", got)
	}
}

func TestDownloadFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte("payload"))
	}))
	defer srv.Close()

	dst := filepath.Join(t.TempDir(), "out", "file.bin")
	if err := downloadFile(srv.URL+"/ok", dst); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, dst); got != "payload" {
		t.Fatalf("unexpected payload: %q", got)
	}
	if err := downloadFile(srv.URL+"/missing", dst); err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestResetDirAndCopyTreeContents(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	writeFile(t, filepath.Join(src, "a.txt"), "A")
	writeFile(t, filepath.Join(src, "sub", "b.txt"), "B")
	writeFile(t, filepath.Join(dst, "old.txt"), "old")

	if err := resetDir(dst); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dst, "old.txt")); !os.IsNotExist(err) {
		t.Fatalf("old.txt should be gone, err=%v", err)
	}
	if err := copyTreeContents(src, dst); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dst, "a.txt")); got != "A" {
		t.Fatalf("unexpected a.txt: %q", got)
	}
	if got := readFile(t, filepath.Join(dst, "sub", "b.txt")); got != "B" {
		t.Fatalf("unexpected sub/b.txt: %q", got)
	}
}

func TestCopyTreeContents_MissingSrcIsNoop(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "dst")
	if err := copyTreeContents(filepath.Join(t.TempDir(), "missing"), dst); err != nil {
		t.Fatal(err)
	}
}

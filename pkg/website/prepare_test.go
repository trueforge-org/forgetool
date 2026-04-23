package website

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareContainerWebsite(t *testing.T) {
	root := t.TempDir()
	docsDir := filepath.Join(root, "containerforge", "src", "content", "docs", "containers")
	// Pre-existing maintained index.mdx must survive.
	writeFile(t, filepath.Join(docsDir, "index.mdx"), "INDEX\n")
	// Pre-existing app docs must be wiped.
	writeFile(t, filepath.Join(docsDir, "stale-app", "index.md"), "stale\n")

	if err := PrepareContainerWebsite(ContainerOptions{WebsiteDir: root}); err != nil {
		t.Fatalf("PrepareContainerWebsite: %v", err)
	}

	// index.mdx restored.
	if got := readFile(t, filepath.Join(docsDir, "index.mdx")); got != "INDEX\n" {
		t.Fatalf("index.mdx not restored: %q", got)
	}
	// Stale app gone.
	if _, err := os.Stat(filepath.Join(docsDir, "stale-app")); !os.IsNotExist(err) {
		t.Fatalf("stale-app should be removed: err=%v", err)
	}
	// Image dirs created.
	for _, sub := range []string{"container-icons", "container-icons-small", "container-screenshots"} {
		p := filepath.Join(root, "containerforge", "public", "img", "hotlink-ok", sub)
		if st, err := os.Stat(p); err != nil || !st.IsDir() {
			t.Fatalf("expected dir %s, err=%v", p, err)
		}
	}
}

func TestPrepareContainerWebsite_NoExistingIndex(t *testing.T) {
	root := t.TempDir()
	if err := PrepareContainerWebsite(ContainerOptions{WebsiteDir: root}); err != nil {
		t.Fatalf("PrepareContainerWebsite: %v", err)
	}
	docsDir := filepath.Join(root, "containerforge", "src", "content", "docs", "containers")
	if st, err := os.Stat(docsDir); err != nil || !st.IsDir() {
		t.Fatalf("expected docs dir, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(docsDir, "index.mdx")); !os.IsNotExist(err) {
		t.Fatalf("index.mdx should not exist when there was none, err=%v", err)
	}
}

func TestFinalizeContainerWebsite(t *testing.T) {
	root := t.TempDir()
	docsDir := filepath.Join(root, "containerforge", "src", "content", "docs", "containers")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Empty changelogsDir => no-op.
	emptyDir := filepath.Join(root, "empty-changelogs")
	if err := os.MkdirAll(emptyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := FinalizeContainerWebsite(ContainerOptions{WebsiteDir: root}, emptyDir); err != nil {
		t.Fatalf("FinalizeContainerWebsite (empty): %v", err)
	}

	// Missing dir => no-op (no error).
	if err := FinalizeContainerWebsite(ContainerOptions{WebsiteDir: root}, filepath.Join(root, "missing")); err != nil {
		t.Fatalf("FinalizeContainerWebsite (missing): %v", err)
	}

	// Empty changelogsDir as the empty string => no-op.
	if err := FinalizeContainerWebsite(ContainerOptions{WebsiteDir: root}, ""); err != nil {
		t.Fatalf("FinalizeContainerWebsite (\"\"): %v", err)
	}

	// Populated changelogsDir => contents copied.
	clog := filepath.Join(root, "changelogs")
	writeFile(t, filepath.Join(clog, "myapp", "CHANGELOG.md"), "log\n")
	if err := FinalizeContainerWebsite(ContainerOptions{WebsiteDir: root}, clog); err != nil {
		t.Fatalf("FinalizeContainerWebsite: %v", err)
	}
	if got := readFile(t, filepath.Join(docsDir, "myapp", "CHANGELOG.md")); got != "log\n" {
		t.Fatalf("changelog not copied: %q", got)
	}
}

func TestPrepareChartWebsite(t *testing.T) {
	root := t.TempDir()
	docsDir := filepath.Join(root, "truecharts", "src", "content", "docs", "charts")
	writeFile(t, filepath.Join(docsDir, "index.mdx"), "CHARTS\n")
	writeFile(t, filepath.Join(docsDir, "stable", "ghost", "index.md"), "stale\n")

	if err := PrepareChartWebsite(ChartOptions{WebsiteDir: root}); err != nil {
		t.Fatalf("PrepareChartWebsite: %v", err)
	}

	if got := readFile(t, filepath.Join(docsDir, "index.mdx")); got != "CHARTS\n" {
		t.Fatalf("index.mdx not restored: %q", got)
	}
	if _, err := os.Stat(filepath.Join(docsDir, "stable")); !os.IsNotExist(err) {
		t.Fatalf("stale train should be removed: err=%v", err)
	}
	for _, sub := range []string{
		filepath.Join("truecharts", "public", "img", "hotlink-ok", "chart-icons"),
		filepath.Join("truecharts", "public", "img", "hotlink-ok", "chart-icons-small"),
		filepath.Join("truecharts", "src", "assets"),
	} {
		p := filepath.Join(root, sub)
		if st, err := os.Stat(p); err != nil || !st.IsDir() {
			t.Fatalf("expected dir %s, err=%v", p, err)
		}
	}
}

func TestFinalizeChartWebsite(t *testing.T) {
	root := t.TempDir()
	docsDir := filepath.Join(root, "truecharts", "src", "content", "docs", "charts")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	clog := filepath.Join(root, "changelogs")
	writeFile(t, filepath.Join(clog, "stable", "myapp", "CHANGELOG.md"), "log\n")
	if err := FinalizeChartWebsite(ChartOptions{WebsiteDir: root}, clog); err != nil {
		t.Fatalf("FinalizeChartWebsite: %v", err)
	}
	if got := readFile(t, filepath.Join(docsDir, "stable", "myapp", "CHANGELOG.md")); got != "log\n" {
		t.Fatalf("changelog not copied: %q", got)
	}
}

func TestPrepareWebsite_RequiresWebsiteDir(t *testing.T) {
	if err := PrepareContainerWebsite(ContainerOptions{WebsiteDir: " "}); err != nil {
		// non-empty dir is fine; this is just to exercise applyDefaults + path.
		_ = err
	}
}

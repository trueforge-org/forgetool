package website

import (
	"os"
	"path/filepath"
	"testing"
)

// containerProcessSetup creates an apps/<app>/docker-bake.hcl plus README
// template, then chdirs into root.
func containerProcessSetup(t *testing.T, root string) string {
	t.Helper()
	app := "myapp"
	writeFile(t, filepath.Join(root, "apps", app, "docker-bake.hcl"), sampleBake)
	writeFile(t, filepath.Join(root, "templates", "README.md.tmpl"), sampleTemplate)
	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	return app
}

func TestProcessApp_ResetDirError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses perms")
	}
	root := t.TempDir()
	app := containerProcessSetup(t, root)
	docsBase := filepath.Join("website", "containerforge", "src", "content", "docs", "containers")
	if err := os.MkdirAll(docsBase, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(docsBase, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(docsBase, 0o755) })
	if err := ProcessApp(ContainerOptions{App: app, IconFallbackBaseURL: "http://127.0.0.1:1"}); err == nil {
		t.Fatal("expected resetDir error")
	}
}

func TestProcessApp_CopyAssetsError(t *testing.T) {
	root := t.TempDir()
	app := containerProcessSetup(t, root)
	// Provide a local icon at source plus a destination icons-dir entry that
	// already exists as a non-empty directory; copyFileIfExists → helper.CopyFile
	// fails when the destination is a directory.
	writeFile(t, filepath.Join(root, "apps", app, "icon.webp"), "icon")
	iconsDir := filepath.Join("website", "containerforge", "public", "img", "hotlink-ok", "container-icons")
	writeFile(t, filepath.Join(iconsDir, app+".webp", "blocker"), "x")
	if err := ProcessApp(ContainerOptions{App: app, IconFallbackBaseURL: "http://127.0.0.1:1"}); err == nil {
		t.Fatal("expected copyContainerAssets error")
	}
}

func TestProcessApp_RestoreSafeDocsError(t *testing.T) {
	root := t.TempDir()
	app := containerProcessSetup(t, root)
	// Source docs include a CHANGELOG.md as a directory so when
	// copyContainerAssets copies docs into the destination the destination
	// CHANGELOG.md path is materialised as a non-empty directory; restoring
	// the saved CHANGELOG.md file then fails on rename.
	writeFile(t, filepath.Join(root, "apps", app, "docs", "CHANGELOG.md", "child"), "x")
	docsBaseApp := filepath.Join("website", "containerforge", "src", "content", "docs", "containers", app)
	writeFile(t, filepath.Join(docsBaseApp, "CHANGELOG.md"), "history\n")
	if err := ProcessApp(ContainerOptions{App: app, IconFallbackBaseURL: "http://127.0.0.1:1"}); err == nil {
		t.Fatal("expected restoreSafeDocs error")
	}
}

func TestProcessContainerIndex_MkdirIndexErrorIsolated(t *testing.T) {
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
	opts := ContainerOptions{App: "myapp", TemplatePath: filepath.Join(root, "tpl"), ComposeTemplatePath: filepath.Join(root, "ctpl")}
	opts.applyDefaults()
	writeFile(t, opts.TemplatePath, sampleTemplate)
	p := containerPaths{
		bake:      filepath.Join(root, "bake.hcl"),
		docsDir:   filepath.Join(root, "no-docs"),
		readme:    filepath.Join(root, "no-readme"),
		settings:  filepath.Join(root, "no-settings"),
		indexFile: filepath.Join(parent, "out", "index.md"),
	}
	writeFile(t, p.bake, sampleBake)
	if err := processContainerIndex(opts, p); err == nil {
		t.Fatal("expected mkdir error")
	}
}

func TestWriteComposePage_TemplateNotExist(t *testing.T) {
	root := t.TempDir()
	opts := ContainerOptions{App: "myapp", AppsDir: root, ComposeTemplatePath: filepath.Join(root, "missing.tmpl")}
	opts.applyDefaults()
	p := containerPaths{
		settings: filepath.Join(root, "settings.yaml"),
		docsDir:  filepath.Join(root, "docs-out"),
	}
	writeFile(t, p.settings, sampleSettings)
	// Pre-create a docker-compose.md file that should be removed.
	writeFile(t, filepath.Join(p.docsDir, "docker-compose.md"), "stale\n")
	if err := writeComposePage(opts, p, map[string]string{"VERSION": "1.0"}); err != nil {
		t.Fatalf("expected nil error when template missing, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(p.docsDir, "docker-compose.md")); !os.IsNotExist(err) {
		t.Fatalf("expected stale page removed, got %v", err)
	}
}

func TestWriteComposePage_BuildComposeYAMLError(t *testing.T) {
	root := t.TempDir()
	app := "myapp"
	appsDir := filepath.Join(root, "apps")
	// Main app settings declare a dependency whose own settings.yaml is
	// invalid, causing dependencyResolver → ParseSettings → BuildComposeYAML
	// to fail.
	writeFile(t, filepath.Join(appsDir, app, "settings.yaml"), "dependencies:\n  - name: dep1\n")
	writeFile(t, filepath.Join(appsDir, "dep1", "settings.yaml"), ":\n\tnot yaml")
	composeTmpl := filepath.Join(root, "templates", "docker-compose.md.tmpl")
	writeFile(t, composeTmpl, sampleComposePageTmpl)
	opts := ContainerOptions{App: app, AppsDir: appsDir, ComposeTemplatePath: composeTmpl}
	opts.applyDefaults()
	p := containerPaths{
		settings: filepath.Join(appsDir, app, "settings.yaml"),
		docsDir:  filepath.Join(root, "docs-out"),
	}
	if err := writeComposePage(opts, p, map[string]string{"VERSION": "1.0"}); err == nil {
		t.Fatal("expected build compose error")
	}
}

func TestDependencyResolver_StatLoopError(t *testing.T) {
	root := t.TempDir()
	appDir := filepath.Join(root, "app")
	writeFile(t, filepath.Join(appDir, "settings.yaml"), sampleSettings)
	// Create a self-referencing symlink at docker-bake.hcl so os.Stat
	// returns ELOOP (a non-NotExist error).
	link := filepath.Join(appDir, "docker-bake.hcl")
	if err := os.Symlink(link, link); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}
	r := dependencyResolver(root)
	if _, _, _, err := r("app"); err == nil {
		t.Fatal("expected stat ELOOP error")
	}
}

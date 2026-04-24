package website

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseBakeVars_OpenError(t *testing.T) {
	if _, err := parseBakeVars(filepath.Join(t.TempDir(), "missing.hcl")); err == nil {
		t.Fatal("expected open error")
	}
}

func TestDiscoverApps_ReadDirError(t *testing.T) {
	if _, err := DiscoverApps(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("expected read error")
	}
}

func TestDiscoverApps_SkipsFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "stray"), "x")
	writeFile(t, filepath.Join(root, "alpha", "docker-bake.hcl"), "")
	apps, err := DiscoverApps(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 1 || apps[0] != "alpha" {
		t.Fatalf("unexpected apps: %v", apps)
	}
}

func TestDependencyResolver_NoSettings(t *testing.T) {
	root := t.TempDir()
	r := dependencyResolver(root)
	_, _, ok, err := r("missing-app")
	if err != nil || ok {
		t.Fatalf("expected !ok, got ok=%v err=%v", ok, err)
	}
}

func TestDependencyResolver_SettingsParseError(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "app", "settings.yaml"), ":\n\tnot yaml")
	r := dependencyResolver(root)
	if _, _, _, err := r("app"); err == nil {
		t.Fatal("expected settings parse error")
	}
}

func TestDependencyResolver_NoBake(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "app", "settings.yaml"), sampleSettings)
	r := dependencyResolver(root)
	settings, ver, ok, err := r("app")
	if err != nil || !ok || ver != "" {
		t.Fatalf("expected ok with empty version, got ver=%q ok=%v err=%v", ver, ok, err)
	}
	if len(settings.Ports) == 0 {
		t.Fatalf("expected settings populated")
	}
}

func TestDependencyResolver_WithBake(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "app", "settings.yaml"), sampleSettings)
	writeFile(t, filepath.Join(root, "app", "docker-bake.hcl"), sampleBake)
	r := dependencyResolver(root)
	_, ver, ok, err := r("app")
	if err != nil || !ok {
		t.Fatalf("expected ok, err=%v", err)
	}
	if ver != "1.2.3" {
		t.Fatalf("got version %q", ver)
	}
}

func TestDependencyResolver_BakeStatError(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "app", "settings.yaml"), sampleSettings)
	// Make bake path traverse a regular file → ENOTDIR
	makeFileAsDir(t, filepath.Join(root, "app", "docker-bake.hcl", "blocker_parent"))
	// Actually simpler: replace docker-bake.hcl path with one that fails Stat.
	// Above made docker-bake.hcl a directory; remove and recreate as broken path.
	_ = os.RemoveAll(filepath.Join(root, "app", "docker-bake.hcl"))
	// Make app/docker-bake.hcl a directory; os.Stat will succeed though.
	// Use a different approach: chmod app dir to 0 to fail Stat.
	if os.Getuid() == 0 {
		t.Skip("root bypasses perms")
	}
	if err := os.Chmod(filepath.Join(root, "app"), 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(root, "app"), 0o755) })
	r := dependencyResolver(root)
	if _, _, _, err := r("app"); err == nil {
		t.Fatal("expected error")
	}
}

func TestDependencyResolver_ParseBakeError(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "app", "settings.yaml"), sampleSettings)
	// Make bake a directory so parseBakeVars (os.Open) fails.
	if err := os.MkdirAll(filepath.Join(root, "app", "docker-bake.hcl"), 0o755); err != nil {
		t.Fatal(err)
	}
	r := dependencyResolver(root)
	if _, _, _, err := r("app"); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestProcessApp_ReadmeBodyError(t *testing.T) {
	root := t.TempDir()
	app := "myapp"
	appDir := filepath.Join(root, "apps", app)
	writeFile(t, filepath.Join(appDir, "docker-bake.hcl"), sampleBake)
	writeFile(t, filepath.Join(root, "templates", "README.md.tmpl"), sampleTemplate)
	// README.md as a directory → readReadmeBody errors out
	if err := os.MkdirAll(filepath.Join(appDir, "README.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	if err := ProcessApp(ContainerOptions{App: app, IconFallbackBaseURL: "http://127.0.0.1:1"}); err == nil {
		t.Fatal("expected readme error")
	}
}

func TestProcessApp_TemplateMissing(t *testing.T) {
	root := t.TempDir()
	app := "myapp"
	writeFile(t, filepath.Join(root, "apps", app, "docker-bake.hcl"), sampleBake)
	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	// No README template.
	if err := ProcessApp(ContainerOptions{App: app, IconFallbackBaseURL: "http://127.0.0.1:1"}); err == nil {
		t.Fatal("expected template read error")
	}
}

func TestProcessApp_StatNonNotExistError(t *testing.T) {
	root := t.TempDir()
	// app/docker-bake.hcl path traverses a file → ENOTDIR on Stat
	makeFileAsDir(t, filepath.Join(root, "apps", "myapp"))
	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	if err := ProcessApp(ContainerOptions{App: "myapp"}); err == nil {
		t.Fatal("expected stat error")
	}
}

func TestWriteComposePage_TemplateReadError(t *testing.T) {
	root := t.TempDir()
	app := "myapp"
	appDir := filepath.Join(root, "apps", app)
	writeFile(t, filepath.Join(appDir, "settings.yaml"), sampleSettings)
	// Make compose template a directory so ReadFile returns a non-NotExist error.
	composeTmpl := filepath.Join(root, "templates", "docker-compose.md.tmpl")
	if err := os.MkdirAll(composeTmpl, 0o755); err != nil {
		t.Fatal(err)
	}
	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	opts := ContainerOptions{App: app, ComposeTemplatePath: composeTmpl}
	opts.applyDefaults()
	p := opts.paths()
	if err := os.MkdirAll(p.docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeComposePage(opts, p, map[string]string{}); err == nil {
		t.Fatal("expected template read error")
	}
}

func TestWriteComposePage_SettingsParseError(t *testing.T) {
	root := t.TempDir()
	app := "myapp"
	writeFile(t, filepath.Join(root, "apps", app, "settings.yaml"), ":\n\tnot yaml")
	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	opts := ContainerOptions{App: app}
	opts.applyDefaults()
	p := opts.paths()
	if err := writeComposePage(opts, p, nil); err == nil {
		t.Fatal("expected settings parse error")
	}
}

func containerPathsForTest(root, app string) (ContainerOptions, containerPaths) {
	opts := ContainerOptions{App: app, AppsDir: filepath.Join(root, "apps"), WebsiteDir: filepath.Join(root, "website"), TemplatePath: filepath.Join(root, "tpl"), ComposeTemplatePath: filepath.Join(root, "ctpl")}
	opts.applyDefaults()
	return opts, opts.paths()
}

func TestCopyContainerAssets_DocsCopyError(t *testing.T) {
	root := t.TempDir()
	opts, p := containerPathsForTest(root, "myapp")
	writeFile(t, filepath.Join(p.docsSrc, "a.md"), "x")
	if err := os.MkdirAll(filepath.Join(p.docsDir, "a.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyContainerAssets(opts, p); err == nil {
		t.Fatal("expected docs copy error")
	}
}

func TestCopyContainerAssets_IconCopyError(t *testing.T) {
	root := t.TempDir()
	opts, p := containerPathsForTest(root, "myapp")
	writeFile(t, filepath.Join(p.appDir, "icon.webp"), "icon")
	makeFileAsDir(t, p.iconsDir)
	if err := copyContainerAssets(opts, p); err == nil {
		t.Fatal("expected icon copy error")
	}
}

func TestCopyContainerAssets_IconSmallCopyError(t *testing.T) {
	root := t.TempDir()
	opts, p := containerPathsForTest(root, "myapp")
	writeFile(t, filepath.Join(p.appDir, "icon-small.webp"), "icon-small")
	// Local icon not present so the local branch isn't taken; small icon is present.
	makeFileAsDir(t, p.iconsSmallDir)
	if err := copyContainerAssets(opts, p); err == nil {
		t.Fatal("expected icon-small copy error")
	}
}

func TestCopyContainerAssets_ScreenshotsCopyError(t *testing.T) {
	root := t.TempDir()
	opts, p := containerPathsForTest(root, "myapp")
	writeFile(t, filepath.Join(p.screenshots, "a.png"), "shot")
	if err := os.MkdirAll(filepath.Join(p.screenshotsDir, "a.png"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyContainerAssets(opts, p); err == nil {
		t.Fatal("expected screenshot copy error")
	}
}

func TestProcessContainerIndex_BakeError(t *testing.T) {
	root := t.TempDir()
	opts, p := containerPathsForTest(root, "myapp")
	if err := os.MkdirAll(p.bake, 0o755); err != nil { // bake as a directory → open fails
		t.Fatal(err)
	}
	if err := processContainerIndex(opts, p); err == nil {
		t.Fatal("expected parse bake error")
	}
}

func TestProcessContainerIndex_ComposeError(t *testing.T) {
	root := t.TempDir()
	opts, p := containerPathsForTest(root, "myapp")
	writeFile(t, p.bake, sampleBake)
	// Make settings.yaml invalid so writeComposePage returns parse error.
	writeFile(t, p.settings, "ports: not a list")
	if err := processContainerIndex(opts, p); err == nil {
		t.Fatal("expected compose error")
	}
}

func TestProcessContainerIndex_LinksError(t *testing.T) {
	root := t.TempDir()
	opts, p := containerPathsForTest(root, "myapp")
	writeFile(t, p.bake, sampleBake)
	// Make docsDir a regular file → ReadDir fails
	makeFileAsDir(t, p.docsDir)
	if err := processContainerIndex(opts, p); err == nil {
		t.Fatal("expected links error")
	}
}

func TestProcessContainerIndex_ReadmeError(t *testing.T) {
	root := t.TempDir()
	opts, p := containerPathsForTest(root, "myapp")
	writeFile(t, p.bake, sampleBake)
	if err := os.MkdirAll(p.readme, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := processContainerIndex(opts, p); err == nil {
		t.Fatal("expected readme error")
	}
}

func TestProcessContainerIndex_TemplateError(t *testing.T) {
	root := t.TempDir()
	opts, p := containerPathsForTest(root, "myapp")
	writeFile(t, p.bake, sampleBake)
	// Template path doesn't exist
	if err := processContainerIndex(opts, p); err == nil {
		t.Fatal("expected template read error")
	}
}

func TestProcessContainerIndex_WriteIndexError(t *testing.T) {
	root := t.TempDir()
	opts, p := containerPathsForTest(root, "myapp")
	writeFile(t, p.bake, sampleBake)
	writeFile(t, opts.TemplatePath, sampleTemplate)
	// indexFile already a directory → WriteFile fails
	if err := os.MkdirAll(p.indexFile, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := processContainerIndex(opts, p); err == nil {
		t.Fatal("expected write index error")
	}
}

func TestWriteComposePage_MkdirDocsDirError(t *testing.T) {
	root := t.TempDir()
	opts, p := containerPathsForTest(root, "myapp")
	writeFile(t, p.settings, sampleSettings)
	writeFile(t, opts.ComposeTemplatePath, sampleComposePageTmpl)
	// Make docsDir parent a file so MkdirAll fails.
	makeFileAsDir(t, filepath.Dir(p.docsDir))
	if err := writeComposePage(opts, p, map[string]string{"VERSION": "1.0"}); err == nil {
		t.Fatal("expected mkdir error")
	}
}

func TestProcessApp_KeepDocsSafeError(t *testing.T) {
	root := t.TempDir()
	app := "myapp"
	writeFile(t, filepath.Join(root, "apps", app, "docker-bake.hcl"), sampleBake)
	writeFile(t, filepath.Join(root, "templates", "README.md.tmpl"), sampleTemplate)
	// WebsiteDir exists as a regular file → docsBase paths inside fail with ENOTDIR.
	wsFile := filepath.Join(root, "ws-file")
	makeFileAsDir(t, wsFile)
	// Create the safe doc inside the file path? Impossible. Instead, prepopulate
	// docsBase/<app>/CHANGELOG.md so keepDocsSafe must Stat it; with WebsiteDir
	// as a file the Stat will fail with ENOTDIR.
	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	if err := ProcessApp(ContainerOptions{
		App:                 app,
		AppsDir:             "apps",
		WebsiteDir:          "ws-file",
		TemplatePath:        "templates/README.md.tmpl",
		IconFallbackBaseURL: "http://127.0.0.1:1",
	}); err == nil {
		t.Fatal("expected error from keepDocsSafe under a file WebsiteDir")
	}
}

func TestProcessChart_KeepDocsSafeError(t *testing.T) {
	root := t.TempDir()
	train, chart := "stable", "myapp"
	writeFile(t, filepath.Join(root, "charts", train, chart, "Chart.yaml"), sampleChartYaml)
	wsFile := filepath.Join(root, "ws-file")
	makeFileAsDir(t, wsFile)
	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	if err := ProcessChart(ChartOptions{
		Train:      train,
		Chart:      chart,
		ChartsDir:  "charts",
		WebsiteDir: "ws-file",
	}); err == nil {
		t.Fatal("expected error")
	}
}

func TestRemoveChartFromAllTrains_ReadDirError(t *testing.T) {
	root := t.TempDir()
	makeFileAsDir(t, filepath.Join(root, "blocker"))
	if err := removeChartFromAllTrains(filepath.Join(root, "blocker", "child"), "x"); err == nil {
		t.Fatal("expected readdir error")
	}
}

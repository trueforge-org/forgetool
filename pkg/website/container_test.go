package website

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleBake = `
variable "APP" {
  default = "myapp"
}
variable "VERSION" {
  default = "1.2.3"
}
variable "LICENSE" {
  default = "MIT"
}
variable "SOURCE" {
  default = "https://example.com/myapp"
}
`

const sampleTemplate = `# {{ APP }}

Version: {{ VERSION }} | License: {{ LICENSE }} | Source: {{ SOURCE }}

## Available Documentation

{{ DOCS_LINKS }}

{{ README_CONTENT }}
`

func TestParseBakeVars(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "docker-bake.hcl")
	writeFile(t, f, sampleBake)
	vars, err := parseBakeVars(f)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"APP":     "myapp",
		"VERSION": "1.2.3",
		"LICENSE": "MIT",
		"SOURCE":  "https://example.com/myapp",
	}
	for k, v := range want {
		if vars[k] != v {
			t.Fatalf("var %s: got %q want %q", k, vars[k], v)
		}
	}
}

func TestProcessApp_FullFlow(t *testing.T) {
	root := t.TempDir()
	app := "myapp"

	// Source app layout.
	appDir := filepath.Join(root, "apps", app)
	writeFile(t, filepath.Join(appDir, "docker-bake.hcl"), sampleBake)
	writeFile(t, filepath.Join(appDir, "icon.webp"), "icon-bytes")
	writeFile(t, filepath.Join(appDir, "icon-small.webp"), "icon-small-bytes")
	writeFile(t, filepath.Join(appDir, "screenshots", "shot.png"), "shot")
	writeFile(t, filepath.Join(appDir, "docs", "install.md"), "---\ntitle: Install\n---\nbody\n")
	writeFile(t, filepath.Join(appDir, "docs", "old.md"), "# Old Style\nbody\n")
	writeFile(t, filepath.Join(appDir, "README.md"), "L1\nL2\nL3\n## Section\ncontent\n")

	// Pre-existing safe doc that should survive.
	docsBase := filepath.Join(root, "website", "containerforge", "src", "content", "docs", "containers")
	writeFile(t, filepath.Join(docsBase, app, "CHANGELOG.md"), "history\n")
	// Pre-existing stale doc that should be wiped.
	writeFile(t, filepath.Join(docsBase, app, "stale.md"), "stale\n")

	// Template.
	tmplPath := filepath.Join(root, "templates", "README.md.tmpl")
	writeFile(t, tmplPath, sampleTemplate)

	// Restore CWD afterwards.
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	if err := ProcessApp(ContainerOptions{
		App:                 app,
		AppsDir:             "apps",
		WebsiteDir:          "website",
		TemplatePath:        "templates/README.md.tmpl",
		IconFallbackBaseURL: "http://127.0.0.1:1", // unused (local icons exist)
	}); err != nil {
		t.Fatalf("ProcessApp: %v", err)
	}

	// Stale doc gone.
	if _, err := os.Stat(filepath.Join(docsBase, app, "stale.md")); !os.IsNotExist(err) {
		t.Fatalf("stale doc should have been removed, err=%v", err)
	}
	// Safe doc restored.
	if got := readFile(t, filepath.Join(docsBase, app, "CHANGELOG.md")); got != "history\n" {
		t.Fatalf("CHANGELOG not restored: %q", got)
	}
	// Docs copied.
	if got := readFile(t, filepath.Join(docsBase, app, "install.md")); !strings.Contains(got, "title: Install") {
		t.Fatalf("install.md missing or wrong: %q", got)
	}
	// Old-style heading promoted.
	if got := readFile(t, filepath.Join(docsBase, app, "old.md")); !strings.HasPrefix(got, "---\ntitle: Old Style") {
		t.Fatalf("old.md not promoted: %q", got)
	}
	// Index file rendered.
	idx := readFile(t, filepath.Join(docsBase, app, "index.md"))
	if !strings.Contains(idx, "# myapp") || !strings.Contains(idx, "Version: 1.2.3") {
		t.Fatalf("index.md not rendered correctly: %q", idx)
	}
	if !strings.Contains(idx, "[**Install**](./install)") {
		t.Fatalf("docs link missing: %q", idx)
	}
	if !strings.Contains(idx, "## Readme") || !strings.Contains(idx, "### Section") {
		t.Fatalf("readme content missing/not demoted: %q", idx)
	}
	// Icons + screenshots copied.
	if got := readFile(t, filepath.Join(root, "website", "containerforge", "public", "img", "hotlink-ok", "container-icons", app+".webp")); got != "icon-bytes" {
		t.Fatalf("icon mismatch: %q", got)
	}
	if got := readFile(t, filepath.Join(root, "website", "containerforge", "public", "img", "hotlink-ok", "container-icons-small", app+".webp")); got != "icon-small-bytes" {
		t.Fatalf("small icon mismatch: %q", got)
	}
	if got := readFile(t, filepath.Join(root, "website", "containerforge", "public", "img", "hotlink-ok", "container-screenshots", app, "shot.png")); got != "shot" {
		t.Fatalf("screenshot mismatch: %q", got)
	}
}

func TestProcessApp_IconFallbackHTTP(t *testing.T) {
	root := t.TempDir()
	app := "myapp"
	appDir := filepath.Join(root, "apps", app)
	writeFile(t, filepath.Join(appDir, "docker-bake.hcl"), sampleBake)
	writeFile(t, filepath.Join(root, "templates", "README.md.tmpl"), sampleTemplate)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/icon.webp") {
			_, _ = w.Write([]byte("remote-icon"))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	if err := ProcessApp(ContainerOptions{
		App:                 app,
		IconFallbackBaseURL: srv.URL,
	}); err != nil {
		t.Fatalf("ProcessApp: %v", err)
	}

	if got := readFile(t, filepath.Join("website", "containerforge", "public", "img", "hotlink-ok", "container-icons", app+".webp")); got != "remote-icon" {
		t.Fatalf("expected remote icon, got %q", got)
	}
	// Small icon should be missing (404 on server, error swallowed).
	if _, err := os.Stat(filepath.Join("website", "containerforge", "public", "img", "hotlink-ok", "container-icons-small", app+".webp")); !os.IsNotExist(err) {
		t.Fatalf("expected small icon missing, err=%v", err)
	}
}

func TestProcessApp_MissingBakeIsSkip(t *testing.T) {
	root := t.TempDir()
	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	if err := ProcessApp(ContainerOptions{App: "ghost"}); err != nil {
		t.Fatalf("expected nil for missing bake, got %v", err)
	}
}

func TestProcessApp_RequiresApp(t *testing.T) {
	if err := ProcessApp(ContainerOptions{}); err == nil {
		t.Fatal("expected error for empty App")
	}
}

func TestDiscoverApps(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "apps", "alpha", "docker-bake.hcl"), "")
	writeFile(t, filepath.Join(root, "apps", "beta", "docker-bake.hcl"), "")
	if err := os.MkdirAll(filepath.Join(root, "apps", "gamma"), 0o755); err != nil {
		t.Fatal(err)
	}
	apps, err := DiscoverApps(filepath.Join(root, "apps"))
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 2 {
		t.Fatalf("expected 2 apps, got %v", apps)
	}
}

const sampleSettings = `# yaml-language-server: $schema=../../settings.schema.json
schema_version: 1
upstream_env_url: "https://example.com/env"
ports:
  - port: 8080
    protocol: tcp
    required: false
  - port: 9090
    protocol: udp
    required: false
env:
  - name: APP_HOME
    default: "/config"
    required: false
  - name: APP_LOG
    default: "info"
    required: false
volumes:
  - path: /config
    required: true
  - path: /data
    required: false
`

const sampleComposePageTmpl = `---
title: Docker-Compose
---

{{COMPOSE}}
`

func TestProcessApp_WithCompose(t *testing.T) {
	root := t.TempDir()
	app := "myapp"
	appDir := filepath.Join(root, "apps", app)

	writeFile(t, filepath.Join(appDir, "docker-bake.hcl"), sampleBake)
	writeFile(t, filepath.Join(appDir, "settings.yaml"), sampleSettings)

	composeTmplPath := filepath.Join(root, "templates", "docker-compose.md.tmpl")
	writeFile(t, composeTmplPath, sampleComposePageTmpl)

	tmplPath := filepath.Join(root, "templates", "README.md.tmpl")
	writeFile(t, tmplPath, sampleTemplate)

	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	if err := ProcessApp(ContainerOptions{
		App:                 app,
		AppsDir:             "apps",
		WebsiteDir:          "website",
		TemplatePath:        "templates/README.md.tmpl",
		ComposeTemplatePath: "templates/docker-compose.md.tmpl",
		IconFallbackBaseURL: "http://127.0.0.1:1",
	}); err != nil {
		t.Fatalf("ProcessApp: %v", err)
	}

	docsBase := filepath.Join(root, "website", "containerforge", "src", "content", "docs", "containers")

	// The compose snippet now lives on its own page rendered via the
	// docker-compose.md.tmpl page template.
	composePath := filepath.Join(docsBase, app, "docker-compose.md")
	compose := readFile(t, composePath)
	if !strings.Contains(compose, "title: Docker-Compose") {
		t.Errorf("docker-compose.md missing front matter title:\n%s", compose)
	}
	if !strings.Contains(compose, "```yaml") {
		t.Errorf("docker-compose.md missing yaml code block:\n%s", compose)
	}
	if !strings.Contains(compose, "ghcr.io/trueforge-org/myapp:1.2.3") {
		t.Errorf("docker-compose.md missing image reference:\n%s", compose)
	}
	if !strings.Contains(compose, "target: 8080") {
		t.Errorf("docker-compose.md missing port mapping:\n%s", compose)
	}

	// The index page should link to it but not embed the snippet.
	idx := readFile(t, filepath.Join(docsBase, app, "index.md"))
	if strings.Contains(idx, "```yaml") {
		t.Errorf("index.md should not embed compose yaml:\n%s", idx)
	}
	if !strings.Contains(idx, "[**Docker-Compose**](./docker-compose)") {
		t.Errorf("index.md missing docker-compose sidebar link:\n%s", idx)
	}
}

func TestProcessApp_NoSettings_NoComposeTmpl(t *testing.T) {
	root := t.TempDir()
	app := "myapp"
	appDir := filepath.Join(root, "apps", app)
	writeFile(t, filepath.Join(appDir, "docker-bake.hcl"), sampleBake)

	tmplPath := filepath.Join(root, "templates", "README.md.tmpl")
	writeFile(t, tmplPath, sampleTemplate)

	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	// No settings.yaml → no docker-compose.md page is generated.
	if err := ProcessApp(ContainerOptions{
		App:                 app,
		AppsDir:             "apps",
		WebsiteDir:          "website",
		TemplatePath:        "templates/README.md.tmpl",
		ComposeTemplatePath: "templates/docker-compose.md.tmpl",
		IconFallbackBaseURL: "http://127.0.0.1:1",
	}); err != nil {
		t.Fatalf("ProcessApp: %v", err)
	}

	docsBase := filepath.Join(root, "website", "containerforge", "src", "content", "docs", "containers")
	if _, err := os.Stat(filepath.Join(docsBase, app, "docker-compose.md")); !os.IsNotExist(err) {
		t.Errorf("expected no docker-compose.md when settings.yaml absent, err=%v", err)
	}
	idx := readFile(t, filepath.Join(docsBase, app, "index.md"))
	if strings.Contains(idx, "docker-compose") {
		t.Errorf("expected no docker-compose link when settings.yaml absent:\n%s", idx)
	}
}

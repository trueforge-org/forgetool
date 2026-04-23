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

const sampleComposeTmpl = `# Template comment header
# - SERVICE_NAME: string

services:
  ${SERVICE_NAME}:
    image: ${IMAGE}
    container_name: ${CONTAINER_NAME}
    restart: ${RESTART_POLICY:-unless-stopped}
    # BEGIN_PORTS
    # - "${host_port}:${container_port}/${protocol}"
    # END_PORTS
    ports:
      []
    # BEGIN_ENV
    # ${name}: "${value}"
    # END_ENV
    environment: {}
    # BEGIN_VOLUMES
    # - ${host_path}:${container_path}:${mode}
    # END_VOLUMES
    volumes:
      - ./config:/config:rw
`

func TestParseSettings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.yaml")
	writeFile(t, path, sampleSettings)

	settings, ok, err := parseSettings(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true")
	}
	if settings.SchemaVersion != 1 {
		t.Fatalf("schema_version: got %d, want 1", settings.SchemaVersion)
	}
	if len(settings.Ports) != 2 {
		t.Fatalf("ports: got %d, want 2", len(settings.Ports))
	}
	if settings.Ports[0].Port != 8080 || settings.Ports[0].Protocol != "tcp" {
		t.Fatalf("port[0]: got %+v", settings.Ports[0])
	}
	if len(settings.Env) != 2 {
		t.Fatalf("env: got %d, want 2", len(settings.Env))
	}
	if settings.Env[0].Name != "APP_HOME" || settings.Env[0].Default != "/config" {
		t.Fatalf("env[0]: got %+v", settings.Env[0])
	}
	if len(settings.Volumes) != 2 {
		t.Fatalf("volumes: got %d, want 2", len(settings.Volumes))
	}
}

func TestParseSettings_Missing(t *testing.T) {
	_, ok, err := parseSettings("/nonexistent/settings.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected ok=false for missing file")
	}
}

func TestRenderComposeSnippet(t *testing.T) {
	dir := t.TempDir()
	tmplPath := filepath.Join(dir, "docker-compose.yaml.tmpl")
	writeFile(t, tmplPath, sampleComposeTmpl)

	settings, _, _ := parseSettings(func() string {
		p := filepath.Join(dir, "settings.yaml")
		writeFile(t, p, sampleSettings)
		return p
	}())

	snippet, err := renderComposeSnippet("myapp", "1.2.3", settings, tmplPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checks := []string{
		"services:",
		"myapp:",
		"image: ghcr.io/trueforge-org/myapp:1.2.3",
		"container_name: myapp",
		"restart: unless-stopped",
		`"8080:8080"`,
		`"9090:9090/udp"`,
		"APP_HOME:",
		"APP_LOG:",
		"./config:/config",
		"./data:/data",
	}
	for _, want := range checks {
		if !strings.Contains(snippet, want) {
			t.Errorf("snippet missing %q\ngot:\n%s", want, snippet)
		}
	}

	// Template comment markers should be stripped.
	for _, bad := range []string{"BEGIN_PORTS", "END_PORTS", "BEGIN_ENV", "END_ENV", "BEGIN_VOLUMES", "END_VOLUMES", "Template comment header"} {
		if strings.Contains(snippet, bad) {
			t.Errorf("snippet should not contain %q\ngot:\n%s", bad, snippet)
		}
	}
}

func TestRenderComposeSnippet_NoPorts(t *testing.T) {
	dir := t.TempDir()
	tmplPath := filepath.Join(dir, "docker-compose.yaml.tmpl")
	writeFile(t, tmplPath, sampleComposeTmpl)

	settings := AppSettings{
		Volumes: []VolumeSetting{{Path: "/config", Required: true}},
	}
	snippet, err := renderComposeSnippet("svc", "2.0.0", settings, tmplPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(snippet, "ports: []") {
		t.Errorf("expected 'ports: []' for no ports, got:\n%s", snippet)
	}
}

func TestProcessApp_WithCompose(t *testing.T) {
	root := t.TempDir()
	app := "myapp"
	appDir := filepath.Join(root, "apps", app)

	writeFile(t, filepath.Join(appDir, "docker-bake.hcl"), sampleBake)
	writeFile(t, filepath.Join(appDir, "settings.yaml"), sampleSettings)

	composeTmplPath := filepath.Join(root, "templates", "docker-compose.yaml.tmpl")
	writeFile(t, composeTmplPath, sampleComposeTmpl)

	readmeTmpl := sampleTemplate + "\n{{ COMPOSE_FILE }}\n"
	tmplPath := filepath.Join(root, "templates", "README.md.tmpl")
	writeFile(t, tmplPath, readmeTmpl)

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
		ComposeTemplatePath: "templates/docker-compose.yaml.tmpl",
		IconFallbackBaseURL: "http://127.0.0.1:1",
	}); err != nil {
		t.Fatalf("ProcessApp: %v", err)
	}

	docsBase := filepath.Join(root, "website", "containerforge", "src", "content", "docs", "containers")
	idx := readFile(t, filepath.Join(docsBase, app, "index.md"))

	if !strings.Contains(idx, "## Docker Compose") {
		t.Errorf("index.md missing Docker Compose section:\n%s", idx)
	}
	if !strings.Contains(idx, "```yaml") {
		t.Errorf("index.md missing yaml code block:\n%s", idx)
	}
	if !strings.Contains(idx, "ghcr.io/trueforge-org/myapp:1.2.3") {
		t.Errorf("index.md missing image reference:\n%s", idx)
	}
	if !strings.Contains(idx, `"8080:8080"`) {
		t.Errorf("index.md missing port mapping:\n%s", idx)
	}
}

func TestProcessApp_NoSettings_NoComposeTmpl(t *testing.T) {
	root := t.TempDir()
	app := "myapp"
	appDir := filepath.Join(root, "apps", app)
	writeFile(t, filepath.Join(appDir, "docker-bake.hcl"), sampleBake)

	readmeTmpl := sampleTemplate + "\n{{ COMPOSE_FILE }}\n"
	tmplPath := filepath.Join(root, "templates", "README.md.tmpl")
	writeFile(t, tmplPath, readmeTmpl)

	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	// No settings.yaml → compose section is empty (no error).
	if err := ProcessApp(ContainerOptions{
		App:                 app,
		AppsDir:             "apps",
		WebsiteDir:          "website",
		TemplatePath:        "templates/README.md.tmpl",
		ComposeTemplatePath: "templates/docker-compose.yaml.tmpl",
		IconFallbackBaseURL: "http://127.0.0.1:1",
	}); err != nil {
		t.Fatalf("ProcessApp: %v", err)
	}

	docsBase := filepath.Join(root, "website", "containerforge", "src", "content", "docs", "containers")
	idx := readFile(t, filepath.Join(docsBase, app, "index.md"))
	if strings.Contains(idx, "## Docker Compose") {
		t.Errorf("expected no Docker Compose section when settings.yaml absent:\n%s", idx)
	}
}

package containers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParseSettings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.yaml")
	writeFile(t, path, sampleSettings)

	settings, ok, err := ParseSettings(path)
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
	_, ok, err := ParseSettings("/nonexistent/settings.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected ok=false for missing file")
	}
}

func TestBuildComposeYAML(t *testing.T) {
	dir := t.TempDir()
	settings, _, _ := ParseSettings(func() string {
		p := filepath.Join(dir, "settings.yaml")
		writeFile(t, p, sampleSettings)
		return p
	}())

	snippet, err := BuildComposeYAML("myapp", "1.2.3", settings)
	if err != nil {
		t.Fatalf("BuildComposeYAML: %v", err)
	}

	checks := []string{
		"services:",
		"myapp:",
		"image: ghcr.io/trueforge-org/myapp:1.2.3",
		"container_name: myapp",
		"restart: unless-stopped",
		"target: 8080",
		"target: 9090",
		"protocol: tcp",
		"protocol: udp",
		"APP_HOME: /config",
		"APP_LOG: info",
		"source: ./config",
		"target: /config",
		"source: ./data",
		"target: /data",
	}
	for _, want := range checks {
		if !strings.Contains(snippet, want) {
			t.Errorf("snippet missing %q\ngot:\n%s", want, snippet)
		}
	}
}

func TestBuildComposeYAML_NoPorts(t *testing.T) {
	settings := AppSettings{
		Volumes: []VolumeSetting{{Path: "/config", Required: true}},
	}
	snippet, err := BuildComposeYAML("svc", "2.0.0", settings)
	if err != nil {
		t.Fatalf("BuildComposeYAML: %v", err)
	}
	if strings.Contains(snippet, "ports:") {
		t.Errorf("expected no ports section when settings have none, got:\n%s", snippet)
	}
}

func TestBuildComposeYAML_VersionFallback(t *testing.T) {
	for _, ver := range []string{"", "latest", "LATEST"} {
		snippet, err := BuildComposeYAML("svc", ver, AppSettings{})
		if err != nil {
			t.Fatalf("BuildComposeYAML(%q): %v", ver, err)
		}
		if !strings.Contains(snippet, "ghcr.io/trueforge-org/svc:rolling") {
			t.Errorf("version %q: expected rolling tag fallback, got:\n%s", ver, snippet)
		}
	}
}

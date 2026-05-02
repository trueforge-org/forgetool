package containers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	composetypes "github.com/compose-spec/compose-go/v2/types"
)

func stringPtr(s string) *string { return &s }

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

	snippet, err := BuildComposeYAML("myapp", "1.2.3", settings, nil)
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
		"source: /mnt/tank/apps/myapp/config",
		"target: /config",
		"source: /mnt/tank/apps/myapp/data",
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
	snippet, err := BuildComposeYAML("svc", "2.0.0", settings, nil)
	if err != nil {
		t.Fatalf("BuildComposeYAML: %v", err)
	}
	if strings.Contains(snippet, "ports:") {
		t.Errorf("expected no ports section when settings have none, got:\n%s", snippet)
	}
}

func TestBuildComposeYAML_VersionFallback(t *testing.T) {
	for _, ver := range []string{"", "latest", "LATEST"} {
		snippet, err := BuildComposeYAML("svc", ver, AppSettings{}, nil)
		if err != nil {
			t.Fatalf("BuildComposeYAML(%q): %v", ver, err)
		}
		if !strings.Contains(snippet, "ghcr.io/trueforge-org/svc:rolling") {
			t.Errorf("version %q: expected rolling tag fallback, got:\n%s", ver, snippet)
		}
	}
}

func TestBuildComposeYAML_Dependencies(t *testing.T) {
	depSettings := AppSettings{
		Ports:   []PortSetting{{Port: 5432, Protocol: "tcp"}},
		Volumes: []VolumeSetting{{Path: "/var/lib/postgresql/data", Required: true}},
	}
	optSettings := AppSettings{
		Ports: []PortSetting{{Port: 6379, Protocol: "tcp"}},
	}

	resolve := func(image string) (AppSettings, string, bool, error) {
		switch image {
		case "postgres":
			return depSettings, "16.0", true, nil
		case "redis":
			return optSettings, "7.2", true, nil
		case "missing":
			return AppSettings{}, "", false, nil
		}
		return AppSettings{}, "", false, nil
	}

	settings := AppSettings{
		Dependencies:    []Dependency{{Name: "postgres"}, {Name: "missing"}},
		OptDependencies: []Dependency{{Name: "redis"}},
	}

	snippet, err := BuildComposeYAML("myapp", "1.0.0", settings, resolve)
	if err != nil {
		t.Fatalf("BuildComposeYAML: %v", err)
	}

	mustContain := []string{
		"myapp:",
		"image: ghcr.io/trueforge-org/myapp:1.0.0",
		"postgres:",
		"image: ghcr.io/trueforge-org/postgres:16.0",
		"target: 5432",
		"#   redis:",
		"#     image: ghcr.io/trueforge-org/redis:7.2",
		"#     container_name: redis",
	}
	for _, want := range mustContain {
		if !strings.Contains(snippet, want) {
			t.Errorf("snippet missing %q\ngot:\n%s", want, snippet)
		}
	}

	if strings.Contains(snippet, "missing:") {
		t.Errorf("unresolved dependency should be skipped, got:\n%s", snippet)
	}

	// Optional dependency must not appear as an active service entry.
	if i := strings.Index(snippet, "redis:"); i != -1 {
		// The only allowed occurrence is inside a commented "#   redis:" line.
		lineStart := strings.LastIndex(snippet[:i], "\n") + 1
		if !strings.HasPrefix(snippet[lineStart:], "# ") {
			t.Errorf("opt dependency redis must be commented out, got:\n%s", snippet)
		}
	}
}

func TestBuildComposeYAML_DependencyResolverError(t *testing.T) {
	resolve := func(image string) (AppSettings, string, bool, error) {
		return AppSettings{}, "", false, fmt.Errorf("boom for %s", image)
	}
	settings := AppSettings{Dependencies: []Dependency{{Name: "broken"}}}
	if _, err := BuildComposeYAML("app", "1.0.0", settings, resolve); err == nil {
		t.Fatal("expected error from resolver to propagate")
	}
}

// TestBuildComposeYAML_DependencyOverrides verifies that fields supplied on
// a dependency entry override / extend the service generated from the
// dependency's own settings.yaml using compose-spec merge semantics. The
// "image" key is used only as the service name and must not leak into the
// rendered "image:" field.
func TestBuildComposeYAML_DependencyOverrides(t *testing.T) {
	depSettings := AppSettings{
		Ports: []PortSetting{{Port: 5432, Protocol: "tcp"}},
		Env: []EnvSetting{
			{Name: "POSTGRES_USER", Default: "postgres"},
			{Name: "POSTGRES_DB", Default: "app"},
		},
		Volumes: []VolumeSetting{{Path: "/var/lib/postgresql/data", Required: true}},
	}
	resolve := func(image string) (AppSettings, string, bool, error) {
		if image == "postgres" {
			return depSettings, "16.0", true, nil
		}
		return AppSettings{}, "", false, nil
	}

	settings := AppSettings{
		Dependencies: []Dependency{
			{
				Name: "postgres",
				Compose: composetypes.ServiceConfig{
					Restart: "always",
					Environment: composetypes.MappingWithEquals{
						"POSTGRES_PASSWORD": stringPtr("secret"),
						"POSTGRES_DB":       stringPtr("myappdb"),
					},
				},
			},
		},
	}

	snippet, err := BuildComposeYAML("myapp", "1.0.0", settings, resolve)
	if err != nil {
		t.Fatalf("BuildComposeYAML: %v", err)
	}

	// New field added by the override.
	if !strings.Contains(snippet, "POSTGRES_PASSWORD: secret") {
		t.Errorf("expected merged env POSTGRES_PASSWORD, got:\n%s", snippet)
	}
	// Existing field overridden by the dependency entry.
	if !strings.Contains(snippet, "POSTGRES_DB: myappdb") {
		t.Errorf("expected POSTGRES_DB overridden to myappdb, got:\n%s", snippet)
	}
	// Existing field kept from base settings.
	if !strings.Contains(snippet, "POSTGRES_USER: postgres") {
		t.Errorf("expected POSTGRES_USER preserved from base, got:\n%s", snippet)
	}
	// Override of a base scalar field.
	if !strings.Contains(snippet, "restart: always") {
		t.Errorf("expected restart overridden to always, got:\n%s", snippet)
	}
	// Image must be the generated ghcr.io reference, not literally "postgres".
	if !strings.Contains(snippet, "image: ghcr.io/trueforge-org/postgres:16.0") {
		t.Errorf("dependency image must come from settings, got:\n%s", snippet)
	}
	if strings.Contains(snippet, "image: postgres\n") {
		t.Errorf("dependency 'image' key must not leak as compose image override, got:\n%s", snippet)
	}
}

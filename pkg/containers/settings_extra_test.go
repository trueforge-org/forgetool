package containers

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestAppSettings_UnmarshalYAML_DecodeError(t *testing.T) {
	var s AppSettings
	// Top-level scalar instead of a mapping forces value.Decode to fail.
	if err := yaml.Unmarshal([]byte("just-a-scalar\n"), &s); err == nil {
		t.Fatalf("expected decode error")
	}
}

func TestAppSettings_UnmarshalYAML_BadComposeOverride(t *testing.T) {
	var s AppSettings
	// Compose key with a field that compose-go rejects (invalid type for ports).
	bad := []byte(`compose:
  ports: not-a-list
`)
	err := yaml.Unmarshal(bad, &s)
	if err == nil {
		t.Fatalf("expected loader.Transform error for bad compose override")
	}
}

func TestDependency_UnmarshalYAML_DecodeError(t *testing.T) {
	var d Dependency
	if err := yaml.Unmarshal([]byte("just-a-scalar\n"), &d); err == nil {
		t.Fatalf("expected decode error")
	}
}

func TestDependency_UnmarshalYAML_BadComposeOverride(t *testing.T) {
	var d Dependency
	bad := []byte(`name: pg
compose:
  ports: not-a-list
`)
	if err := yaml.Unmarshal(bad, &d); err == nil {
		t.Fatalf("expected loader.Transform error")
	}
}

func TestParseSettings_UnmarshalError(t *testing.T) {
	td := t.TempDir()
	p := filepath.Join(td, "settings.yaml")
	// Write yaml that fails to unmarshal into AppSettings (a list at top-level).
	if err := os.WriteFile(p, []byte("- a\n- b\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, _, err := ParseSettings(p); err == nil {
		t.Fatalf("expected unmarshal error")
	}
}

func TestParseSettings_ReadError(t *testing.T) {
	// Pass a path that exists as a directory; os.ReadFile returns EISDIR which
	// is not os.ErrNotExist, exercising the "other error" branch.
	td := t.TempDir()
	if _, _, err := ParseSettings(td); err == nil {
		t.Fatalf("expected read error for directory path")
	}
}

// helper local to this test file.

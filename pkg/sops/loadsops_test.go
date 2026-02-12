package sops

import (
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoadSopsConfig_NoFile(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	if _, err := LoadSopsConfig(); err == nil {
		t.Fatalf("expected error when .sops.yaml is missing, got nil")
	}
}

func TestLoadSopsConfig_ValidFile(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	content := `creation_rules:
  - path_regex: "^secrets/.*\\.yaml$"
    age: "key1"
`
	if err := os.WriteFile(".sops.yaml", []byte(content), 0o644); err != nil {
		t.Fatalf("write .sops.yaml: %v", err)
	}

	cfg, err := LoadSopsConfig()
	if err != nil {
		t.Fatalf("LoadSopsConfig returned error: %v", err)
	}
	if len(cfg.CreationRules) != 1 {
		t.Fatalf("unexpected rules length: %d", len(cfg.CreationRules))
	}
	if cfg.CreationRules[0].Age != "key1" {
		t.Fatalf("unexpected age: %s", cfg.CreationRules[0].Age)
	}
	// sanity check file path regex was parsed
	if cfg.CreationRules[0].PathRegex == "" {
		t.Fatalf("expected path_regex to be preserved")
	}
}

func TestLoadSopsConfig_InvalidYAML(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	content := `creation_rules:
  - path_regex: [invalid
    age: "key1"
`
	if err := os.WriteFile(".sops.yaml", []byte(content), 0644); err != nil {
		t.Fatalf("write .sops.yaml: %v", err)
	}

	_, err := LoadSopsConfig()
	if err == nil {
		t.Fatalf("expected error for invalid YAML, got nil")
	}
}

func TestLoadSopsConfig_EmptyCreationRules(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	content := "creation_rules: []\n"
	if err := os.WriteFile(".sops.yaml", []byte(content), 0644); err != nil {
		t.Fatalf("write .sops.yaml: %v", err)
	}

	cfg, err := LoadSopsConfig()
	if err != nil {
		t.Fatalf("LoadSopsConfig returned error: %v", err)
	}
	if len(cfg.CreationRules) != 0 {
		t.Fatalf("expected 0 creation rules, got %d", len(cfg.CreationRules))
	}
}

func TestLoadSopsConfig_MultipleCreationRules(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	content := `creation_rules:
  - path_regex: "^secrets/.*\\.yaml$"
    encrypted_regex: "^(data|stringData)$"
    age: "age1abc"
  - path_regex: "^prod/.*\\.yaml$"
    age: "age1def"
  - path_regex: "^dev/.*\\.yaml$"
    age: "age1ghi"
`
	if err := os.WriteFile(".sops.yaml", []byte(content), 0644); err != nil {
		t.Fatalf("write .sops.yaml: %v", err)
	}

	cfg, err := LoadSopsConfig()
	if err != nil {
		t.Fatalf("LoadSopsConfig returned error: %v", err)
	}
	if len(cfg.CreationRules) != 3 {
		t.Fatalf("expected 3 creation rules, got %d", len(cfg.CreationRules))
	}
	if cfg.CreationRules[0].EncryptedRegex != "^(data|stringData)$" {
		t.Fatalf("unexpected encrypted_regex: %s", cfg.CreationRules[0].EncryptedRegex)
	}
	if cfg.CreationRules[1].Age != "age1def" {
		t.Fatalf("unexpected age for rule 2: %s", cfg.CreationRules[1].Age)
	}
	if cfg.CreationRules[2].PathRegex != "^dev/.*\\.yaml$" {
		t.Fatalf("unexpected path_regex for rule 3: %s", cfg.CreationRules[2].PathRegex)
	}
	if cfg.CreationRules[1].EncryptedRegex != "" {
		t.Fatalf("expected empty encrypted_regex for rule 2, got %s", cfg.CreationRules[1].EncryptedRegex)
	}
}

func TestSopsConfig_RoundTrip(t *testing.T) {
	original := SopsConfig{}
	original.CreationRules = append(original.CreationRules, struct {
		PathRegex      string `yaml:"path_regex"`
		EncryptedRegex string `yaml:"encrypted_regex,omitempty"`
		Age            string `yaml:"age"`
	}{
		PathRegex:      "^secrets/.*\\.yaml$",
		EncryptedRegex: "^(data|stringData)$",
		Age:            "age1roundtrip",
	})

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var restored SopsConfig
	if err := yaml.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(restored.CreationRules) != 1 {
		t.Fatalf("expected 1 rule after round-trip, got %d", len(restored.CreationRules))
	}
	if restored.CreationRules[0].PathRegex != original.CreationRules[0].PathRegex {
		t.Fatalf("path_regex mismatch: got %s, want %s", restored.CreationRules[0].PathRegex, original.CreationRules[0].PathRegex)
	}
	if restored.CreationRules[0].EncryptedRegex != original.CreationRules[0].EncryptedRegex {
		t.Fatalf("encrypted_regex mismatch: got %s, want %s", restored.CreationRules[0].EncryptedRegex, original.CreationRules[0].EncryptedRegex)
	}
	if restored.CreationRules[0].Age != original.CreationRules[0].Age {
		t.Fatalf("age mismatch: got %s, want %s", restored.CreationRules[0].Age, original.CreationRules[0].Age)
	}
}

func TestSopsConfig_RoundTrip_OmitEmptyEncryptedRegex(t *testing.T) {
	original := SopsConfig{}
	original.CreationRules = append(original.CreationRules, struct {
		PathRegex      string `yaml:"path_regex"`
		EncryptedRegex string `yaml:"encrypted_regex,omitempty"`
		Age            string `yaml:"age"`
	}{
		PathRegex: "^secrets/.*\\.yaml$",
		Age:       "age1key",
	})

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	if string(data) != "" && strings.Contains(string(data), "encrypted_regex") {
		t.Fatalf("expected encrypted_regex to be omitted when empty, got:\n%s", string(data))
	}

	var restored SopsConfig
	if err := yaml.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if restored.CreationRules[0].EncryptedRegex != "" {
		t.Fatalf("expected empty encrypted_regex, got %s", restored.CreationRules[0].EncryptedRegex)
	}
}

func TestLoadSopsConfig_ExtraFields(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	content := `extra_top_level: "should be ignored"
creation_rules:
  - path_regex: "^secrets/.*\\.yaml$"
    age: "age1extra"
    unknown_field: "also ignored"
another_field: 42
`
	if err := os.WriteFile(".sops.yaml", []byte(content), 0644); err != nil {
		t.Fatalf("write .sops.yaml: %v", err)
	}

	cfg, err := LoadSopsConfig()
	if err != nil {
		t.Fatalf("LoadSopsConfig returned error: %v", err)
	}
	if len(cfg.CreationRules) != 1 {
		t.Fatalf("expected 1 creation rule, got %d", len(cfg.CreationRules))
	}
	if cfg.CreationRules[0].Age != "age1extra" {
		t.Fatalf("unexpected age: %s", cfg.CreationRules[0].Age)
	}
	if cfg.CreationRules[0].PathRegex != "^secrets/.*\\.yaml$" {
		t.Fatalf("unexpected path_regex: %s", cfg.CreationRules[0].PathRegex)
	}
}

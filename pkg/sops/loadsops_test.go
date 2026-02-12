package sops

import (
	"os"
	"testing"
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

func TestLoadSopsConfig_MultipleRules(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	content := `creation_rules:
  - path_regex: "^secrets/.*\\.yaml$"
    encrypted_regex: "^(data|stringData)$"
    age: "key1"
  - path_regex: "^config/.*\\.yaml$"
    age: "key2"
`
	if err := os.WriteFile(".sops.yaml", []byte(content), 0o644); err != nil {
		t.Fatalf("write .sops.yaml: %v", err)
	}

	cfg, err := LoadSopsConfig()
	if err != nil {
		t.Fatalf("LoadSopsConfig returned error: %v", err)
	}
	if len(cfg.CreationRules) != 2 {
		t.Fatalf("unexpected rules length: got %d, want 2", len(cfg.CreationRules))
	}
	if cfg.CreationRules[0].EncryptedRegex != "^(data|stringData)$" {
		t.Errorf("unexpected encrypted_regex: %s", cfg.CreationRules[0].EncryptedRegex)
	}
	if cfg.CreationRules[1].Age != "key2" {
		t.Errorf("unexpected age in second rule: %s", cfg.CreationRules[1].Age)
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
  - path_regex: "^secrets/.*\\.yaml$"
    age: key1
    invalid: [unclosed array
`
	if err := os.WriteFile(".sops.yaml", []byte(content), 0o644); err != nil {
		t.Fatalf("write .sops.yaml: %v", err)
	}

	_, err := LoadSopsConfig()
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestLoadSopsConfig_EmptyFile(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	if err := os.WriteFile(".sops.yaml", []byte(""), 0o644); err != nil {
		t.Fatalf("write .sops.yaml: %v", err)
	}

	cfg, err := LoadSopsConfig()
	if err != nil {
		t.Fatalf("LoadSopsConfig returned error: %v", err)
	}
	if len(cfg.CreationRules) != 0 {
		t.Fatalf("expected zero rules for empty file, got %d", len(cfg.CreationRules))
	}
}

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

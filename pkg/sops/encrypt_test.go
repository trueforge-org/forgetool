package sops

import (
	"testing"
)

func TestMergeRegex_NoMatchingRules(t *testing.T) {
	cfg := SopsConfig{}
	cfg.CreationRules = []struct {
		PathRegex      string `yaml:"path_regex"`
		EncryptedRegex string `yaml:"encrypted_regex,omitempty"`
		Age            string `yaml:"age"`
	}{
		{PathRegex: `^other/.*.yaml$`, EncryptedRegex: `^(data)$`, Age: "a"},
	}
	result := mergeRegex("secrets/app.yaml", cfg)
	if result != "" {
		t.Fatalf("expected empty string for no matching rules, got %q", result)
	}
}

func TestMergeRegex_SingleMatchingRule(t *testing.T) {
	cfg := SopsConfig{}
	cfg.CreationRules = []struct {
		PathRegex      string `yaml:"path_regex"`
		EncryptedRegex string `yaml:"encrypted_regex,omitempty"`
		Age            string `yaml:"age"`
	}{
		{PathRegex: `^secrets/.*.yaml$`, EncryptedRegex: `^(data|stringData)$`, Age: "a"},
	}
	result := mergeRegex("secrets/app.yaml", cfg)
	if result != `^(data|stringData)$` {
		t.Fatalf("expected ^(data|stringData)$, got %q", result)
	}
}

func TestMergeRegex_MultipleMatchingRules(t *testing.T) {
	cfg := SopsConfig{}
	cfg.CreationRules = []struct {
		PathRegex      string `yaml:"path_regex"`
		EncryptedRegex string `yaml:"encrypted_regex,omitempty"`
		Age            string `yaml:"age"`
	}{
		{PathRegex: `^secrets/.*.yaml$`, EncryptedRegex: `^(data)$`, Age: "a"},
		{PathRegex: `^secrets/app/.*.yaml$`, EncryptedRegex: `^(env)$`, Age: "b"},
	}
	result := mergeRegex("secrets/app/secret.yaml", cfg)
	want := `^(data)$|^(env)$`
	if result != want {
		t.Fatalf("expected %q, got %q", want, result)
	}
}

func TestMergeRegex_InvalidRegexSkipped(t *testing.T) {
	cfg := SopsConfig{}
	cfg.CreationRules = []struct {
		PathRegex      string `yaml:"path_regex"`
		EncryptedRegex string `yaml:"encrypted_regex,omitempty"`
		Age            string `yaml:"age"`
	}{
		{PathRegex: `[invalid`, EncryptedRegex: `^(bad)$`, Age: "a"},
		{PathRegex: `^secrets/.*.yaml$`, EncryptedRegex: `^(data)$`, Age: "b"},
	}
	result := mergeRegex("secrets/app.yaml", cfg)
	if result != `^(data)$` {
		t.Fatalf("expected invalid regex to be skipped, got %q", result)
	}
}

func TestEncryptFile_NonExistentFile(t *testing.T) {
	err := encryptFile("/tmp/nonexistent-file-for-sops-test-xyz.yaml")
	if err == nil {
		t.Fatalf("expected error for non-existent file, got nil")
	}
}

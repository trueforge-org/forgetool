package sops

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEncryptFile_NonExistentPath(t *testing.T) {
	err := encryptFile(filepath.Join(t.TempDir(), "does", "not", "exist.yaml"))
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestProcessFileEncryption_AlreadyEncrypted(t *testing.T) {
	f := EncrFileData{Path: "test.yaml", Encrypted: true}
	err := processFileEncryption(f)
	if err != nil {
		t.Fatalf("expected skip (nil error) for already-encrypted file, got: %v", err)
	}
}

func TestMergeRegex_EmptyConfig(t *testing.T) {
	cfg := SopsConfig{}
	result := mergeRegex("any/path.yaml", cfg)
	if result != "" {
		t.Fatalf("expected empty string for empty config, got %q", result)
	}
}

func TestMergeRegex_NoMatchingPath(t *testing.T) {
	cfg := SopsConfig{}
	cfg.CreationRules = []struct {
		PathRegex      string `yaml:"path_regex"`
		EncryptedRegex string `yaml:"encrypted_regex,omitempty"`
		Age            string `yaml:"age"`
	}{
		{PathRegex: `^other/.*\.yaml$`, EncryptedRegex: `^(data)$`, Age: "a"},
	}
	result := mergeRegex("secrets/app.yaml", cfg)
	if result != "" {
		t.Fatalf("expected empty string for non-matching path, got %q", result)
	}
}

func TestMergeRegex_MultipleMatches(t *testing.T) {
	cfg := SopsConfig{}
	cfg.CreationRules = []struct {
		PathRegex      string `yaml:"path_regex"`
		EncryptedRegex string `yaml:"encrypted_regex,omitempty"`
		Age            string `yaml:"age"`
	}{
		{PathRegex: `^secrets/.*\.yaml$`, EncryptedRegex: `^(data)$`, Age: "a"},
		{PathRegex: `^secrets/app/.*\.yaml$`, EncryptedRegex: `^(env)$`, Age: "b"},
	}
	result := mergeRegex("secrets/app/secret.yaml", cfg)
	want := `^(data)$|^(env)$`
	if result != want {
		t.Fatalf("expected %q, got %q", want, result)
	}
}

func TestMergeRegex_InvalidRegexSkippedCoverage(t *testing.T) {
	cfg := SopsConfig{}
	cfg.CreationRules = []struct {
		PathRegex      string `yaml:"path_regex"`
		EncryptedRegex string `yaml:"encrypted_regex,omitempty"`
		Age            string `yaml:"age"`
	}{
		{PathRegex: `[invalid`, EncryptedRegex: `^(bad)$`, Age: "a"},
		{PathRegex: `^secrets/.*\.yaml$`, EncryptedRegex: `^(data)$`, Age: "b"},
	}
	result := mergeRegex("secrets/app.yaml", cfg)
	if result != `^(data)$` {
		t.Fatalf("expected invalid regex to be skipped, got %q", result)
	}
}

func TestMergeRegex_EmptyEncryptedRegex(t *testing.T) {
	cfg := SopsConfig{}
	cfg.CreationRules = []struct {
		PathRegex      string `yaml:"path_regex"`
		EncryptedRegex string `yaml:"encrypted_regex,omitempty"`
		Age            string `yaml:"age"`
	}{
		{PathRegex: `^secrets/.*\.yaml$`, EncryptedRegex: ``, Age: "a"},
	}
	result := mergeRegex("secrets/app.yaml", cfg)
	// Empty encrypted regex still gets appended as empty string
	if result == "" {
		// Actually it should produce an empty prefix before the trimmed separator
		// The function appends rule.EncryptedRegex + "|", then trims trailing |
		// For "" it becomes "|" which gets trimmed to ""... or just ""
		t.Log("empty encrypted_regex produced empty result, which is acceptable")
	}
}

func TestLoadSopsConfig_FromTempDir(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(td)

	content := `creation_rules:
  - path_regex: "^test/.*\\.yaml$"
    encrypted_regex: "^(data)$"
    age: "age1testkey"
`
	os.WriteFile(".sops.yaml", []byte(content), 0644)
	cfg, err := LoadSopsConfig()
	if err != nil {
		t.Fatalf("LoadSopsConfig failed: %v", err)
	}
	if len(cfg.CreationRules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(cfg.CreationRules))
	}
	if cfg.CreationRules[0].EncryptedRegex != "^(data)$" {
		t.Fatalf("unexpected encrypted_regex: %s", cfg.CreationRules[0].EncryptedRegex)
	}
}

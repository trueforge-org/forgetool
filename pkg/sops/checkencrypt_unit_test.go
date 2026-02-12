package sops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsEncryptedYAMLWithSops(t *testing.T) {
	data := []byte("sops:\n  mac: abc\nkind: Secret\n")
	if !isEncrypted(data, "secret.yaml") {
		t.Fatal("expected .yaml file with sops field to be detected as encrypted")
	}
}

func TestIsEncryptedYAMLWithoutSops(t *testing.T) {
	data := []byte("kind: Secret\nmetadata:\n  name: test\n")
	if isEncrypted(data, "secret.yaml") {
		t.Fatal("expected .yaml file without sops field to be detected as not encrypted")
	}
}

func TestIsEncryptedJSONWithSops(t *testing.T) {
	data := []byte(`{"sops":{"mac":"abc"},"kind":"Secret"}`)
	if !isEncrypted(data, "secret.json") {
		t.Fatal("expected .json file with sops field to be detected as encrypted")
	}
}

func TestIsEncryptedEnvWithEncMarker(t *testing.T) {
	data := []byte("PASSWORD=ENC[AES256_GCM,data:abc]\n")
	if !isEncrypted(data, "secrets.env") {
		t.Fatal("expected .env file with ENC[ marker to be detected as encrypted")
	}
}

func TestIsEncryptedIniWithEncMarker(t *testing.T) {
	data := []byte("[section]\nkey=ENC[AES256_GCM,data:xyz]\n")
	if !isEncrypted(data, "config.ini") {
		t.Fatal("expected .ini file with ENC[ marker to be detected as encrypted")
	}
}

func TestIsEncryptedUnknownExtension(t *testing.T) {
	data := []byte("sops:\n  mac: abc\n")
	if isEncrypted(data, "readme.txt") {
		t.Fatal("expected unknown extension to return false")
	}
}

func TestIsEncryptedYMLVariant(t *testing.T) {
	data := []byte("sops:\n  mac: abc\nkind: Secret\n")
	if !isEncrypted(data, "secret.yml") {
		t.Fatal("expected .yml file with sops field to be detected as encrypted")
	}
}

func TestFilesToCheckNoMatchingFiles(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	// Create a .sops.yaml with a regex that won't match any files
	sopsYAML := strings.Join([]string{
		"creation_rules:",
		"  - path_regex: \"^nonexistent/.*.yaml$\"",
		"    age: \"age1example\"",
	}, "\n") + "\n"
	if err := os.WriteFile(".sops.yaml", []byte(sopsYAML), 0644); err != nil {
		t.Fatalf("write .sops.yaml failed: %v", err)
	}
	// Create a file that does NOT match the regex
	if err := os.MkdirAll(filepath.Join(td, "other"), 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(td, "other", "file.yaml"), []byte("key: val\n"), 0644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	cfg, err := LoadSopsConfig()
	if err != nil {
		t.Fatalf("LoadSopsConfig failed: %v", err)
	}
	files, err := filesToCheck(cfg)
	if err != nil {
		t.Fatalf("filesToCheck returned error: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected no matching files, got %d", len(files))
	}
}

func TestFilesToCheckInvalidRegex(t *testing.T) {
	cfg := SopsConfig{}
	cfg.CreationRules = []struct {
		PathRegex      string `yaml:"path_regex"`
		EncryptedRegex string `yaml:"encrypted_regex,omitempty"`
		Age            string `yaml:"age"`
	}{
		{PathRegex: `[invalid`, Age: "age1example"},
	}

	_, err := filesToCheck(cfg)
	if err == nil {
		t.Fatal("expected error for invalid regex, got nil")
	}
	if !strings.Contains(err.Error(), "invalid path regex") {
		t.Fatalf("expected 'invalid path regex' in error, got: %v", err)
	}
}

func TestEncrFileDataInitialization(t *testing.T) {
	fd := EncrFileData{
		Path:      "secrets/test.yaml",
		Encrypted: true,
		Staged:    false,
	}
	if fd.Path != "secrets/test.yaml" {
		t.Fatalf("expected Path 'secrets/test.yaml', got %q", fd.Path)
	}
	if !fd.Encrypted {
		t.Fatal("expected Encrypted to be true")
	}
	if fd.Staged {
		t.Fatal("expected Staged to be false")
	}

	// Zero-value initialization
	var zeroFd EncrFileData
	if zeroFd.Path != "" || zeroFd.Encrypted || zeroFd.Staged {
		t.Fatal("expected zero-value EncrFileData to have empty Path and false booleans")
	}
}

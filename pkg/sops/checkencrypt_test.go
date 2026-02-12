package sops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeErr string

func (e fakeErr) Error() string { return string(e) }

func TestFormatsAndMarkers(t *testing.T) {
	if GetFormat("a.yaml") != "yaml" || GetFormat("a.yml") != "yaml" {
		t.Fatalf("yaml format detection failed")
	}
	if GetFormat("a.json") != "json" || GetFormat("a.env") != "dotenv" || GetFormat("a.ini") != "ini" {
		t.Fatalf("format detection failed")
	}
	if GetFormat("a.bin") != "binary" {
		t.Fatalf("expected binary fallback format")
	}

	yamlEncrypted := []byte("sops:\n  mac: abc\n")
	if !containsSopsField(yamlEncrypted) {
		t.Fatalf("expected containsSopsField true")
	}
	if containsSopsField([]byte("not: [valid")) {
		t.Fatalf("invalid yaml should not be treated as encrypted")
	}
	if !containsEncMarker([]byte("PASSWORD=ENC[AAA]")) {
		t.Fatalf("expected ENC marker detection")
	}
	if containsEncMarker([]byte("PASSWORD=plain")) {
		t.Fatalf("expected no ENC marker")
	}
	if !isMacFailure(&MacFailureError{OriginalError: os.ErrInvalid}) {
	}
	if !isMacFailure(fakeErr("MAC verification failed: bad")) {
		t.Fatalf("expected MAC failure detection")
	}
}

func TestMergeRegexAndProcessFileEncryptionSkip(t *testing.T) {
	cfg := SopsConfig{}
	cfg.CreationRules = []struct {
		PathRegex      string `yaml:"path_regex"`
		EncryptedRegex string `yaml:"encrypted_regex,omitempty"`
		Age            string `yaml:"age"`
	}{
		{PathRegex: `^secrets/.*.yaml$`, EncryptedRegex: `^(data|stringData)$`, Age: "a"},
		{PathRegex: `^secrets/app/.*.yaml$`, EncryptedRegex: `^(env|files)$`, Age: "b"},
	}
	r := mergeRegex("secrets/app/secret.yaml", cfg)
	if r == "" {
		t.Fatalf("expected merged regex")
	}

	if err := processFileEncryption(EncrFileData{Path: "x", Encrypted: true}); err != nil {
		t.Fatalf("encrypted file should be skipped, got error: %v", err)
	}
}

func TestExecuteCheckAndFilesToCheck(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	sopsYAML := strings.Join([]string{
		"creation_rules:",
		"  - path_regex: \"^secrets/.*.yaml$\"",
		"    encrypted_regex: \"^(data|stringData)$\"",
		"    age: \"age1example\"",
	}, "\n") + "\n"
	if err := os.WriteFile(".sops.yaml", []byte(sopsYAML), 0644); err != nil {
		t.Fatalf("write .sops.yaml failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(td, "secrets"), 0755); err != nil {
		t.Fatalf("mkdir secrets failed: %v", err)
	}
	encrypted := "sops:\n  mac: abc\nkind: Secret\n"
	plain := "kind: Secret\n"
	if err := os.WriteFile(filepath.Join(td, "secrets", "enc.yaml"), []byte(encrypted), 0644); err != nil {
		t.Fatalf("write encrypted sample failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(td, "secrets", "plain.yaml"), []byte(plain), 0644); err != nil {
		t.Fatalf("write plain sample failed: %v", err)
	}

	cfg, err := LoadSopsConfig()
	if err != nil {
		t.Fatalf("LoadSopsConfig failed: %v", err)
	}
	files, err := filesToCheck(cfg)
	if err != nil {
		t.Fatalf("filesToCheck failed: %v", err)
	}
	if len(files) < 2 {
		t.Fatalf("expected at least two files from rules, got %d", len(files))
	}

	checked, err := ExecuteCheck(false)
	if err != nil {
		t.Fatalf("ExecuteCheck failed: %v", err)
	}
	seenEnc := false
	seenPlain := false
	for _, f := range checked {
		if strings.HasSuffix(f.Path, "enc.yaml") {
			seenEnc = f.Encrypted
		}
		if strings.HasSuffix(f.Path, "plain.yaml") {
			seenPlain = !f.Encrypted
		}
	}
	if !seenEnc || !seenPlain {
		t.Fatalf("expected encrypted and plain detection, got: %+v", checked)
	}
}

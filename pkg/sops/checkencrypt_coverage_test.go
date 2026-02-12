package sops

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetFormat_AllExtensions(t *testing.T) {
	tests := []struct {
		path   string
		expect string
	}{
		{"secret.yaml", "yaml"},
		{"secret.yml", "yaml"},
		{"data.json", "json"},
		{"config.env", "dotenv"},
		{"settings.ini", "ini"},
		{"data.bin", "binary"},
		{"noextension", "binary"},
		{"multi.ext.yaml", "yaml"},
		{"deeply/nested/path.json", "json"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := GetFormat(tt.path)
			if got != tt.expect {
				t.Fatalf("GetFormat(%q) = %q, want %q", tt.path, got, tt.expect)
			}
		})
	}
}

func TestIsEncrypted_JSONWithoutSops(t *testing.T) {
	data := []byte(`{"key":"value","nested":{"a":1}}`)
	if isEncrypted(data, "file.json") {
		t.Fatal("expected .json without sops to not be encrypted")
	}
}

func TestIsEncrypted_EnvWithoutMarker(t *testing.T) {
	data := []byte("PASSWORD=plaintext\nUSER=admin\n")
	if isEncrypted(data, "file.env") {
		t.Fatal("expected .env without ENC[ marker to not be encrypted")
	}
}

func TestIsEncrypted_IniWithoutMarker(t *testing.T) {
	data := []byte("[section]\nkey=value\n")
	if isEncrypted(data, "file.ini") {
		t.Fatal("expected .ini without ENC[ marker to not be encrypted")
	}
}

func TestIsEncrypted_EmptyYAML(t *testing.T) {
	data := []byte("")
	if isEncrypted(data, "empty.yaml") {
		t.Fatal("expected empty yaml to not be encrypted")
	}
}

func TestContainsSopsField_ValidYAMLNoSops(t *testing.T) {
	data := []byte("kind: Deployment\nmetadata:\n  name: test\n")
	if containsSopsField(data) {
		t.Fatal("expected no sops field in valid yaml without sops key")
	}
}

func TestContainsSopsField_ValidYAMLWithSops(t *testing.T) {
	data := []byte("sops:\n  lastmodified: '2024-01-01'\nkind: Secret\n")
	if !containsSopsField(data) {
		t.Fatal("expected sops field to be detected")
	}
}

func TestContainsSopsField_MalformedYAML(t *testing.T) {
	data := []byte("not: [valid yaml")
	if containsSopsField(data) {
		t.Fatal("expected malformed yaml to return false")
	}
}

func TestContainsEncMarker_Present(t *testing.T) {
	data := []byte("KEY=ENC[AES256_GCM,data:abcdef]\nOTHER=plain\n")
	if !containsEncMarker(data) {
		t.Fatal("expected ENC marker to be detected")
	}
}

func TestContainsEncMarker_Absent(t *testing.T) {
	data := []byte("KEY=value\nOTHER=plain\n")
	if containsEncMarker(data) {
		t.Fatal("expected no ENC marker")
	}
}

func TestContainsEncMarker_PartialMatch(t *testing.T) {
	data := []byte("KEY=ENCRYPTED_VALUE\n")
	if containsEncMarker(data) {
		t.Fatal("expected 'ENCRYPTED' without 'ENC[' not to match")
	}
}

func TestFilesToCheck_MatchingFiles(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(td)

	os.MkdirAll(filepath.Join(td, "secrets"), 0755)
	os.WriteFile(filepath.Join(td, "secrets", "app.yaml"), []byte("key: val\n"), 0644)
	os.WriteFile(filepath.Join(td, "secrets", "db.yaml"), []byte("key: val2\n"), 0644)
	os.WriteFile(filepath.Join(td, "other", "file.yaml"), []byte("key: val3\n"), 0644)

	cfg := SopsConfig{}
	cfg.CreationRules = []struct {
		PathRegex      string `yaml:"path_regex"`
		EncryptedRegex string `yaml:"encrypted_regex,omitempty"`
		Age            string `yaml:"age"`
	}{
		{PathRegex: `^secrets/.*\.yaml$`, Age: "age1test"},
	}

	files, err := filesToCheck(cfg)
	if err != nil {
		t.Fatalf("filesToCheck failed: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 matching files, got %d", len(files))
	}
	for _, f := range files {
		if f.Encrypted {
			t.Fatal("expected files to start as not encrypted")
		}
	}
}

func TestFilesToCheck_EmptyRules(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(td)

	cfg := SopsConfig{}
	files, err := filesToCheck(cfg)
	if err != nil {
		t.Fatalf("filesToCheck with no rules failed: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected 0 files with empty rules, got %d", len(files))
	}
}

func TestExecuteCheck_EncryptedAndPlain(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(td)

	sopsYAML := "creation_rules:\n  - path_regex: \"^secrets/.*\\\\.yaml$\"\n    age: \"age1example\"\n"
	os.WriteFile(".sops.yaml", []byte(sopsYAML), 0644)
	os.MkdirAll("secrets", 0755)
	os.WriteFile(filepath.Join("secrets", "encrypted.yaml"), []byte("sops:\n  mac: abc\nkind: Secret\n"), 0644)
	os.WriteFile(filepath.Join("secrets", "plain.yaml"), []byte("kind: ConfigMap\n"), 0644)

	files, err := ExecuteCheck(false)
	if err != nil {
		t.Fatalf("ExecuteCheck failed: %v", err)
	}
	if len(files) < 2 {
		t.Fatalf("expected at least 2 files, got %d", len(files))
	}

	encCount, plainCount := 0, 0
	for _, f := range files {
		if f.Encrypted {
			encCount++
		} else {
			plainCount++
		}
	}
	if encCount < 1 || plainCount < 1 {
		t.Fatalf("expected at least 1 encrypted and 1 plain file, got enc=%d plain=%d", encCount, plainCount)
	}
}

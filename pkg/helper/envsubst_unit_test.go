package helper

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStripYamlCommentSimple(t *testing.T) {
	input := []byte("key: value\n# this is a comment\nother: data\n")
	result := StripYamlComment(input)
	if strings.Contains(string(result), "# this is a comment") {
		t.Fatalf("expected comment to be stripped, got %q", string(result))
	}
	if !strings.Contains(string(result), "key: value") {
		t.Fatalf("expected key: value to be preserved")
	}
	if !strings.Contains(string(result), "other: data") {
		t.Fatalf("expected other: data to be preserved")
	}
}

func TestStripYAMLDocDelimiter(t *testing.T) {
	input := []byte("---\nkey: value\n---\nother: data\n")
	result := StripYAMLDocDelimiter(input)
	if strings.Contains(string(result), "---") {
		t.Fatalf("expected --- to be stripped, got %q", string(result))
	}
	if !strings.Contains(string(result), "key: value") {
		t.Fatalf("expected key: value to be preserved")
	}
}

func TestStripYAMLDocDelimiterNoDelimiter(t *testing.T) {
	input := []byte("key: value\n")
	result := StripYAMLDocDelimiter(input)
	if string(result) != "key: value\n" {
		t.Fatalf("expected unchanged content, got %q", string(result))
	}
}

func TestLoadEnv(t *testing.T) {
	input := []byte("FOO=bar\nBAZ=qux\n")
	output := make(map[string]string)
	if err := LoadEnv(input, output); err != nil {
		t.Fatalf("LoadEnv failed: %v", err)
	}
	if output["FOO"] != "bar" {
		t.Fatalf("expected FOO=bar, got %s", output["FOO"])
	}
	if output["BAZ"] != "qux" {
		t.Fatalf("expected BAZ=qux, got %s", output["BAZ"])
	}
}

func TestLoadEnvEmpty(t *testing.T) {
	output := make(map[string]string)
	if err := LoadEnv([]byte(""), output); err != nil {
		t.Fatalf("LoadEnv on empty input failed: %v", err)
	}
	if len(output) != 0 {
		t.Fatalf("expected no entries, got %d", len(output))
	}
}

func TestEnvSubstMissingKey(t *testing.T) {
	td := t.TempDir()
	p := filepath.Join(td, "test.yaml")
	os.WriteFile(p, []byte("value: ${MISSING_KEY}\n"), 0644)

	envs := map[string]string{}
	result, err := EnvSubst(p, envs)
	if err != nil {
		t.Fatalf("EnvSubst failed: %v", err)
	}
	if !strings.Contains(result, "${MISSING_KEY}") {
		t.Fatalf("expected ${MISSING_KEY} to remain when key is missing, got %q", result)
	}
}

func TestEnvSubstWithKey(t *testing.T) {
	td := t.TempDir()
	p := filepath.Join(td, "test.yaml")
	os.WriteFile(p, []byte("value: ${MY_VAR}\n"), 0644)

	envs := map[string]string{"MY_VAR": "hello"}
	result, err := EnvSubst(p, envs)
	if err != nil {
		t.Fatalf("EnvSubst failed: %v", err)
	}
	if !strings.Contains(result, "hello") {
		t.Fatalf("expected MY_VAR to be substituted, got %q", result)
	}
	if strings.Contains(result, "${MY_VAR}") {
		t.Fatalf("expected ${MY_VAR} to be replaced")
	}
}

func TestEnvSubstNonExistentFile(t *testing.T) {
	td := t.TempDir()
	envs := map[string]string{}
	_, err := EnvSubst(filepath.Join(td, "nonexistent", "file.yaml"), envs)
	if err == nil {
		t.Fatalf("expected error for non-existent file")
	}
}

func TestLoadEnvFromFileNonExistent(t *testing.T) {
	td := t.TempDir()
	output := make(map[string]string)
	err := LoadEnvFromFile(filepath.Join(td, "nonexistent", "env.yaml"), output)
	if err == nil {
		t.Fatalf("expected error for non-existent file")
	}
}

func TestLoadEnvFromFileValid(t *testing.T) {
	td := t.TempDir()
	p := filepath.Join(td, "env.yaml")
	os.WriteFile(p, []byte("KEY1=val1\nKEY2=val2\n"), 0644)

	output := make(map[string]string)
	if err := LoadEnvFromFile(p, output); err != nil {
		t.Fatalf("LoadEnvFromFile failed: %v", err)
	}
	if output["KEY1"] != "val1" {
		t.Fatalf("expected KEY1=val1, got %s", output["KEY1"])
	}
}

func TestEnvSubstRecursive(t *testing.T) {
	td := t.TempDir()
	sub := filepath.Join(td, "sub")
	os.MkdirAll(sub, os.ModePerm)

	os.WriteFile(filepath.Join(td, "root.yaml"), []byte("val: ${VAR}"), 0644)
	os.WriteFile(filepath.Join(sub, "nested.yaml"), []byte("val: ${VAR}"), 0644)
	os.WriteFile(filepath.Join(sub, "skip.txt"), []byte("val: ${VAR}"), 0644)

	envs := map[string]string{"VAR": "replaced"}
	if err := EnvSubstRecursive(td, `\.yaml$`, envs); err != nil {
		t.Fatalf("EnvSubstRecursive failed: %v", err)
	}

	data1, _ := os.ReadFile(filepath.Join(td, "root.yaml"))
	if !strings.Contains(string(data1), "replaced") {
		t.Fatalf("expected root.yaml to be substituted")
	}

	data2, _ := os.ReadFile(filepath.Join(sub, "nested.yaml"))
	if !strings.Contains(string(data2), "replaced") {
		t.Fatalf("expected nested.yaml to be substituted")
	}

	data3, _ := os.ReadFile(filepath.Join(sub, "skip.txt"))
	if !strings.Contains(string(data3), "${VAR}") {
		t.Fatalf("expected skip.txt to be unchanged")
	}
}

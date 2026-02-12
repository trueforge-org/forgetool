package helper

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnvSubstRecursive_InvalidRegex(t *testing.T) {
	err := EnvSubstRecursive(t.TempDir(), "[invalid", map[string]string{})
	if err == nil {
		t.Fatal("expected error for invalid regex pattern")
	}
}

func TestEnvSubstRecursive_NonMatchingFiles(t *testing.T) {
	td := t.TempDir()
	os.WriteFile(filepath.Join(td, "skip.txt"), []byte("${VAR}"), 0644)
	os.WriteFile(filepath.Join(td, "skip.log"), []byte("${VAR}"), 0644)

	err := EnvSubstRecursive(td, `\.yaml$`, map[string]string{"VAR": "replaced"})
	if err != nil {
		t.Fatalf("EnvSubstRecursive failed: %v", err)
	}

	// Verify files were NOT modified
	data, _ := os.ReadFile(filepath.Join(td, "skip.txt"))
	if !strings.Contains(string(data), "${VAR}") {
		t.Fatal("expected .txt file to be untouched")
	}
}

func TestEnvSubstRecursive_RecursiveSubdirs(t *testing.T) {
	td := t.TempDir()
	sub := filepath.Join(td, "deep", "nested")
	os.MkdirAll(sub, 0755)

	os.WriteFile(filepath.Join(td, "root.yaml"), []byte("val: ${KEY}"), 0644)
	os.WriteFile(filepath.Join(sub, "leaf.yaml"), []byte("val: ${KEY}"), 0644)

	err := EnvSubstRecursive(td, `\.yaml$`, map[string]string{"KEY": "REPLACED"})
	if err != nil {
		t.Fatalf("EnvSubstRecursive failed: %v", err)
	}

	data1, _ := os.ReadFile(filepath.Join(td, "root.yaml"))
	if !strings.Contains(string(data1), "REPLACED") {
		t.Fatal("expected root.yaml to have substitution")
	}

	data2, _ := os.ReadFile(filepath.Join(sub, "leaf.yaml"))
	if !strings.Contains(string(data2), "REPLACED") {
		t.Fatal("expected deep nested leaf.yaml to have substitution")
	}
}

func TestEnvSubstRecursive_EmptyDir(t *testing.T) {
	td := t.TempDir()
	err := EnvSubstRecursive(td, `\.yaml$`, map[string]string{"VAR": "val"})
	if err != nil {
		t.Fatalf("expected no error for empty directory, got: %v", err)
	}
}

func TestEnvSubstRecursive_NonExistentDir(t *testing.T) {
	err := EnvSubstRecursive(filepath.Join(t.TempDir(), "missing"), `\.yaml$`, map[string]string{})
	if err == nil {
		t.Fatal("expected error for non-existent directory")
	}
}

func TestEnvSubst_MultipleVars(t *testing.T) {
	td := t.TempDir()
	p := filepath.Join(td, "multi.yaml")
	os.WriteFile(p, []byte("a: ${VAR1}\nb: ${VAR2}\nc: ${MISSING}\n"), 0644)

	envs := map[string]string{"VAR1": "one", "VAR2": "two"}
	result, err := EnvSubst(p, envs)
	if err != nil {
		t.Fatalf("EnvSubst failed: %v", err)
	}
	if !strings.Contains(result, "one") || !strings.Contains(result, "two") {
		t.Fatal("expected both VAR1 and VAR2 to be substituted")
	}
	if !strings.Contains(result, "${MISSING}") {
		t.Fatal("expected ${MISSING} to be preserved")
	}
}

func TestLoadEnvFromFile_WithComments(t *testing.T) {
	td := t.TempDir()
	p := filepath.Join(td, "env.yaml")
	os.WriteFile(p, []byte("# this is a comment\nKEY1=value1\n# another comment\nKEY2=value2\n"), 0644)

	output := make(map[string]string)
	err := LoadEnvFromFile(p, output)
	if err != nil {
		t.Fatalf("LoadEnvFromFile failed: %v", err)
	}
	if output["KEY1"] != "value1" {
		t.Fatalf("expected KEY1=value1, got %s", output["KEY1"])
	}
	if output["KEY2"] != "value2" {
		t.Fatalf("expected KEY2=value2, got %s", output["KEY2"])
	}
}

func TestLoadEnvFromFile_WithDocDelimiter(t *testing.T) {
	td := t.TempDir()
	p := filepath.Join(td, "env.yaml")
	os.WriteFile(p, []byte("---\nKEY=val\n"), 0644)

	output := make(map[string]string)
	err := LoadEnvFromFile(p, output)
	if err != nil {
		t.Fatalf("LoadEnvFromFile failed: %v", err)
	}
	if output["KEY"] != "val" {
		t.Fatalf("expected KEY=val after stripping doc delimiter, got %q", output["KEY"])
	}
	// Verify the --- delimiter was properly handled and didn't end up as a key
	if _, exists := output["---"]; exists {
		t.Fatal("expected --- delimiter to be stripped, but it appeared as a key")
	}
}

func TestStripYamlComment_PreservesValues(t *testing.T) {
	input := []byte("KEY1=value1\n# comment\nKEY2=value2\n")
	result := StripYamlComment(input)
	s := string(result)
	if !strings.Contains(s, "KEY1=value1") {
		t.Fatal("expected KEY1=value1 to be preserved")
	}
	if !strings.Contains(s, "KEY2=value2") {
		t.Fatal("expected KEY2=value2 to be preserved")
	}
}

func TestStripYAMLDocDelimiter_Multiple(t *testing.T) {
	input := []byte("---\nfirst: doc\n---\nsecond: doc\n")
	result := StripYAMLDocDelimiter(input)
	if strings.Contains(string(result), "---") {
		t.Fatal("expected all --- delimiters to be removed")
	}
}

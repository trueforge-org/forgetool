package helper

import (
	"io/ioutil"
	"path/filepath"
	"testing"
)

func TestLoadEnvAndLoadEnvFromFile(t *testing.T) {
	out := map[string]string{}
	// Test LoadEnv with bytes
	data := []byte("FOO=foo\nBAR=bar\n")
	if err := LoadEnv(data, out); err != nil {
		t.Fatalf("LoadEnv failed: %v", err)
	}
	if out["FOO"] != "foo" || out["BAR"] != "bar" {
		t.Fatalf("unexpected map contents: %#v", out)
	}

	// Test LoadEnvFromFile with an existing temp file
	dir := t.TempDir()
	fpath := filepath.Join(dir, "envfile.env")
	if err := ioutil.WriteFile(fpath, []byte("BAZ=baz\n#comment\n"), 0644); err != nil {
		t.Fatalf("writing tmp file: %v", err)
	}
	m := map[string]string{}
	if err := LoadEnvFromFile(fpath, m); err != nil {
		t.Fatalf("LoadEnvFromFile returned error: %v", err)
	}
	if m["BAZ"] != "baz" {
		t.Fatalf("expected BAZ=baz, got: %q", m["BAZ"])
	}

	// Test LoadEnvFromFile with non-existing file returns an error
	if err := LoadEnvFromFile(filepath.Join(dir, "does-not-exist"), map[string]string{}); err == nil {
		t.Fatalf("expected error for missing file, got nil")
	}
}

func TestStripHelpers(t *testing.T) {
	// Ensure YAML doc delimiter is removed
	src := []byte("key=val\n---\nother=1\n")
	out := StripYAMLDocDelimiter(src)
	if string(out) == string(src) {
		t.Fatalf("StripYAMLDocDelimiter did not change input")
	}

	// Ensure comments are stripped
	src2 := []byte("FOO=1\n# a comment\nBAR=2\n")
	s2 := StripYamlComment(src2)
	if string(s2) == string(src2) {
		t.Fatalf("StripYamlComment did not change input")
	}
}

func TestEnvSubstAndRecursive(t *testing.T) {
	dir := t.TempDir()
	// create a file with a placeholder
	f := filepath.Join(dir, "hello.txt")
	if err := ioutil.WriteFile(f, []byte("Hello ${NAME}\n"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	envs := map[string]string{"NAME": "World"}
	modified, err := EnvSubst(f, envs)
	if err != nil {
		t.Fatalf("EnvSubst failed: %v", err)
	}
	if modified == "" || modified != "Hello World\n" {
		t.Fatalf("unexpected modified content: %q", modified)
	}

	// create another file that should be skipped by regex
	other := filepath.Join(dir, "ignore.me")
	if err := ioutil.WriteFile(other, []byte("Keep ${NAME}\n"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	// recursive substitution for *.txt only
	if err := EnvSubstRecursive(dir, `.*\.txt$`, envs); err != nil {
		t.Fatalf("EnvSubstRecursive failed: %v", err)
	}

	// check that hello.txt was substituted
	b, err := ioutil.ReadFile(f)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(b) != "Hello World\n" {
		t.Fatalf("unexpected file contents after recursive subst: %q", string(b))
	}

	// check that ignore.me was not changed (still contains placeholder)
	b2, err := ioutil.ReadFile(other)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(b2) != "Keep ${NAME}\n" {
		t.Fatalf("unexpected contents for ignored file: %q", string(b2))
	}
}

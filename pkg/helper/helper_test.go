package helper

import (
	"bytes"
	"testing"
)

func TestMarshalYaml_NoTopLevelIndent(t *testing.T) {
	type sample struct {
		Name  string `yaml:"name"`
		Value int    `yaml:"value"`
	}

	s := sample{Name: "foo", Value: 42}
	var buf bytes.Buffer
	if err := MarshalYaml(&buf, s); err != nil {
		t.Fatalf("MarshalYaml returned error: %v", err)
	}

	out := buf.String()
	// top-level keys should not be indented
	if !bytes.Contains([]byte(out), []byte("name: foo")) {
		t.Fatalf("expected top-level key 'name' without indent, got:\n%s", out)
	}
	if !bytes.Contains([]byte(out), []byte("value: 42")) {
		t.Fatalf("expected top-level key 'value' without indent, got:\n%s", out)
	}
}

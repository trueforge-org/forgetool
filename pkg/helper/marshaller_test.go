package helper

import (
	"bytes"
	"strings"
	"testing"
)

func TestMarshalYaml_SimpleMap(t *testing.T) {
	input := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	var buf bytes.Buffer
	if err := MarshalYaml(&buf, input); err != nil {
		t.Fatalf("MarshalYaml returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "key1: value1") {
		t.Errorf("expected 'key1: value1' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "key2: value2") {
		t.Errorf("expected 'key2: value2' in output, got:\n%s", out)
	}
}

func TestMarshalYaml_NestedStructure(t *testing.T) {
	type Inner struct {
		Field string `yaml:"field"`
	}
	type Outer struct {
		Name  string `yaml:"name"`
		Inner Inner  `yaml:"inner"`
	}

	input := Outer{
		Name:  "test",
		Inner: Inner{Field: "nested_value"},
	}

	var buf bytes.Buffer
	if err := MarshalYaml(&buf, input); err != nil {
		t.Fatalf("MarshalYaml returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "name: test") {
		t.Errorf("expected 'name: test' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "field: nested_value") {
		t.Errorf("expected 'field: nested_value' in output, got:\n%s", out)
	}
}

func TestMarshalYaml_EmptyInput(t *testing.T) {
	input := map[string]string{}

	var buf bytes.Buffer
	if err := MarshalYaml(&buf, input); err != nil {
		t.Fatalf("MarshalYaml returned error: %v", err)
	}

	out := strings.TrimSpace(buf.String())
	if out != "{}" {
		t.Errorf("expected '{}' for empty map, got:\n%s", out)
	}
}

func TestMarshalYaml_Indentation(t *testing.T) {
	type Child struct {
		Value string `yaml:"value"`
	}
	type Parent struct {
		Top   string `yaml:"top"`
		Child Child  `yaml:"child"`
	}

	input := Parent{
		Top:   "hello",
		Child: Child{Value: "world"},
	}

	var buf bytes.Buffer
	if err := MarshalYaml(&buf, input); err != nil {
		t.Fatalf("MarshalYaml returned error: %v", err)
	}

	lines := strings.Split(buf.String(), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		trimmed := strings.TrimLeft(line, " ")
		// Top-level keys should not be indented
		if strings.HasPrefix(trimmed, "top:") || strings.HasPrefix(trimmed, "child:") {
			if line != trimmed {
				t.Errorf("top-level key should not be indented: %q", line)
			}
		}
		// Nested key should be indented
		if strings.HasPrefix(trimmed, "value:") {
			if line == trimmed {
				t.Errorf("nested key should be indented: %q", line)
			}
		}
	}
}

func TestMarshalYaml_UnmarshalableValue(t *testing.T) {
	// yaml.v3 panics on unmarshalable types like channels;
	// verify that MarshalYaml does not silently succeed.
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic when marshalling a channel, but none occurred")
		}
	}()

	ch := make(chan int)
	var buf bytes.Buffer
	_ = MarshalYaml(&buf, ch)
}

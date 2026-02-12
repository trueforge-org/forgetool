package helper

import (
	"bytes"
	"strings"
	"testing"
)

func TestYamlNewEncoder(t *testing.T) {
	var buf bytes.Buffer
	encoder := YamlNewEncoder(&buf)

	if encoder == nil {
		t.Fatal("YamlNewEncoder returned nil")
	}

	if encoder.writer != &buf {
		t.Error("Writer not set correctly")
	}

	if encoder.indent != 0 {
		t.Errorf("Default indent = %d, want 0", encoder.indent)
	}
}

func TestEncoder_SetIndent(t *testing.T) {
	var buf bytes.Buffer
	encoder := YamlNewEncoder(&buf)

	tests := []int{0, 2, 4, 8}
	for _, indent := range tests {
		encoder.SetIndent(indent)
		if encoder.indent != indent {
			t.Errorf("SetIndent(%d): indent = %d, want %d", indent, encoder.indent, indent)
		}
	}
}

func TestEncoder_Encode(t *testing.T) {
	tests := []struct {
		name   string
		data   interface{}
		indent int
		check  func(t *testing.T, got string)
	}{
		{
			name:   "Simple struct no indent",
			data:   struct{ Name string }{Name: "test"},
			indent: 0,
			check: func(t *testing.T, got string) {
				if !strings.HasPrefix(got, "name: test") {
					t.Errorf("Expected 'name: test', got: %q", got)
				}
			},
		},
		{
			name:   "Simple struct with indent",
			data:   struct{ Name string }{Name: "test"},
			indent: 2,
			check: func(t *testing.T, got string) {
				if !strings.HasPrefix(got, "  name: test") {
					t.Errorf("Expected indented output, got: %q", got)
				}
			},
		},
		{
			name:   "Map no indent",
			data:   map[string]string{"key": "value"},
			indent: 0,
			check: func(t *testing.T, got string) {
				if !strings.Contains(got, "key: value") {
					t.Errorf("Expected 'key: value', got: %q", got)
				}
			},
		},
		{
			name:   "Map with indent",
			data:   map[string]string{"key": "value"},
			indent: 4,
			check: func(t *testing.T, got string) {
				if !strings.Contains(got, "    key: value") {
					t.Errorf("Expected 4-space indent, got: %q", got)
				}
			},
		},
		{
			name:   "Slice no indent",
			data:   []string{"a", "b", "c"},
			indent: 0,
			check: func(t *testing.T, got string) {
				if !strings.Contains(got, "- a") || !strings.Contains(got, "- b") {
					t.Errorf("Expected list items, got: %q", got)
				}
			},
		},
		{
			name:   "Slice with indent",
			data:   []string{"a", "b"},
			indent: 2,
			check: func(t *testing.T, got string) {
				if !strings.Contains(got, "  - a") || !strings.Contains(got, "  - b") {
					t.Errorf("Expected indented list, got: %q", got)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			encoder := YamlNewEncoder(&buf)
			encoder.SetIndent(tt.indent)

			err := encoder.Encode(tt.data)
			if err != nil {
				t.Fatalf("Encode() error = %v", err)
			}

			got := buf.String()
			tt.check(t, got)
		})
	}
}

func TestEncoder_Encode_ComplexStruct(t *testing.T) {
	type Inner struct {
		Value string
	}
	type Outer struct {
		Name  string
		Inner Inner
	}

	data := Outer{
		Name:  "test",
		Inner: Inner{Value: "nested"},
	}

	var buf bytes.Buffer
	encoder := YamlNewEncoder(&buf)
	encoder.SetIndent(2)

	err := encoder.Encode(data)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	got := buf.String()
	// Check that output is indented
	if !strings.HasPrefix(got, "  ") {
		t.Error("Expected indented output")
	}
	// Check that nested structure is present
	if !strings.Contains(got, "name:") || !strings.Contains(got, "value:") {
		t.Error("Expected nested structure in output")
	}
}

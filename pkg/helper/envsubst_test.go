package helper

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadEnv(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		want    map[string]string
		wantErr bool
	}{
		{
			name:    "Simple key-value pairs",
			content: []byte("KEY1=value1\nKEY2=value2"),
			want: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
			},
			wantErr: false,
		},
		{
			name:    "Empty content",
			content: []byte(""),
			want:    map[string]string{},
			wantErr: false,
		},
		{
			name:    "With quotes",
			content: []byte(`KEY="quoted value"`),
			want: map[string]string{
				"KEY": "quoted value",
			},
			wantErr: false,
		},
		{
			name:    "With spaces",
			content: []byte("KEY=value with spaces"),
			want: map[string]string{
				"KEY": "value with spaces",
			},
			wantErr: false,
		},
		{
			name:    "Multiple lines with empty lines",
			content: []byte("KEY1=val1\n\nKEY2=val2\n"),
			want: map[string]string{
				"KEY1": "val1",
				"KEY2": "val2",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := make(map[string]string)
			err := LoadEnv(tt.content, output)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadEnv() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(output) != len(tt.want) {
					t.Errorf("LoadEnv() got %d entries, want %d", len(output), len(tt.want))
				}
				for k, v := range tt.want {
					if output[k] != v {
						t.Errorf("LoadEnv() key %q = %q, want %q", k, output[k], v)
					}
				}
			}
		})
	}
}

func TestLoadEnvFromFile(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(t *testing.T) string
		want      map[string]string
		wantErr   bool
	}{
		{
			name: "Valid file",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				filePath := filepath.Join(tmpDir, "test.env")
				content := "KEY1=value1\nKEY2=value2"
				os.WriteFile(filePath, []byte(content), 0644)
				return filePath
			},
			want: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
			},
			wantErr: false,
		},
		{
			name: "File with YAML comments",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				filePath := filepath.Join(tmpDir, "test.yaml")
				content := "# This is a comment\nKEY1=value1\n# Another comment\nKEY2=value2"
				os.WriteFile(filePath, []byte(content), 0644)
				return filePath
			},
			want: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
			},
			wantErr: false,
		},
		{
			name: "Nonexistent file",
			setupFunc: func(t *testing.T) string {
				return "/nonexistent/file.env"
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := tt.setupFunc(t)
			output := make(map[string]string)

			err := LoadEnvFromFile(filePath, output)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadEnvFromFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				for k, v := range tt.want {
					if output[k] != v {
						t.Errorf("LoadEnvFromFile() key %q = %q, want %q", k, output[k], v)
					}
				}
			}
		})
	}
}

func TestStripYamlComment(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		check func(t *testing.T, output []byte)
	}{
		{
			name:  "No comments",
			input: []byte("KEY1=value1\nKEY2=value2"),
			check: func(t *testing.T, output []byte) {
				s := string(output)
				if !strings.Contains(s, "KEY1=value1") || !strings.Contains(s, "KEY2=value2") {
					t.Errorf("Expected key-value pairs to remain, got: %s", output)
				}
			},
		},
		{
			name:  "With comments",
			input: []byte("# Comment\nKEY1=value1\n# Another comment\nKEY2=value2"),
			check: func(t *testing.T, output []byte) {
				s := string(output)
				if strings.Contains(s, "# Comment") || strings.Contains(s, "# Another comment") {
					t.Error("Expected comments to be stripped")
				}
				if !strings.Contains(s, "KEY1=value1") || !strings.Contains(s, "KEY2=value2") {
					t.Error("Expected key-value pairs to remain")
				}
			},
		},
		{
			name:  "Empty input",
			input: []byte(""),
			check: func(t *testing.T, output []byte) {
				if len(output) != 0 {
					t.Errorf("Expected empty output, got: %s", output)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := StripYamlComment(tt.input)
			tt.check(t, output)
		})
	}
}

func TestStripYAMLDocDelimiter(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		check func(t *testing.T, output []byte)
	}{
		{
			name:  "With document delimiter",
			input: []byte("---\nKEY1=value1\nKEY2=value2"),
			check: func(t *testing.T, output []byte) {
				s := string(output)
				// Delimiter should be replaced with newline
				if strings.HasPrefix(s, "---") {
					t.Error("Expected --- to be removed")
				}
				if !strings.Contains(s, "KEY1=value1") {
					t.Error("Expected content to remain")
				}
			},
		},
		{
			name:  "Without delimiter",
			input: []byte("KEY1=value1\nKEY2=value2"),
			check: func(t *testing.T, output []byte) {
				if string(output) != string([]byte("KEY1=value1\nKEY2=value2")) {
					t.Errorf("Expected no change, got: %s", output)
				}
			},
		},
		{
			name:  "Multiple delimiters",
			input: []byte("---\nKEY1=value1\n---\nKEY2=value2"),
			check: func(t *testing.T, output []byte) {
				s := string(output)
				if strings.Contains(s, "---") {
					t.Error("Expected all --- to be removed")
				}
			},
		},
		{
			name:  "Empty input",
			input: []byte(""),
			check: func(t *testing.T, output []byte) {
				if len(output) != 0 {
					t.Errorf("Expected empty output, got: %s", output)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := StripYAMLDocDelimiter(tt.input)
			tt.check(t, output)
		})
	}
}

// Note: EnvSubst and EnvSubstRecursive functions are not tested here as they would require
// complex file system setup and are better suited for integration tests.
// These functions operate on actual file systems and directories.

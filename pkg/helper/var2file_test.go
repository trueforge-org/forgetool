package helper

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVarToFile(t *testing.T) {
	tests := []struct {
		name       string
		filename   string
		content    string
		setupFunc  func(t *testing.T, filename string)
		wantErr    bool
		checkFunc  func(t *testing.T, filename string)
	}{
		{
			name:     "Create new file with content",
			filename: "test_new.txt",
			content:  "test content",
			setupFunc: func(t *testing.T, filename string) {
				// No setup needed - file should not exist
			},
			wantErr: false,
			checkFunc: func(t *testing.T, filename string) {
				content, err := os.ReadFile(filename)
				if err != nil {
					t.Fatalf("Failed to read created file: %v", err)
				}
				if string(content) != "test content" {
					t.Errorf("File content = %q, want %q", string(content), "test content")
				}
			},
		},
		{
			name:     "Do not overwrite existing file",
			filename: "test_existing.txt",
			content:  "new content",
			setupFunc: func(t *testing.T, filename string) {
				err := os.WriteFile(filename, []byte("original content"), 0644)
				if err != nil {
					t.Fatalf("Failed to setup existing file: %v", err)
				}
			},
			wantErr: false,
			checkFunc: func(t *testing.T, filename string) {
				content, err := os.ReadFile(filename)
				if err != nil {
					t.Fatalf("Failed to read file: %v", err)
				}
				// Should still have original content
				if string(content) != "original content" {
					t.Errorf("File content = %q, want %q (should not be overwritten)", string(content), "original content")
				}
			},
		},
		{
			name:     "Empty content",
			filename: "test_empty.txt",
			content:  "",
			setupFunc: func(t *testing.T, filename string) {
				// No setup needed
			},
			wantErr: false,
			checkFunc: func(t *testing.T, filename string) {
				content, err := os.ReadFile(filename)
				if err != nil {
					t.Fatalf("Failed to read file: %v", err)
				}
				if string(content) != "" {
					t.Errorf("File content = %q, want empty string", string(content))
				}
			},
		},
		{
			name:     "Multiline content",
			filename: "test_multiline.txt",
			content:  "line1\nline2\nline3",
			setupFunc: func(t *testing.T, filename string) {
				// No setup needed
			},
			wantErr: false,
			checkFunc: func(t *testing.T, filename string) {
				content, err := os.ReadFile(filename)
				if err != nil {
					t.Fatalf("Failed to read file: %v", err)
				}
				if string(content) != "line1\nline2\nline3" {
					t.Errorf("File content mismatch")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			fullPath := filepath.Join(tmpDir, tt.filename)

			tt.setupFunc(t, fullPath)

			err := VarToFile(fullPath, tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("VarToFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				tt.checkFunc(t, fullPath)
			}
		})
	}
}

func TestVarToFile_InvalidPath(t *testing.T) {
	// Test with an invalid directory path
	err := VarToFile("/nonexistent/directory/file.txt", "content")
	if err == nil {
		t.Error("Expected error for invalid path, got nil")
	}
}

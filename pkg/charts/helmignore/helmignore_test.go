package helmignore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSmokeHelmignore(t *testing.T) {}

func TestGenerateHelmIgnore(t *testing.T) {
	tests := []struct {
		name         string
		setupFunc    func(t *testing.T) (templatePath, chartPath string)
		wantErr      bool
		checkContent func(t *testing.T, target string)
	}{
		{
			name: "Valid template and chart path",
			setupFunc: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				
				// Create template directory and file
				templateDir := filepath.Join(tmpDir, "templates")
				os.MkdirAll(templateDir, 0755)
				templateContent := "*.tmp\n*.bak\n.DS_Store\n"
				os.WriteFile(filepath.Join(templateDir, "helmignore.tpl"), []byte(templateContent), 0644)
				
				// Create chart directory
				chartDir := filepath.Join(tmpDir, "chart")
				os.MkdirAll(chartDir, 0755)
				chartPath := filepath.Join(chartDir, "Chart.yaml")
				
				return tmpDir, chartPath
			},
			wantErr: false,
			checkContent: func(t *testing.T, target string) {
				content, err := os.ReadFile(target)
				if err != nil {
					t.Fatalf("Failed to read generated file: %v", err)
				}
				if string(content) != "*.tmp\n*.bak\n.DS_Store\n" {
					t.Errorf("Content mismatch: got %q", content)
				}
			},
		},
		{
			name: "Template file does not exist",
			setupFunc: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				chartDir := filepath.Join(tmpDir, "chart")
				os.MkdirAll(chartDir, 0755)
				chartPath := filepath.Join(chartDir, "Chart.yaml")
				
				return tmpDir, chartPath
			},
			wantErr: true,
			checkContent: func(t *testing.T, target string) {
				// No check needed for error case
			},
		},
		{
			name: "Empty template content",
			setupFunc: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				
				templateDir := filepath.Join(tmpDir, "templates")
				os.MkdirAll(templateDir, 0755)
				os.WriteFile(filepath.Join(templateDir, "helmignore.tpl"), []byte(""), 0644)
				
				chartDir := filepath.Join(tmpDir, "chart")
				os.MkdirAll(chartDir, 0755)
				chartPath := filepath.Join(chartDir, "Chart.yaml")
				
				return tmpDir, chartPath
			},
			wantErr: false,
			checkContent: func(t *testing.T, target string) {
				content, err := os.ReadFile(target)
				if err != nil {
					t.Fatalf("Failed to read generated file: %v", err)
				}
				if len(content) != 0 {
					t.Errorf("Expected empty content, got %q", content)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			templatePath, chartPath := tt.setupFunc(t)

			err := GenerateHelmIgnore(templatePath, chartPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateHelmIgnore() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				target := filepath.Join(filepath.Dir(chartPath), ".helmignore")
				if _, err := os.Stat(target); os.IsNotExist(err) {
					t.Fatal(".helmignore file was not created")
				}
				tt.checkContent(t, target)
			}
		})
	}
}


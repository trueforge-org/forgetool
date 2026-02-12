package readme

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSmokeReadme(t *testing.T) {}

func TestGenerateReadme(t *testing.T) {
	tests := []struct {
		name         string
		setupFunc    func(t *testing.T) (templatePath, chartPath, chartName, train string)
		wantErr      bool
		checkContent func(t *testing.T, target, chartName, train string)
	}{
		{
			name: "Valid template with placeholders",
			setupFunc: func(t *testing.T) (string, string, string, string) {
				tmpDir := t.TempDir()
				
				// Create template directory and file
				templateDir := filepath.Join(tmpDir, "templates")
				os.MkdirAll(templateDir, 0755)
				templateContent := "# CHARTPLACEHOLDER\n\nTrain: TRAINPLACEHOLDER\n"
				os.WriteFile(filepath.Join(templateDir, "README.md.tpl"), []byte(templateContent), 0644)
				
				// Create chart directory
				chartDir := filepath.Join(tmpDir, "chart")
				os.MkdirAll(chartDir, 0755)
				chartPath := filepath.Join(chartDir, "Chart.yaml")
				
				return tmpDir, chartPath, "test-chart", "stable"
			},
			wantErr: false,
			checkContent: func(t *testing.T, target, chartName, train string) {
				content, err := os.ReadFile(target)
				if err != nil {
					t.Fatalf("Failed to read generated file: %v", err)
				}
				
				contentStr := string(content)
				if !strings.Contains(contentStr, chartName) {
					t.Errorf("README does not contain chart name %q", chartName)
				}
				if !strings.Contains(contentStr, train) {
					t.Errorf("README does not contain train %q", train)
				}
				if strings.Contains(contentStr, "CHARTPLACEHOLDER") || strings.Contains(contentStr, "TRAINPLACEHOLDER") {
					t.Error("README still contains placeholders")
				}
			},
		},
		{
			name: "Template file does not exist",
			setupFunc: func(t *testing.T) (string, string, string, string) {
				tmpDir := t.TempDir()
				chartDir := filepath.Join(tmpDir, "chart")
				os.MkdirAll(chartDir, 0755)
				chartPath := filepath.Join(chartDir, "Chart.yaml")
				
				return tmpDir, chartPath, "test-chart", "stable"
			},
			wantErr: true,
			checkContent: func(t *testing.T, target, chartName, train string) {
				// No check needed for error case
			},
		},
		{
			name: "Multiple placeholder replacements",
			setupFunc: func(t *testing.T) (string, string, string, string) {
				tmpDir := t.TempDir()
				
				templateDir := filepath.Join(tmpDir, "templates")
				os.MkdirAll(templateDir, 0755)
				// Template with multiple occurrences
				templateContent := "# CHARTPLACEHOLDER\n\nThis is CHARTPLACEHOLDER in TRAINPLACEHOLDER train.\n\nMore about CHARTPLACEHOLDER.\n"
				os.WriteFile(filepath.Join(templateDir, "README.md.tpl"), []byte(templateContent), 0644)
				
				chartDir := filepath.Join(tmpDir, "chart")
				os.MkdirAll(chartDir, 0755)
				chartPath := filepath.Join(chartDir, "Chart.yaml")
				
				return tmpDir, chartPath, "my-app", "incubator"
			},
			wantErr: false,
			checkContent: func(t *testing.T, target, chartName, train string) {
				content, err := os.ReadFile(target)
				if err != nil {
					t.Fatalf("Failed to read generated file: %v", err)
				}
				
				contentStr := string(content)
				// Count occurrences
				chartCount := strings.Count(contentStr, chartName)
				if chartCount != 3 {
					t.Errorf("Expected 3 occurrences of %q, got %d", chartName, chartCount)
				}
				trainCount := strings.Count(contentStr, train)
				if trainCount != 1 {
					t.Errorf("Expected 1 occurrence of %q, got %d", train, trainCount)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			templatePath, chartPath, chartName, train := tt.setupFunc(t)

			err := GenerateReadme(templatePath, chartPath, chartName, train)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateReadme() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				target := filepath.Join(filepath.Dir(chartPath), "README.md")
				if _, err := os.Stat(target); os.IsNotExist(err) {
					t.Fatal("README.md file was not created")
				}
				tt.checkContent(t, target, chartName, train)
			}
		})
	}
}


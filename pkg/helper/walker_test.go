package helper

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWalkCharts2(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(t *testing.T) ([]string, int)
		mode      WalkMode
		wantErr   bool
	}{
		{
			name: "Walk directory with Chart.yaml files in sync mode",
			setupFunc: func(t *testing.T) ([]string, int) {
				tmpDir := t.TempDir()
				
				// Create chart directories
				chart1 := filepath.Join(tmpDir, "chart1")
				chart2 := filepath.Join(tmpDir, "chart2")
				
				os.MkdirAll(chart1, 0755)
				os.MkdirAll(chart2, 0755)
				
				// Create Chart.yaml files
				os.WriteFile(filepath.Join(chart1, "Chart.yaml"), []byte("name: chart1\nversion: 1.0.0\n"), 0644)
				os.WriteFile(filepath.Join(chart2, "Chart.yaml"), []byte("name: chart2\nversion: 1.0.0\n"), 0644)
				
				return []string{tmpDir}, 2
			},
			mode:    SyncMode,
			wantErr: false,
		},
		{
			name: "Walk directory in async mode",
			setupFunc: func(t *testing.T) ([]string, int) {
				tmpDir := t.TempDir()
				
				chart1 := filepath.Join(tmpDir, "chart1")
				os.MkdirAll(chart1, 0755)
				os.WriteFile(filepath.Join(chart1, "Chart.yaml"), []byte("name: chart1\n"), 0644)
				
				return []string{tmpDir}, 1
			},
			mode:    AsyncMode,
			wantErr: false,
		},
		{
			name: "Empty paths defaults to ./charts",
			setupFunc: func(t *testing.T) ([]string, int) {
				// This will try to walk ./charts which may or may not exist
				return []string{}, 0
			},
			mode:    SyncMode,
			wantErr: false,
		},
		{
			name: "Multiple paths",
			setupFunc: func(t *testing.T) ([]string, int) {
				tmpDir1 := t.TempDir()
				tmpDir2 := t.TempDir()
				
				os.MkdirAll(tmpDir1, 0755)
				os.MkdirAll(tmpDir2, 0755)
				
				os.WriteFile(filepath.Join(tmpDir1, "file1.txt"), []byte("test"), 0644)
				os.WriteFile(filepath.Join(tmpDir2, "file2.txt"), []byte("test"), 0644)
				
				return []string{tmpDir1, tmpDir2}, 2
			},
			mode:    SyncMode,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths, expectedMin := tt.setupFunc(t)

			var foundPaths []string
			walkFunc := func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				foundPaths = append(foundPaths, path)
				return nil
			}

			err := WalkCharts2(paths, walkFunc, tt.mode)
			if (err != nil) != tt.wantErr {
				t.Errorf("WalkCharts2() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && expectedMin > 0 {
				if len(foundPaths) < expectedMin {
					t.Errorf("Found %d paths, expected at least %d", len(foundPaths), expectedMin)
				}
			}
		})
	}
}

func TestWalkCharts2_NonexistentPath(t *testing.T) {
	// Test with a path that doesn't exist
	// The function logs errors but doesn't return them
	walkFunc := func(path string, d fs.DirEntry, err error) error {
		return nil
	}

	err := WalkCharts2([]string{"/nonexistent/path/that/does/not/exist"}, walkFunc, SyncMode)
	// Function returns nil even if paths don't exist (logs internally)
	if err != nil {
		t.Errorf("WalkCharts2() with nonexistent path returned error: %v", err)
	}
}

func TestWalkCharts2_WalkFuncReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(tmpDir, 0755)
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644)

	// Walk function that returns an error on specific condition
	walkFunc := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, "test.txt") {
			return fs.SkipDir
		}
		return nil
	}

	// The error is logged but not returned by WalkCharts2
	err := WalkCharts2([]string{tmpDir}, walkFunc, SyncMode)
	if err != nil {
		t.Errorf("WalkCharts2() error = %v, expected nil (errors are logged, not returned)", err)
	}
}

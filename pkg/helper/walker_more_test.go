package helper

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestWalkCharts2_EmptyPaths(t *testing.T) {
	// With empty paths, WalkCharts2 defaults to "./charts" which likely doesn't exist.
	// It should not return an error (errors are logged, not returned).
	var calls int32
	fn := func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		atomic.AddInt32(&calls, 1)
		return nil
	}

	err := WalkCharts2(nil, fn, SyncMode)
	if err != nil {
		t.Fatalf("WalkCharts2 with nil paths should not return error, got: %v", err)
	}

	err = WalkCharts2([]string{}, fn, AsyncMode)
	if err != nil {
		t.Fatalf("WalkCharts2 with empty paths should not return error, got: %v", err)
	}
}

func TestWalkCharts2_MultiplePaths(t *testing.T) {
	td := t.TempDir()

	// Create two separate directory trees
	dir1 := filepath.Join(td, "dir1")
	dir2 := filepath.Join(td, "dir2")
	if err := os.MkdirAll(dir1, 0755); err != nil {
		t.Fatalf("mkdir dir1: %v", err)
	}
	if err := os.MkdirAll(dir2, 0755); err != nil {
		t.Fatalf("mkdir dir2: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir1, "Chart.yaml"), []byte("name: c1\n"), 0644); err != nil {
		t.Fatalf("write chart1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir2, "Chart.yaml"), []byte("name: c2\n"), 0644); err != nil {
		t.Fatalf("write chart2: %v", err)
	}

	var found int32
	fn := func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, "Chart.yaml") {
			atomic.AddInt32(&found, 1)
		}
		return nil
	}

	// Test both modes with multiple paths
	for _, mode := range []WalkMode{SyncMode, AsyncMode} {
		atomic.StoreInt32(&found, 0)
		if err := WalkCharts2([]string{dir1, dir2}, fn, mode); err != nil {
			t.Fatalf("WalkCharts2 multiple paths (mode=%d) failed: %v", mode, err)
		}
		if atomic.LoadInt32(&found) != 2 {
			t.Fatalf("expected 2 Chart.yaml found (mode=%d), got %d", mode, found)
		}
	}
}

func TestWalkCharts2_NonExistentDir(t *testing.T) {
	// WalkCharts2 logs errors but always returns nil
	var calls int32
	fn := func(path string, d os.DirEntry, err error) error {
		atomic.AddInt32(&calls, 1)
		return err
	}

	err := WalkCharts2([]string{"/nonexistent/path/xyz"}, fn, SyncMode)
	if err != nil {
		t.Fatalf("WalkCharts2 should not return error for nonexistent dir, got: %v", err)
	}

	err = WalkCharts2([]string{"/nonexistent/path/xyz"}, fn, AsyncMode)
	if err != nil {
		t.Fatalf("WalkCharts2 async should not return error for nonexistent dir, got: %v", err)
	}
}

func TestWalkCharts_ExcludedDirs(t *testing.T) {
	td := t.TempDir()

	// Create charts in both normal and excluded directories
	normalChart := filepath.Join(td, "mychart")
	excludedChart := filepath.Join(td, "templates")
	if err := os.MkdirAll(normalChart, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(excludedChart, 0755); err != nil {
		t.Fatalf("mkdir excluded: %v", err)
	}
	if err := os.WriteFile(filepath.Join(normalChart, "Chart.yaml"), []byte("name: ok\nversion: 1.0.0\n"), 0644); err != nil {
		t.Fatalf("write normal chart: %v", err)
	}
	if err := os.WriteFile(filepath.Join(excludedChart, "Chart.yaml"), []byte("name: excluded\nversion: 1.0.0\n"), 0644); err != nil {
		t.Fatalf("write excluded chart: %v", err)
	}

	var mu sync.Mutex
	var seen []string
	action := func(path, bump string) error {
		mu.Lock()
		seen = append(seen, path)
		mu.Unlock()
		return nil
	}

	if err := WalkCharts([]string{td}, action, "", SyncMode); err != nil {
		t.Fatalf("WalkCharts failed: %v", err)
	}

	// Only the normal chart should be found, not the one in "templates"
	for _, p := range seen {
		if strings.Contains(p, "templates") {
			t.Fatalf("excluded directory 'templates' should be skipped, but found: %s", p)
		}
	}
	if len(seen) != 1 {
		t.Fatalf("expected 1 chart, got %d: %v", len(seen), seen)
	}
}

func TestWalkCharts_MultipleExcludedDirs(t *testing.T) {
	td := t.TempDir()

	// Create charts in various excluded directories
	for _, excluded := range []string{".github", "docs", ".vscode", "testdata"} {
		dir := filepath.Join(td, excluded)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("mkdir %s: %v", excluded, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "Chart.yaml"), []byte("name: x\nversion: 0.0.1\n"), 0644); err != nil {
			t.Fatalf("write chart in %s: %v", excluded, err)
		}
	}

	// Create one valid chart
	validDir := filepath.Join(td, "valid")
	if err := os.MkdirAll(validDir, 0755); err != nil {
		t.Fatalf("mkdir valid: %v", err)
	}
	if err := os.WriteFile(filepath.Join(validDir, "Chart.yaml"), []byte("name: valid\nversion: 0.0.1\n"), 0644); err != nil {
		t.Fatalf("write valid chart: %v", err)
	}

	var mu sync.Mutex
	var seen []string
	action := func(path, bump string) error {
		mu.Lock()
		seen = append(seen, path)
		mu.Unlock()
		return nil
	}

	if err := WalkCharts([]string{td}, action, "", SyncMode); err != nil {
		t.Fatalf("WalkCharts failed: %v", err)
	}

	if len(seen) != 1 {
		t.Fatalf("expected only 1 chart (valid), got %d: %v", len(seen), seen)
	}
	if !strings.Contains(seen[0], "valid") {
		t.Fatalf("expected valid chart path, got: %s", seen[0])
	}
}

func TestWalkCharts_AsyncModeMultipleCharts(t *testing.T) {
	td := t.TempDir()

	// Create multiple valid chart directories
	for _, name := range []string{"chart-a", "chart-b", "chart-c"} {
		dir := filepath.Join(td, name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "Chart.yaml"), []byte("name: "+name+"\nversion: 0.0.1\n"), 0644); err != nil {
			t.Fatalf("write chart %s: %v", name, err)
		}
	}

	var mu sync.Mutex
	var seen []string
	action := func(path, bump string) error {
		mu.Lock()
		seen = append(seen, path)
		mu.Unlock()
		return nil
	}

	if err := WalkCharts([]string{td}, action, "patch", AsyncMode); err != nil {
		t.Fatalf("WalkCharts async failed: %v", err)
	}

	if len(seen) != 3 {
		t.Fatalf("expected 3 charts, got %d: %v", len(seen), seen)
	}
}

func TestWalkCharts_NonExistentPath(t *testing.T) {
	action := func(path, bump string) error {
		return nil
	}

	err := WalkCharts([]string{"/nonexistent/path/xyz"}, action, "", SyncMode)
	if err == nil {
		t.Fatalf("expected error for nonexistent path")
	}
}

func TestGetWalkDirFunc_SkipsExcludedAndFindsChart(t *testing.T) {
	td := t.TempDir()

	// Create structure: td/valid/Chart.yaml and td/templates/Chart.yaml
	validDir := filepath.Join(td, "valid")
	tmplDir := filepath.Join(td, "templates")
	if err := os.MkdirAll(validDir, 0755); err != nil {
		t.Fatalf("mkdir valid: %v", err)
	}
	if err := os.MkdirAll(tmplDir, 0755); err != nil {
		t.Fatalf("mkdir templates: %v", err)
	}
	if err := os.WriteFile(filepath.Join(validDir, "Chart.yaml"), []byte("name: v\nversion: 0.0.1\n"), 0644); err != nil {
		t.Fatalf("write valid: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmplDir, "Chart.yaml"), []byte("name: t\nversion: 0.0.1\n"), 0644); err != nil {
		t.Fatalf("write tmpl: %v", err)
	}

	var mu sync.Mutex
	var paths []string
	var wg sync.WaitGroup
	action := func(path, bump string) error {
		mu.Lock()
		paths = append(paths, path)
		mu.Unlock()
		return nil
	}

	fn := getWalkDirFunc(action, "minor", SyncMode, &wg)

	if err := filepath.WalkDir(td, fn); err != nil {
		t.Fatalf("WalkDir failed: %v", err)
	}

	if len(paths) != 1 {
		t.Fatalf("expected 1 path, got %d: %v", len(paths), paths)
	}
	if !strings.Contains(paths[0], "valid") {
		t.Fatalf("expected valid chart, got: %s", paths[0])
	}
}

func TestWalkCharts2_SyncModeSequential(t *testing.T) {
	td := t.TempDir()

	// Create two dirs
	for _, name := range []string{"first", "second"} {
		dir := filepath.Join(td, name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "data.txt"), []byte(name), 0644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	var mu sync.Mutex
	var order []string
	fn := func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			mu.Lock()
			order = append(order, filepath.Base(filepath.Dir(path)))
			mu.Unlock()
		}
		return nil
	}

	dir1 := filepath.Join(td, "first")
	dir2 := filepath.Join(td, "second")
	if err := WalkCharts2([]string{dir1, dir2}, fn, SyncMode); err != nil {
		t.Fatalf("WalkCharts2 sync failed: %v", err)
	}

	if len(order) != 2 {
		t.Fatalf("expected 2 files visited, got %d", len(order))
	}
	// In sync mode, first should be processed before second
	if order[0] != "first" || order[1] != "second" {
		t.Fatalf("expected sequential order [first, second], got %v", order)
	}
}

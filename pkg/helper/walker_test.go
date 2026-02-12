package helper

import (
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestWalkCharts2_SyncMode(t *testing.T) {
	td := t.TempDir()
	chartDir := filepath.Join(td, "trainX", "chart1")
	if err := os.MkdirAll(chartDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	chartPath := filepath.Join(chartDir, "Chart.yaml")
	if err := os.WriteFile(chartPath, []byte("name: c\nversion: 0.0.1\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	var calls int32
	fn := func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, "Chart.yaml") {
			atomic.AddInt32(&calls, 1)
			return filepath.SkipDir
		}
		return nil
	}

	if err := WalkCharts2([]string{td}, fn, SyncMode); err != nil {
		t.Fatalf("WalkCharts2 failed: %v", err)
	}

	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("expected 1 call to fn, got %d", calls)
	}
}

func TestWalkCharts2_AsyncMode(t *testing.T) {
	td := t.TempDir()
	chartDir := filepath.Join(td, "trainY", "chart2")
	if err := os.MkdirAll(chartDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	chartPath := filepath.Join(chartDir, "Chart.yaml")
	if err := os.WriteFile(chartPath, []byte("name: c2\nversion: 0.0.2\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	var calls int32
	fn := func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, "Chart.yaml") {
			atomic.AddInt32(&calls, 1)
			return filepath.SkipDir
		}
		return nil
	}

	if err := WalkCharts2([]string{td}, fn, AsyncMode); err != nil {
		t.Fatalf("WalkCharts2 failed: %v", err)
	}

	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("expected 1 call to fn, got %d", calls)
	}
}

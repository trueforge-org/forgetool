package helper

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestUniqueNonEmptyElementsOf(t *testing.T) {
	in := []string{"a", "", "b", "a", "c", "b", ""}
	got := UniqueNonEmptyElementsOf(in)

	// Expect unique non-empty elements in first-seen order: a,b,c
	if len(got) != 3 {
		t.Fatalf("unexpected length: got %d want 3; got=%v", len(got), got)
	}
	if got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("unexpected values: %v", got)
	}
}

func TestWalkCharts_SyncAndAsync(t *testing.T) {
	// prepare a temporary charts layout using repo testdata
	td := t.TempDir()
	chartsDir := filepath.Join(td, "charts", "mychart")
	if err := os.MkdirAll(chartsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// copy an existing sample Chart.yaml from repo testdata (repo root)
	src := filepath.Join("..", "..", "testdata", "chart_yaml", "validChart.yaml")
	dst := filepath.Join(chartsDir, "Chart.yaml")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read sample: %v", err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("write chart: %v", err)
	}

	// action collector
	collect := func(mode WalkMode) []string {
		var mu sync.Mutex
		var seen []string
		action := func(path, bump string) error {
			mu.Lock()
			seen = append(seen, path)
			mu.Unlock()
			return nil
		}

		if err := WalkCharts([]string{filepath.Join(td, "charts")}, action, "", mode); err != nil {
			t.Fatalf("WalkCharts returned error: %v", err)
		}
		return seen
	}

	// Sync
	s := collect(SyncMode)
	if len(s) != 1 {
		t.Fatalf("sync: expected 1 chart, got %d (%v)", len(s), s)
	}

	// Async
	a := collect(AsyncMode)
	if len(a) != 1 {
		t.Fatalf("async: expected 1 chart, got %d (%v)", len(a), a)
	}
}

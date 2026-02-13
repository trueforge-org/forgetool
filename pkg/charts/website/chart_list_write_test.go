package website

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteChartList_NilList(t *testing.T) {
	opts := &ChartListOptions{OutputPath: filepath.Join(t.TempDir(), "out.json")}
	if err := opts.WriteChartList(); err == nil {
		t.Fatalf("expected error when list is nil")
	}
}

func TestWriteChartList_WritesFile(t *testing.T) {
	td := t.TempDir()
	out := td + "/charts.json"
	opts := &ChartListOptions{OutputPath: out}
	opts.list = &ChartList{
		TotalCount: 1,
		Trains:     []Train{{Name: "t", Count: 1, Charts: []Chart{{Name: "c"}}}},
	}

	if err := opts.WriteChartList(); err != nil {
		t.Fatalf("WriteChartList failed: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("expected output file: %v", err)
	}
	var cl ChartList
	if err := json.Unmarshal(data, &cl); err != nil {
		t.Fatalf("invalid json written: %v", err)
	}
	if cl.TotalCount != 1 || len(cl.Trains) != 1 || cl.Trains[0].Name != "t" {
		t.Fatalf("unexpected content: %+v", cl)
	}
}

func TestWriteChartList_WriteError(t *testing.T) {
	td := t.TempDir()
	opts := &ChartListOptions{OutputPath: td}
	opts.list = &ChartList{TotalCount: 1}
	if err := opts.WriteChartList(); err == nil {
		t.Fatalf("expected write error when output path is a directory")
	}
}

func TestWriteChartList_MarshalError(t *testing.T) {
	orig := marshalChartList
	marshalChartList = func(_ interface{}) ([]byte, error) {
		return nil, errors.New("marshal fail")
	}
	t.Cleanup(func() { marshalChartList = orig })

	opts := &ChartListOptions{OutputPath: filepath.Join(t.TempDir(), "out.json")}
	opts.list = &ChartList{TotalCount: 1}
	if err := opts.WriteChartList(); err == nil {
		t.Fatalf("expected marshal error")
	}
}

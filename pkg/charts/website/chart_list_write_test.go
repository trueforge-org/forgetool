package website

import (
	"encoding/json"
	"os"
	"testing"
)

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

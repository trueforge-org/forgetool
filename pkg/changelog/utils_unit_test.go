package changelog

import (
	"sync"
	"testing"
)

func TestGetChartNameDeeperPath(t *testing.T) {
	got := getChartName("charts/stable/mychart/templates/deployment.yaml")
	if got != "mychart" {
		t.Fatalf("expected mychart, got %s", got)
	}
}

func TestGetChartNameSingleSegment(t *testing.T) {
	got := getChartName("onlyone")
	if got != invalidName {
		t.Fatalf("expected invalidName for single segment path, got %s", got)
	}
}

func TestGetChartNameTwoSegments(t *testing.T) {
	got := getChartName("charts/train")
	if got != invalidName {
		t.Fatalf("expected invalidName for two-segment path, got %s", got)
	}
}

func TestGetChartPathNested(t *testing.T) {
	p, err := getChartPath("charts/stable/mychart/templates/deployment.yaml")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := "charts/stable/mychart/Chart.yaml"
	if p != expected {
		t.Fatalf("expected %s, got %s", expected, p)
	}
}

func TestGetChartPathDirectChart(t *testing.T) {
	p, err := getChartPath("charts/stable/mychart/Chart.yaml")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := "charts/stable/mychart/Chart.yaml"
	if p != expected {
		t.Fatalf("expected %s, got %s", expected, p)
	}
}

func TestGetVersionClean(t *testing.T) {
	sample := "name: app\nversion: 2.5.0\n"
	v, err := getVersion(sample)
	if err != nil {
		t.Fatalf("getVersion error: %v", err)
	}
	if v != "2.5.0" {
		t.Fatalf("expected 2.5.0, got %s", v)
	}
}

func TestGetVersionWithSpaces(t *testing.T) {
	sample := "name: app\nversion:   3.1.0  \n"
	v, err := getVersion(sample)
	if err != nil {
		t.Fatalf("getVersion error: %v", err)
	}
	if v != "3.1.0" {
		t.Fatalf("expected 3.1.0, got %s", v)
	}
}

func TestGetVersionMultipleDashes(t *testing.T) {
	sample := "name: app\nversion: 1.0.-2-rc\n"
	v, err := getVersion(sample)
	if err != nil {
		t.Fatalf("getVersion error: %v", err)
	}
	if v != "1.0.2rc" {
		t.Fatalf("expected 1.0.2rc, got %s", v)
	}
}

func TestGetChartTrainThreeSegments(t *testing.T) {
	got := getChartTrain("charts/enterprise/app/Chart.yaml")
	if got != "enterprise" {
		t.Fatalf("expected enterprise, got %s", got)
	}
}

func TestGetChartTrainEmpty(t *testing.T) {
	got := getChartTrain("x")
	if got != "" {
		t.Fatalf("expected empty, got %s", got)
	}
}

func TestStatusIncSkippedCount(t *testing.T) {
	s := &status{
		processedCount: 5,
		skippedCount:   0,
		mu:             &sync.RWMutex{},
	}
	s.incSkippedCount()
	if s.skippedCount != 1 {
		t.Fatalf("expected skippedCount 1, got %d", s.skippedCount)
	}
	if s.processedCount != 4 {
		t.Fatalf("expected processedCount 4, got %d", s.processedCount)
	}
}

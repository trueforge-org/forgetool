package changelog

import (
	"testing"
)

func TestChartGetAppName(t *testing.T) {
	got := chartGetAppName("charts/stable/mychart/Chart.yaml")
	if got != "mychart" {
		t.Fatalf("expected mychart, got %s", got)
	}

	got = chartGetAppName("charts/stable/mychart/templates/deploy.yaml")
	if got != "mychart" {
		t.Fatalf("expected mychart, got %s", got)
	}

	got = chartGetAppName("short")
	if got != invalidName {
		t.Fatalf("expected invalidName for single-segment path, got %s", got)
	}
}

func TestChartGetAppTrain(t *testing.T) {
	got := chartGetAppTrain("charts/stable/mychart/Chart.yaml")
	if got != "stable" {
		t.Fatalf("expected stable, got %s", got)
	}

	got = chartGetAppTrain("x")
	if got != "" {
		t.Fatalf("expected empty for single-segment, got %s", got)
	}
}

func TestChartGetAppPath(t *testing.T) {
	p, err := chartGetAppPath("charts/stable/mychart/templates/deployment.yaml")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := "charts/stable/mychart/Chart.yaml"
	if p != expected {
		t.Fatalf("expected %s, got %s", expected, p)
	}

	_, err = chartGetAppPath(".")
	if err == nil {
		t.Fatalf("expected error for short path")
	}
}

func TestChartGetVersion(t *testing.T) {
	sample := "name: app\nversion: 2.5.0\n"
	v, err := chartGetVersion(sample)
	if err != nil {
		t.Fatalf("chartGetVersion error: %v", err)
	}
	if v != "2.5.0" {
		t.Fatalf("expected 2.5.0, got %s", v)
	}

	_, err = chartGetVersion("no version here")
	if err == nil {
		t.Fatalf("expected error when version missing")
	}
}

func TestChartIsPreferredFile(t *testing.T) {
	if !chartIsPreferredFile("charts/stable/app/Chart.yaml") {
		t.Fatalf("expected Chart.yaml to be preferred")
	}
	if chartIsPreferredFile("charts/stable/app/values.yaml") {
		t.Fatalf("expected values.yaml to not be preferred")
	}
}

func TestChartRenderOutputPath(t *testing.T) {
	got := chartRenderOutputPath("/output", "stable", "mychart")
	if got != "/output/stable/mychart" {
		t.Fatalf("expected /output/stable/mychart, got %s", got)
	}
}

func TestChartParseActiveApp(t *testing.T) {
	a := &ActiveApps{items: make(map[string]ActiveApp), mu: newRWMutex()}

	if err := chartParseActiveApp(a, "charts/stable/mychart/Chart.yaml"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !a.isActiveApp("mychart") {
		t.Fatalf("expected mychart to be active")
	}
	app := a.items["mychart"]
	if app.Train != "stable" {
		t.Fatalf("expected train stable, got %s", app.Train)
	}

	// Duplicate
	if err := chartParseActiveApp(a, "charts/stable/mychart/Chart.yaml"); err != nil {
		t.Fatalf("expected no error on duplicate, got %v", err)
	}
	if len(a.items) != 1 {
		t.Fatalf("expected 1 item after duplicate, got %d", len(a.items))
	}

	// Too short
	if err := chartParseActiveApp(a, "a/Chart.yaml"); err == nil {
		t.Fatalf("expected error for short path")
	}
}

package changelog

import (
	"path/filepath"
	"testing"
)

func TestGetAppName(t *testing.T) {
	got := getAppName("charts/trainA/mychart/Chart.yaml")
	if got != "mychart" {
		t.Fatalf("expected mychart got %s", got)
	}

	// too short
	got = getAppName("charts/only")
	if got == "" || got == invalidName {
		// expected invalidName for short paths
	} else {
		t.Fatalf("expected invalidName for short path, got %s", got)
	}
}

func TestGetAppPath(t *testing.T) {
	// when path already contains Chart.yaml
	p := filepath.Join("charts", "train", "app", "Chart.yaml")
	got, err := getAppPath(filepath.Dir(p))
	if err != nil {
		t.Fatalf("getAppPath error: %v", err)
	}
	if got != p {
		t.Fatalf("expected %s got %s", p, got)
	}

	// too short
	_, err = getAppPath(".")
	if err == nil {
		t.Fatalf("expected error for short path")
	}
}

func TestGetVersion(t *testing.T) {
	sample := "name: foo\nversion: 1.2.-3\nother: x"
	v, err := getVersion(sample)
	if err != nil {
		t.Fatalf("getVersion error: %v", err)
	}
	if v != "1.2.3" {
		t.Fatalf("expected 1.2.3 got %s", v)
	}

	// missing version
	_, err = getVersion("no version here")
	if err == nil {
		t.Fatalf("expected error when version missing")
	}
}

func TestGetAppTrain(t *testing.T) {
	if got := getAppTrain("charts/trainX/chartY/Chart.yaml"); got != "trainX" {
		t.Fatalf("expected trainX got %s", got)
	}
	if got := getAppTrain("short"); got != "" {
		t.Fatalf("expected empty train for short path, got %s", got)
	}
}

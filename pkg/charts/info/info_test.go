package info

import (
	"testing"
	"time"
)

func TestNewInfoAndPrint(t *testing.T) {
	d := NewInfo()
	if d == nil {
		t.Fatalf("NewInfo returned nil")
	}
	if d.GoVersion == "" {
		t.Fatalf("expected GoVersion to be populated")
	}

	custom := &Data{
		GoVersion: "go1.x",
		GoOS:      "darwin",
		GoArch:    "arm64",
		GoC:       true,
		GitCommit: "abc123",
		GitDirty:  false,
	}
	custom.Print()
}

func TestApplyBuildSetting_AllSupportedKeys(t *testing.T) {
	d := &Data{}

	applyBuildSetting(d, "GOARCH", "arm64")
	applyBuildSetting(d, "GOOS", "darwin")
	applyBuildSetting(d, "CGO_ENABLED", "1")
	applyBuildSetting(d, "vcs.revision", "abc123")
	applyBuildSetting(d, "vcs.time", "2026-01-02T03:04:05Z")
	applyBuildSetting(d, "vcs.modified", "true")

	if d.GoArch != "arm64" || d.GoOS != "darwin" || !d.GoC || d.GitCommit != "abc123" || !d.GitDirty {
		t.Fatalf("unexpected data after applyBuildSetting: %+v", d)
	}
	if d.GitDate.IsZero() {
		t.Fatalf("expected GitDate to be parsed")
	}
}

func TestApplyBuildSetting_InvalidTime(t *testing.T) {
	d := &Data{GitDate: time.Now()}
	applyBuildSetting(d, "vcs.time", "not-a-time")
	if !d.GitDate.IsZero() {
		t.Fatalf("expected invalid vcs.time to produce zero GitDate")
	}
}

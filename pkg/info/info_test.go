package info

import (
	"runtime/debug"
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

func TestNewInfo_ParsesVCSSettings(t *testing.T) {
	oldRead := readBuildInfoFn
	t.Cleanup(func() {
		readBuildInfoFn = oldRead
	})

	readBuildInfoFn = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			GoVersion: "go1.test",
			Settings: []debug.BuildSetting{
				{Key: "GOARCH", Value: "amd64"},
				{Key: "GOOS", Value: "linux"},
				{Key: "CGO_ENABLED", Value: "1"},
				{Key: "vcs.revision", Value: "deadbeef"},
				{Key: "vcs.time", Value: "2025-01-02T03:04:05Z"},
				{Key: "vcs.modified", Value: "true"},
			},
		}, true
	}

	d := NewInfo()
	if d.GoArch != "amd64" || d.GoOS != "linux" {
		t.Fatalf("unexpected go platform values: arch=%q os=%q", d.GoArch, d.GoOS)
	}
	if !d.GoC {
		t.Fatal("expected GoC true from CGO_ENABLED=1")
	}
	if d.GitCommit != "deadbeef" {
		t.Fatalf("unexpected git commit: %q", d.GitCommit)
	}
	if !d.GitDirty {
		t.Fatal("expected GitDirty true")
	}
	wantDate, _ := time.Parse(time.RFC3339, "2025-01-02T03:04:05Z")
	if !d.GitDate.Equal(wantDate) {
		t.Fatalf("unexpected git date: got=%s want=%s", d.GitDate, wantDate)
	}
}

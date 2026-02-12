package info

import "testing"

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

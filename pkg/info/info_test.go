package info

import (
	"testing"
)

func TestSmokeInfo(t *testing.T) {}

func TestNewInfo(t *testing.T) {
	data := NewInfo()
	
	if data == nil {
		t.Fatal("NewInfo() returned nil")
	}
	
	// Verify that at least some fields are populated
	if data.GoVersion == "" {
		t.Error("GoVersion should not be empty")
	}
	
	if data.GoOS == "" {
		t.Error("GoOS should not be empty")
	}
	
	if data.GoArch == "" {
		t.Error("GoArch should not be empty")
	}
}

func TestData_Print(t *testing.T) {
	data := &Data{
		GoVersion: "go1.21.0",
		GoOS:      "linux",
		GoArch:    "amd64",
		GoC:       true,
		GitCommit: "abc123",
		GitDirty:  false,
	}
	
	// This test just ensures Print() doesn't panic
	// Output goes to logger which we're not capturing
	data.Print()
}

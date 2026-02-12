package info

import (
	"testing"
	"time"
)

func TestSmokeChartsInfo(t *testing.T) {}

func TestNewInfo(t *testing.T) {
	data := NewInfo()

	if data == nil {
		t.Fatal("NewInfo() returned nil")
	}

	// Check that at least GoVersion is populated (always available)
	if data.GoVersion == "" {
		t.Error("GoVersion should not be empty")
	}

	// GoArch and GoOS are also typically available
	if data.GoArch == "" {
		t.Error("GoArch should not be empty")
	}

	if data.GoOS == "" {
		t.Error("GoOS should not be empty")
	}

	// GitCommit and GitDate may be empty if not built with vcs info
	// GitDirty and GoC have valid zero values (false)
}

func TestData_Print(t *testing.T) {
	tests := []struct {
		name string
		data *Data
	}{
		{
			name: "Complete data",
			data: &Data{
				GoVersion: "go1.21.0",
				GoArch:    "amd64",
				GoOS:      "linux",
				GoC:       true,
				GitCommit: "abc123",
				GitDate:   time.Now(),
				GitDirty:  false,
			},
		},
		{
			name: "Minimal data",
			data: &Data{
				GoVersion: "go1.20.0",
				GoArch:    "arm64",
				GoOS:      "darwin",
			},
		},
		{
			name: "Empty git info",
			data: &Data{
				GoVersion: "go1.19.0",
				GoArch:    "386",
				GoOS:      "windows",
				GoC:       false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test just ensures Print() doesn't panic
			// The output goes to the logger which we're not capturing
			tt.data.Print()
		})
	}
}

func TestNewInfo_BuildInfoParsing(t *testing.T) {
	// Test that NewInfo can handle different build info scenarios
	data := NewInfo()

	// Verify struct is properly initialized
	if data == nil {
		t.Fatal("NewInfo() should never return nil")
	}

	// Check that boolean fields have valid values (even if false)
	_ = data.GoC
	_ = data.GitDirty

	// Check that time field is initialized (may be zero value)
	_ = data.GitDate
}


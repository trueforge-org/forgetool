package changelog

import (
	"sync"
	"testing"
)

func newRWMutex() *sync.RWMutex { return &sync.RWMutex{} }

func TestConfigureForAppTypeChart(t *testing.T) {
	if err := configureForAppType(AppTypeChart); err != nil {
		t.Fatalf("configureForAppType(chart) error: %v", err)
	}
	if activeAppsManifestFile != chartManifestFile {
		t.Fatalf("expected manifest file %s, got %s", chartManifestFile, activeAppsManifestFile)
	}
}

func TestConfigureForAppTypeContainer(t *testing.T) {
	if err := configureForAppType(AppTypeContainer); err != nil {
		t.Fatalf("configureForAppType(container) error: %v", err)
	}
	if activeAppsManifestFile != containerManifestFile {
		t.Fatalf("expected manifest file %s, got %s", containerManifestFile, activeAppsManifestFile)
	}
	// Reset to chart defaults
	_ = configureForAppType(AppTypeChart)
}

func TestConfigureForAppTypeEmpty(t *testing.T) {
	if err := configureForAppType(""); err != nil {
		t.Fatalf("configureForAppType(\"\") should default to chart, got error: %v", err)
	}
	if activeAppsManifestFile != chartManifestFile {
		t.Fatalf("expected chart manifest file for empty type")
	}
}

func TestConfigureForAppTypeUnknown(t *testing.T) {
	if err := configureForAppType("unknown"); err == nil {
		t.Fatalf("expected error for unknown app type")
	}
	// Reset to chart defaults
	_ = configureForAppType(AppTypeChart)
}

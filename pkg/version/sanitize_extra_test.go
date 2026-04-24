package version

import (
	"errors"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
)

func TestSanitizeAt_SemverParseFailureFallback(t *testing.T) {
	old := semverNewVersion
	t.Cleanup(func() { semverNewVersion = old })
	semverNewVersion = func(string) (*semver.Version, error) { return nil, errors.New("forced") }

	now := time.Date(2025, 6, 7, 0, 0, 0, 0, time.UTC)
	got := SanitizeAt("1.2.3", now)
	if got.IsValidSemver {
		t.Fatalf("expected IsValidSemver=false, got true")
	}
	if got.Semantic != "2025.6.7" {
		t.Fatalf("expected calver fallback, got %q", got.Semantic)
	}
	if got.Raw != "1.2.3" {
		t.Fatalf("expected raw passthrough, got %q", got.Raw)
	}
}

func TestSanitize_PublicWrapperRespectsRealSemver(t *testing.T) {
	got := Sanitize("v1.0.0-rc1")
	if !got.IsValidSemver {
		t.Fatalf("expected valid semver")
	}
	if got.Raw != "1.0.0" {
		t.Fatalf("expected stripped raw, got %q", got.Raw)
	}
}

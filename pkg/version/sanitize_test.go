package version

import (
	"testing"
	"time"
)

// Fixed reference time for deterministic calver output.
var refTime = time.Date(2025, 4, 17, 12, 0, 0, 0, time.UTC)

func TestSanitizeAt_TableDriven(t *testing.T) {
	tests := []struct {
		name          string
		upstream      string
		wantValid     bool
		wantSemantic  string
		wantRaw       string
		wantUpstream  string
	}{
		{
			name:         "full-semver",
			upstream:     "1.2.3",
			wantValid:    true,
			wantSemantic: "1.2.3",
			wantRaw:      "1.2.3",
			wantUpstream: "1.2.3",
		},
		{
			name:         "v-prefix",
			upstream:     "v1.2.3",
			wantValid:    true,
			wantSemantic: "1.2.3",
			wantRaw:      "1.2.3",
			wantUpstream: "v1.2.3",
		},
		{
			name:         "pre-release-suffix",
			upstream:     "v2.0.1-rc1",
			wantValid:    true,
			wantSemantic: "2.0.1",
			wantRaw:      "2.0.1",
			wantUpstream: "v2.0.1-rc1",
		},
		{
			name:         "major-only",
			upstream:     "5",
			wantValid:    true,
			wantSemantic: "5.0.0",
			wantRaw:      "5",
			wantUpstream: "5",
		},
		{
			name:         "major-minor-only",
			upstream:     "3.7",
			wantValid:    true,
			wantSemantic: "3.7.0",
			wantRaw:      "3.7",
			wantUpstream: "3.7",
		},
		{
			name:         "v-major-minor",
			upstream:     "v10.1",
			wantValid:    true,
			wantSemantic: "10.1.0",
			wantRaw:      "10.1",
			wantUpstream: "v10.1",
		},
		{
			name:         "pre-release-with-build",
			upstream:     "1.0.0-beta.1",
			wantValid:    true,
			wantSemantic: "1.0.0",
			wantRaw:      "1.0.0",
			wantUpstream: "1.0.0-beta.1",
		},
		{
			name:         "non-semver-string",
			upstream:     "latest",
			wantValid:    false,
			wantSemantic: "2025.4.17",
			wantRaw:      "latest",
			wantUpstream: "latest",
		},
		{
			name:         "non-semver-hash",
			upstream:     "abc123",
			wantValid:    false,
			wantSemantic: "2025.4.17",
			wantRaw:      "abc123",
			wantUpstream: "abc123",
		},
		{
			name:         "empty-string",
			upstream:     "",
			wantValid:    false,
			wantSemantic: "2025.4.17",
			wantRaw:      "",
			wantUpstream: "",
		},
		{
			name:         "zero-version",
			upstream:     "0.0.0",
			wantValid:    true,
			wantSemantic: "0.0.0",
			wantRaw:      "0.0.0",
			wantUpstream: "0.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeAt(tt.upstream, refTime)

			if got.IsValidSemver != tt.wantValid {
				t.Errorf("IsValidSemver = %v, want %v", got.IsValidSemver, tt.wantValid)
			}
			if got.Semantic != tt.wantSemantic {
				t.Errorf("Semantic = %q, want %q", got.Semantic, tt.wantSemantic)
			}
			if got.Raw != tt.wantRaw {
				t.Errorf("Raw = %q, want %q", got.Raw, tt.wantRaw)
			}
			if got.Upstream != tt.wantUpstream {
				t.Errorf("Upstream = %q, want %q", got.Upstream, tt.wantUpstream)
			}
		})
	}
}

func TestSanitize_UsesCurrentTime(t *testing.T) {
	info := Sanitize("not-a-version")
	if info.IsValidSemver {
		t.Fatal("expected non-semver input to report IsValidSemver=false")
	}
	if info.Semantic == "" {
		t.Fatal("expected Semantic calver to be non-empty")
	}
}

func TestCalver(t *testing.T) {
	got := calver(time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC))
	if got != "2024.1.5" {
		t.Fatalf("calver() = %q, want %q", got, "2024.1.5")
	}
}

func TestSanitizeRaw(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"v1.2.3", "1.2.3"},
		{"1.2.3-rc1", "1.2.3"},
		{"v1.0.0-beta.1", "1.0.0"},
		{"hello", "hello"},
		{"v", ""},
		{"", ""},
	}

	for _, tt := range tests {
		got := sanitizeRaw(tt.in)
		if got != tt.want {
			t.Errorf("sanitizeRaw(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

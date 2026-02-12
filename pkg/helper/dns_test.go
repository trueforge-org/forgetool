package helper

import (
	"testing"
)

func TestCheckDNSResolution(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		want   bool
	}{
		{
			name:   "Valid domain - google.com",
			domain: "google.com",
			want:   true,
		},
		{
			name:   "Valid domain - github.com",
			domain: "github.com",
			want:   true,
		},
		{
			name:   "Invalid domain - nonexistent",
			domain: "this-domain-absolutely-does-not-exist-12345.com",
			want:   false,
		},
		{
			name:   "Empty domain",
			domain: "",
			want:   false,
		},
		{
			name:   "Invalid format",
			domain: "invalid..domain",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if testing.Short() && tt.want == true {
				t.Skip("Skipping network test in short mode")
			}

			got := checkDNSResolution(tt.domain)
			if got != tt.want {
				t.Errorf("checkDNSResolution(%q) = %v, want %v", tt.domain, got, tt.want)
			}
		})
	}
}

// Note: checkAllDomains and CheckReqDomains call os.Exit(1) on failure,
// making them difficult to unit test without refactoring.
// These would require either:
// 1. Refactoring to return errors instead of calling os.Exit
// 2. Using a test helper that captures os.Exit calls
// 3. Integration tests that run in separate processes
//
// For now, we test the core DNS resolution logic via checkDNSResolution.
// The checkAllDomains function is a simple loop wrapper that can be visually inspected.
func TestCheckReqDomains_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test will only pass if all required domains resolve
	// If it fails, the function will call os.Exit(1)
	// In a real environment, this should pass as these are common domains
	// However, we can't easily test the failure case without refactoring

	// Note: We skip actual execution because CheckReqDomains calls os.Exit
	// which would terminate the test process
	t.Skip("CheckReqDomains calls os.Exit(1) on failure - needs refactoring for proper testing")
}

package helper

import (
	"testing"
)

func TestIsPathIgnored_MatchesPrefix(t *testing.T) {
	prefixes := []string{"repos", "clusters"}
	tests := []struct {
		path    string
		ignored bool
	}{
		{"repos/charts", true},
		{"clusters/main", true},
		{"other/path", false},
		{"", false},
		{"repos", true},
	}
	for _, tt := range tests {
		got := isPathIgnored(tt.path, prefixes)
		if got != tt.ignored {
			t.Errorf("isPathIgnored(%q, ...) = %v, want %v", tt.path, got, tt.ignored)
		}
	}
}

func TestIsPathIgnored_EmptyPrefixes(t *testing.T) {
	if isPathIgnored("anything", nil) {
		t.Fatal("expected false for nil prefixes")
	}
	if isPathIgnored("anything", []string{}) {
		t.Fatal("expected false for empty prefixes")
	}
}

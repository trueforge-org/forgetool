package helper

import (
	"testing"
)

func TestWalkCharts_InvalidMode(t *testing.T) {
	// calling WalkCharts with an invalid mode (out of defined range) should return an error
	err := WalkCharts([]string{"./nonexistent"}, func(p, b string) error { return nil }, "", WalkMode(999))
	if err == nil {
		t.Fatalf("expected error for invalid WalkMode, got nil")
	}
}

package image

import "testing"

func TestClean(t *testing.T) {
	if err := Clean("v1.2.3"); err != nil {
		t.Fatalf("Clean should succeed for valid tag: %v", err)
	}
}

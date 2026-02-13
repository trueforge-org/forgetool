//go:build darwin && arm64
// +build darwin,arm64

package embed

import "testing"

func TestStaticFiles_NotEmpty(t *testing.T) {
	entries, err := StaticFiles.ReadDir(".")
	if err != nil {
		t.Fatalf("expected embedded fs root to be readable: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one embedded entry")
	}
}

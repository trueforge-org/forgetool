package helper

import (
	"testing"
)

func TestIsPathIgnored(t *testing.T) {
	prefixes := []string{"repositories", "forgetool/repositories"}
	if !isPathIgnored("repositories/foo", prefixes) {
		t.Fatalf("expected repositories/foo to be ignored by prefix")
	}
	if !isPathIgnored("forgetool/repositories/bar", prefixes) {
		t.Fatalf("expected forgetool/repositories/bar to be ignored by prefix")
	}
	if isPathIgnored("other/path", prefixes) {
		t.Fatalf("did not expect other/path to be ignored")
	}
}

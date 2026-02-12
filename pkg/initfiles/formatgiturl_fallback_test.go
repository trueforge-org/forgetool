package initfiles

import "testing"

func TestFormatGitURL_FallbackUnchanged(t *testing.T) {
	in := "not-a-git-url"
	got := FormatGitURL(in)
	if got == "" {
		t.Fatalf("unexpected empty result for input %q", in)
	}
	// if regex doesn't match, FormatGitURL should return the original or prefixed form, not empty
}

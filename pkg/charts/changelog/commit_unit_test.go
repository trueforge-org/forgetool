package changelog

import (
	"sync"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestGetCommitKindAllTypes(t *testing.T) {
	tests := []struct {
		message  string
		expected string
	}{
		{"chore(deps): update deps", "chore"},
		{"feat(core): add feature", "feat"},
		{"fix(ui): fix bug", "fix"},
		{"docs(readme): update readme", "docs"},
		{"refactor(core): cleanup", ""},
		{"", ""},
		{"just a message", ""},
		{"feat: no scope", ""},
	}

	for _, tt := range tests {
		c := &object.Commit{Message: tt.message}
		got := getCommitKind(c)
		if got != tt.expected {
			t.Errorf("getCommitKind(%q) = %q, want %q", tt.message, got, tt.expected)
		}
	}
}

func TestGetCommitMessageMultiline(t *testing.T) {
	c := &object.Commit{Message: "  feat(x): trimmed  \nSecond line\nThird line"}
	got := getCommitMessage(c)
	if got != "feat(x): trimmed" {
		t.Fatalf("expected trimmed first line, got %q", got)
	}
}

func TestGetCommitMessageSingleLine(t *testing.T) {
	c := &object.Commit{Message: "fix(y): single line"}
	got := getCommitMessage(c)
	if got != "fix(y): single line" {
		t.Fatalf("expected single line, got %q", got)
	}
}

func TestIsValidCommitWithGoodMessageAndSkipEnabled(t *testing.T) {
	currentStatus = status{mu: &sync.RWMutex{}}
	skipCommitsWithBadMessage = true
	c := &object.Commit{
		Message:      "feat(scope): valid message",
		ParentHashes: []plumbing.Hash{{}},
	}
	if !isValidCommit(c) {
		t.Fatalf("expected valid commit with good conventional message to pass even when skip enabled")
	}
}

func TestIsValidCommitEmptyParentHashes(t *testing.T) {
	currentStatus = status{mu: &sync.RWMutex{}}
	skipCommitsWithBadMessage = false
	c := &object.Commit{
		Message:      "feat(x): valid",
		ParentHashes: []plumbing.Hash{},
	}
	if isValidCommit(c) {
		t.Fatalf("expected commit with empty ParentHashes to be invalid")
	}
}

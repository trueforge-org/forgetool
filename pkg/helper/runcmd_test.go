package helper

import (
	"bytes"
	"strings"
	"testing"
)

func TestFilteredWriter(t *testing.T) {
	var buf bytes.Buffer
	fw := &filteredWriter{writer: &buf, filters: []string{"skip", "secret"}}

	input := "line1\nskip this line\nline2 with secret\nline3\n"
	if _, err := fw.Write([]byte(input)); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	got := buf.String()
	if strings.Contains(got, "skip this line") {
		t.Fatalf("filtered output contains skipped line")
	}
	if strings.Contains(got, "secret") {
		t.Fatalf("filtered output contains secret line")
	}
	if !strings.Contains(got, "line1") || !strings.Contains(got, "line3") {
		t.Fatalf("expected remaining lines present, got: %q", got)
	}
}

func TestRunCommandSuccessAndError(t *testing.T) {
	// success case
	out, err := RunCommand([]string{"sh", "-c", "printf 'hello\\nworld\\n'"}, true)
	if err != nil {
		t.Fatalf("expected nil error, got: %v (out=%q)", err, out)
	}
	if out != "hello\nworld\n" {
		t.Fatalf("unexpected output: %q", out)
	}

	// error case: write to stderr and exit non-zero
	out2, err2 := RunCommand([]string{"sh", "-c", "printf 'err-message' 1>&2; exit 3"}, true)
	if err2 == nil {
		t.Fatalf("expected error from failing command, got nil (out=%q)", out2)
	}
	if !strings.Contains(out2, "err-message") {
		t.Fatalf("expected stderr in output, got: %q", out2)
	}
}

package helper

import (
	"os"
	"strings"
	"testing"
)

func TestReplaceInFile(t *testing.T) {
	f, err := os.CreateTemp("", "replacein_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	content := "hello OLD world\nOLD again"
	f.WriteString(content)
	f.Close()

	if err := ReplaceInFile(f.Name(), "OLD", "NEW"); err != nil {
		t.Fatalf("ReplaceInFile error: %v", err)
	}
	data, _ := os.ReadFile(f.Name())
	if !strings.Contains(string(data), "NEW") {
		t.Fatalf("replacement not applied: %s", string(data))
	}
}

func TestReplaceContentBetweenLines(t *testing.T) {
	target, err := os.CreateTemp("", "target_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(target.Name())

	// target file with markers
	target.WriteString("line1\nFROM_MARKER\nold content\nTILL_MARKER\nline2\n")
	target.Close()

	// source file containing replacement (including markers which should be stripped)
	src, err := os.CreateTemp("", "source_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(src.Name())
	src.WriteString("FROM_MARKER\nnew content\nTILL_MARKER\n")
	src.Close()

	if err := ReplaceContentBetweenLines(target.Name(), src.Name(), "FROM_MARKER", "TILL_MARKER"); err != nil {
		t.Fatalf("ReplaceContentBetweenLines error: %v", err)
	}

	data, _ := os.ReadFile(target.Name())
	s := string(data)
	if !strings.Contains(s, "new content") {
		t.Fatalf("replacement content missing: %s", s)
	}
	if strings.Contains(s, "old content") {
		t.Fatalf("old content still present: %s", s)
	}
}

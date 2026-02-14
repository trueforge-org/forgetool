package helper

import (
	"bufio"
	"errors"
	"strings"
	"testing"
)

func TestGetYesOrNo_NonInteractiveReturnsDefault(t *testing.T) {
	oldNonInteractive := NonInteractive
	NonInteractive = true
	t.Cleanup(func() { NonInteractive = oldNonInteractive })

	if !GetYesOrNo("ignored", true) {
		t.Fatal("expected default true in non-interactive mode")
	}
	if GetYesOrNo("ignored", false) {
		t.Fatal("expected default false in non-interactive mode")
	}
}

func TestGetYesOrNo_ReadErrorThenValid(t *testing.T) {
	oldNonInteractive := NonInteractive
	oldNewReaderFn := promptNewReaderFn
	oldReadStringFn := promptReadStringFn
	t.Cleanup(func() {
		NonInteractive = oldNonInteractive
		promptNewReaderFn = oldNewReaderFn
		promptReadStringFn = oldReadStringFn
	})

	NonInteractive = false
	promptNewReaderFn = func() *bufio.Reader { return bufio.NewReader(strings.NewReader("unused\n")) }

	calls := 0
	promptReadStringFn = func(reader *bufio.Reader) (string, error) {
		calls++
		if calls == 1 {
			return "", errors.New("read failed")
		}
		return "y\n", nil
	}

	if !GetYesOrNo("continue? ", false) {
		t.Fatal("expected true after retry with y")
	}
}

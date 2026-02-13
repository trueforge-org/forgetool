package helper

import (
	"os"
	"testing"
)

func TestGetYesOrNoYes(t *testing.T) {
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe create failed: %v", err)
	}
	os.Stdin = r
	if _, err := w.Write([]byte("yes\n")); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	w.Close()

	if !GetYesOrNo("continue? ", false) {
		t.Fatalf("expected true for 'yes' input")
	}
}

func TestGetYesOrNoYESUpperCase(t *testing.T) {
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe create failed: %v", err)
	}
	os.Stdin = r
	if _, err := w.Write([]byte("YES\n")); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	w.Close()

	if !GetYesOrNo("continue? ", false) {
		t.Fatalf("expected true for 'YES' input")
	}
}

func TestGetYesOrNoNo(t *testing.T) {
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe create failed: %v", err)
	}
	os.Stdin = r
	if _, err := w.Write([]byte("no\n")); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	w.Close()

	if GetYesOrNo("continue? ", false) {
		t.Fatalf("expected false for 'no' input")
	}
}

func TestGetYesOrNoNUpperCase(t *testing.T) {
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe create failed: %v", err)
	}
	os.Stdin = r
	if _, err := w.Write([]byte("N\n")); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	w.Close()

	if GetYesOrNo("continue? ", false) {
		t.Fatalf("expected false for 'N' input")
	}
}

func TestGetYesOrNoInvalidThenValid(t *testing.T) {
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe create failed: %v", err)
	}
	os.Stdin = r
	if _, err := w.Write([]byte("invalid\nyes\n")); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	w.Close()

	if !GetYesOrNo("continue? ", false) {
		t.Fatalf("expected true after invalid input followed by 'yes'")
	}
}

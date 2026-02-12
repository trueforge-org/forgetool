package sops

import (
	"errors"
	"fmt"
	"testing"
)

func TestMacFailureError_Error(t *testing.T) {
	tests := []struct {
		name     string
		origErr  error
		wantSub  string
	}{
		{"simple error", errors.New("something went wrong"), "MAC failure: something went wrong"},
		{"wrapped error", fmt.Errorf("outer: %w", errors.New("inner")), "MAC failure: outer: inner"},
		{"empty error", errors.New(""), "MAC failure: "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &MacFailureError{OriginalError: tt.origErr}
			got := e.Error()
			if got != tt.wantSub {
				t.Fatalf("MacFailureError.Error() = %q, want %q", got, tt.wantSub)
			}
		})
	}
}

func TestIsMacFailure(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"matching", errors.New("MAC verification failed: bad checksum"), true},
		{"exact", errors.New("MAC verification failed"), true},
		{"non-matching", errors.New("decryption key not found"), false},
		{"partial mismatch", errors.New("MAC mismatch"), false},
		{"empty", errors.New(""), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isMacFailure(tt.err); got != tt.want {
				t.Fatalf("isMacFailure(%q) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestDecryptDataWithRetry_NonMacError(t *testing.T) {
	// Invalid data should produce a non-MAC decryption error that passes through.
	_, err := decryptDataWithRetry([]byte("not-encrypted"), "yaml")
	if err == nil {
		t.Fatalf("expected error decrypting invalid data, got nil")
	}
}

func TestDecryptData_InvalidData(t *testing.T) {
	tests := []struct {
		name   string
		data   []byte
		format string
	}{
		{"invalid yaml", []byte("not: encrypted: data"), "yaml"},
		{"invalid json", []byte(`{"key":"value"}`), "json"},
		{"garbage binary", []byte{0x00, 0x01, 0x02}, "binary"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decryptData(tt.data, tt.format)
			if err == nil {
				t.Fatalf("expected error for invalid data, got nil")
			}
		})
	}
}

func TestDecryptDataIgnoringMac_InvalidData(t *testing.T) {
	tests := []struct {
		name   string
		data   []byte
		format string
	}{
		{"invalid yaml", []byte("plain: text"), "yaml"},
		{"invalid json", []byte(`{"a":1}`), "json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decryptDataIgnoringMac(tt.data, tt.format)
			if err == nil {
				t.Fatalf("expected error for non-sops data, got nil")
			}
		})
	}
}

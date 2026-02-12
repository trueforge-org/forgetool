package sops

import (
	"errors"
	"testing"
)

func TestDecryptDataWithRetry_NonMacNonNilError(t *testing.T) {
	// Non-encrypted data should return a non-MAC error through the else branch
	_, err := decryptDataWithRetry([]byte("plain text data"), "yaml")
	if err == nil {
		t.Fatal("expected error for plain text data, got nil")
	}
	// Verify it's not a MacFailureError
	var macErr *MacFailureError
	if errors.As(err, &macErr) {
		t.Fatal("error should not be a MacFailureError for plain text")
	}
}

func TestDecryptData_NonEncrypted(t *testing.T) {
	_, err := decryptData([]byte("not encrypted yaml"), "yaml")
	if err == nil {
		t.Fatal("expected error for non-encrypted data")
	}
}

func TestDecryptData_EmptyData(t *testing.T) {
	_, err := decryptData([]byte(""), "yaml")
	if err == nil {
		t.Fatal("expected error for empty data")
	}
}

func TestDecryptData_BinaryFormat(t *testing.T) {
	_, err := decryptData([]byte{0x00, 0x01, 0x02, 0x03}, "binary")
	if err == nil {
		t.Fatal("expected error for random binary data")
	}
}

func TestDecryptDataIgnoringMac_EmptyData(t *testing.T) {
	_, err := decryptDataIgnoringMac([]byte(""), "yaml")
	if err == nil {
		t.Fatal("expected error for empty data")
	}
}

func TestDecryptDataIgnoringMac_BinaryData(t *testing.T) {
	_, err := decryptDataIgnoringMac([]byte{0xFF, 0xFE}, "binary")
	if err == nil {
		t.Fatal("expected error for invalid binary data")
	}
}

func TestDecryptDataWithRetry_ValidButNotEncrypted(t *testing.T) {
	// Valid YAML but not SOPS encrypted - should fail
	data := []byte("key: value\nother: data\n")
	_, err := decryptDataWithRetry(data, "yaml")
	if err == nil {
		t.Fatal("expected error for valid but non-encrypted YAML")
	}
}

func TestIsMacFailure_EmptyError(t *testing.T) {
	if isMacFailure(errors.New("")) {
		t.Fatal("expected false for empty error")
	}
}

func TestIsMacFailure_PartialMatch(t *testing.T) {
	if isMacFailure(errors.New("MAC but not verification")) {
		t.Fatal("expected false for partial match")
	}
}

func TestMacFailureError_Implements(t *testing.T) {
	var err error = &MacFailureError{OriginalError: errors.New("test")}
	if err.Error() != "MAC failure: test" {
		t.Fatalf("unexpected error string: %s", err.Error())
	}
}

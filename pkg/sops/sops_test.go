package sops

import "testing"

func TestSmokeSops(t *testing.T) {}

func TestNewCypherDecryptInvalid(t *testing.T) {
	c := NewCypher()
	if c == nil {
		t.Fatalf("NewCypher returned nil")
	}
	if _, err := c.Decrypt([]byte("not-encrypted-data"), "yaml"); err == nil {
		t.Fatalf("expected error decrypting invalid data, got nil")
	}
}

func TestNewMasterKeyHandlesInvalid(t *testing.T) {
	// Ensure NewMasterKey does not panic on invalid input
	_ = NewMasterKey("not-a-valid-recipient")
}

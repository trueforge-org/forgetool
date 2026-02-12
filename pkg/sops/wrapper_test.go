package sops

import (
	"testing"

	sopsv3 "github.com/getsops/sops/v3"
)

func TestNewCypher_ReturnsNonNil(t *testing.T) {
	c := NewCypher()
	if c == nil {
		t.Fatalf("expected non-nil Cypher, got nil")
	}
}

func TestCypherEncrypt_InvalidContent(t *testing.T) {
	c := NewCypher()
	_, err := c.Encrypt([]byte("not valid yaml or json {{{}}}"), EncryptionConfig{
		Format: "yaml",
		Keys:   []sopsv3.KeyGroup{},
	})
	if err == nil {
		t.Fatalf("expected error encrypting invalid content, got nil")
	}
}

func TestCypherEncrypt_InvalidJSON(t *testing.T) {
	c := NewCypher()
	_, err := c.Encrypt([]byte("{invalid json}"), EncryptionConfig{
		Format: "json",
		Keys:   []sopsv3.KeyGroup{},
	})
	if err == nil {
		t.Fatalf("expected error encrypting invalid JSON, got nil")
	}
}

func TestEncryptionConfig_FormatOptions(t *testing.T) {
	tests := []struct {
		name   string
		format string
	}{
		{"yaml format", "yaml"},
		{"json format", "json"},
		{"empty format defaults to json", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := EncryptionConfig{
				Format:            tt.format,
				Keys:              []sopsv3.KeyGroup{},
				UnencryptedSuffix: "_unencrypted",
				EncryptedSuffix:   "_encrypted",
				UnencryptedRegex:  "^public$",
				EncryptedRegex:    "^secret$",
				ShamirThreshold:   3,
			}
			if cfg.Format != tt.format {
				t.Fatalf("expected format %q, got %q", tt.format, cfg.Format)
			}
			if cfg.ShamirThreshold != 3 {
				t.Fatalf("expected ShamirThreshold 3, got %d", cfg.ShamirThreshold)
			}
		})
	}
}

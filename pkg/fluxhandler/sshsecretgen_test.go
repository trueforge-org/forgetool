package fluxhandler

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"os"
	"strings"
	"testing"
)

func TestIndentYaml(t *testing.T) {
	in := "a\nb\n"
	out := indentYaml(in, "  ")
	if !strings.HasPrefix(out, "  a\n") {
		t.Fatalf("indentYaml did not indent properly: %q", out)
	}
}

func TestReplacePlaceholder(t *testing.T) {
	f, err := os.CreateTemp("", "replace_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("hello PLACEHOLDER world")
	f.Close()

	if err := ReplacePlaceholder(f.Name(), "PLACEHOLDER", "there"); err != nil {
		t.Fatalf("ReplacePlaceholder error: %v", err)
	}
	data, _ := os.ReadFile(f.Name())
	if !strings.Contains(string(data), "hello there world") {
		t.Fatalf("replacement failed: %s", string(data))
	}
}

func TestPemAndPublicKeyHelpers(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	pemb, err := pemBlockForKey(priv)
	if err != nil {
		t.Fatalf("pemBlockForKey error: %v", err)
	}
	if !strings.Contains(string(pemb), "EC PRIVATE KEY") {
		t.Fatalf("unexpected PEM output: %s", string(pemb))
	}

	pubssh, err := publicKeyToOpenSSH(&priv.PublicKey)
	if err != nil {
		t.Fatalf("publicKeyToOpenSSH error: %v", err)
	}
	if !strings.Contains(pubssh, "ssh-ecdsa") && !strings.Contains(pubssh, "ecdsa-sha2") {
		t.Fatalf("unexpected public key format: %s", pubssh)
	}
}

func TestKnownHostsAndBase64Stubs(t *testing.T) {
	g := getKnownHostsEntry("github.com")
	if g != getGithubKnownHostsEntry() {
		t.Fatalf("github known hosts mismatch")
	}

	other := getKnownHostsEntry("example.com")
	if !strings.HasPrefix(other, "example.com ") {
		t.Fatalf("generated known hosts missing prefix: %s", other)
	}

	// encode/decode stubs currently mirror input
	enc := encodeToBase64([]byte("abc"))
	if enc != "abc" {
		t.Fatalf("encodeToBase64 unexpected: %s", enc)
	}
	dec, err := decodeBase64("xyz")
	if err != nil {
		t.Fatalf("decodeBase64 returned error: %v", err)
	}
	if string(dec) != "xyz" {
		t.Fatalf("decodeBase64 unexpected: %s", string(dec))
	}
}

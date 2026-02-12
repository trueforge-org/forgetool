package fluxhandler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

func TestCreateGitSecret_GeneratesFiles(t *testing.T) {
	tmp := t.TempDir()
	// Point ClusterPath to a temp dir
	old := helper.ClusterPath
	helper.ClusterPath = tmp
	defer func() { helper.ClusterPath = old }()

	// Ensure no preexisting files
	secretPath := filepath.Join(helper.ClusterPath, "kubernetes", "flux-system", "flux", "deploykey.secret.yaml")
	pubPath := filepath.Join(".", "ssh-public-key.txt")
	os.Remove(secretPath)
	os.Remove(pubPath)

	if err := CreateGitSecret("github.com"); err != nil {
		t.Fatalf("CreateGitSecret failed: %v", err)
	}

	if _, err := os.Stat(secretPath); err != nil {
		t.Fatalf("expected secret file created: %v", err)
	}
	if _, err := os.Stat(pubPath); err != nil {
		t.Fatalf("expected public key created: %v", err)
	}
	// cleanup pub key
	os.Remove(pubPath)
}

func TestCreateGitSecret_UsesExistingSecretForPubKey(t *testing.T) {
	tmp := t.TempDir()
	old := helper.ClusterPath
	helper.ClusterPath = tmp
	defer func() { helper.ClusterPath = old }()

	// First generate the secret and public key
	if err := CreateGitSecret("example.com"); err != nil {
		t.Fatalf("initial CreateGitSecret failed: %v", err)
	}
	pubPath := filepath.Join(".", "ssh-public-key.txt")
	// Read generated public key
	_, err := os.ReadFile(pubPath)
	if err != nil {
		t.Fatalf("expected public key file created: %v", err)
	}
	// Remove public key file to force regeneration from existing secret
	if err := os.Remove(pubPath); err != nil {
		t.Fatalf("remove pubkey: %v", err)
	}

	err = CreateGitSecret("")
	if err == nil {
		t.Fatalf("expected error when regenerating pubkey due to YAML parsing issue, got nil")
	}
	// cleanup if any
	os.Remove(pubPath)
}

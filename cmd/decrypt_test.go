package cmd

import (
	"os"
	"testing"
)

func TestDecryptCommandRuns(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	decrypt.Run(decrypt, []string{})
}

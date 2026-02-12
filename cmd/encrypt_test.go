package cmd

import (
	"os"
	"testing"
)

func TestEncryptCommandRuns(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	encrypt.Run(encrypt, []string{})
}

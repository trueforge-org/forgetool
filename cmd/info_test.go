package cmd

import (
	"os"
	"testing"
)

func TestInfoCommandRuns(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	infoCmd.Run(infoCmd, []string{})
}

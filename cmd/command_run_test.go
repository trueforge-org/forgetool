package cmd

import (
	"os"
	"os/exec"
	"testing"
)

func TestHelperCheckcryptProcess(t *testing.T) {
	if os.Getenv("GO_WANT_CHECKCRYPT_HELPER") != "1" {
		return
	}
	checkcrypt.Run(checkcrypt, []string{})
	os.Exit(0)
}

func TestSimpleCommandRuns(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	decrypt.Run(decrypt, []string{})
	encrypt.Run(encrypt, []string{})
	infoCmd.Run(infoCmd, []string{})
}

func TestCheckcryptRunExitsNonZero(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperCheckcryptProcess")
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "GO_WANT_CHECKCRYPT_HELPER=1")
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected non-zero exit from checkcrypt command")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() == 0 {
		t.Fatalf("expected non-zero exit code, got: %v", err)
	}
}

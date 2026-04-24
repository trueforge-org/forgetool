package helper

import (
	"os"
	"os/exec"
	"testing"
)

func TestHelperProcessCheckAllDomainsFatal(t *testing.T) {
	if os.Getenv("GO_WANT_DOMAINS_FATAL") != "1" {
		return
	}
	checkAllDomains([]string{"invalid.domain.that.does.not.exist.example"}, false)
	os.Exit(0)
}

func TestCheckAllDomains_FatalSubprocess(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcessCheckAllDomainsFatal")
	cmd.Env = append(os.Environ(), "GO_WANT_DOMAINS_FATAL=1")
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected non-zero exit from log.Fatal")
	}
	if _, ok := err.(*exec.ExitError); !ok {
		t.Fatalf("unexpected error type: %T", err)
	}
}

package helper

import (
	"os"
	"os/exec"
	"testing"
)

func TestCheckDNSResolution_Localhost(t *testing.T) {
	if !checkDNSResolution("localhost") {
		t.Fatalf("expected localhost to resolve")
	}
}

func TestCheckDNSResolution_InvalidDomain(t *testing.T) {
	if checkDNSResolution("invalid.domain.that.does.not.exist.example") {
		t.Fatalf("expected invalid domain not to resolve")
	}
}

func TestCheckAllDomains_ValidDomains(t *testing.T) {
	// Should not exit or panic with resolvable domains.
	checkAllDomains([]string{"localhost"}, false)
	checkAllDomains([]string{"localhost"}, true)
}

func TestHelperProcessCheckReqDomains(t *testing.T) {
	if os.Getenv("GO_WANT_CHECK_REQ_DOMAINS") != "1" {
		return
	}
	CheckReqDomains()
	os.Exit(0)
}

func TestCheckReqDomains_Subprocess(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcessCheckReqDomains")
	cmd.Env = append(os.Environ(), "GO_WANT_CHECK_REQ_DOMAINS=1")
	err := cmd.Run()
	// CheckReqDomains may exit 0 (all resolve) or 1 (some fail);
	// we only verify it does not crash unexpectedly.
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("unexpected error type: %T: %v", err, err)
		}
		if exitErr.ExitCode() != 1 {
			t.Fatalf("expected exit code 0 or 1, got %d", exitErr.ExitCode())
		}
	}
}

package nodestatus

import (
	"errors"
	"strings"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

func TestHasUnknownAuthorityErrorCoverage(t *testing.T) {
	if hasUnknownAuthorityError("certificate signed by unknown authority", nil) != true {
		t.Fatal("expected true when output contains unknown-authority")
	}
	if hasUnknownAuthorityError("", errors.New("certificate signed by unknown authority")) != true {
		t.Fatal("expected true when error contains unknown-authority")
	}
	if hasUnknownAuthorityError("ok", nil) != false {
		t.Fatal("expected false when neither output nor error contain unknown-authority")
	}
}

func TestFormatStatusErrorCoverage(t *testing.T) {
	err := formatStatusError("payload", nil)
	if err == nil || err.Error() != "status: payload" {
		t.Fatalf("unexpected nil-error formatting: %v", err)
	}

	wrapped := formatStatusError("payload", errors.New("boom"))
	if wrapped == nil || !strings.Contains(wrapped.Error(), "status: payload error: boom") {
		t.Fatalf("unexpected wrapped error formatting: %v", wrapped)
	}
}

func TestCheckReadyStatusInsecureRetryErrorBranches(t *testing.T) {
	oldCache := helper.CacheDir
	oldClusterPath := helper.ClusterPath
	helper.CacheDir = t.TempDir()
	helper.ClusterPath = t.TempDir()
	t.Cleanup(func() {
		helper.CacheDir = oldCache
		helper.ClusterPath = oldClusterPath
	})

	writeFakeTalos(t, func(args []string) (string, error) {
		for _, arg := range args {
			if arg == "--insecure" {
				return "retry output", errors.New("retry failed")
			}
		}
		return "certificate signed by unknown authority", errors.New("certificate signed by unknown authority")
	})

	status, err := CheckReadyStatus("10.0.0.2", false)
	if err == nil {
		t.Fatal("expected insecure retry failure error")
	}
	if status != "ERROR" {
		t.Fatalf("expected ERROR status, got %q", status)
	}

	status, err = CheckReadyStatus("10.0.0.2", true)
	if err == nil {
		t.Fatal("expected insecure retry failure error in silent mode")
	}
	if status != "ERROR" {
		t.Fatalf("expected ERROR status in silent mode, got %q", status)
	}
}

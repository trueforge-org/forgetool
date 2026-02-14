package nodestatus

import (
	"errors"
	"strings"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

func TestEvaluateHealthStatus_NullAndEmptyBranches(t *testing.T) {
	if err := evaluateHealthStatus("n1", "", "", true); err == nil {
		t.Fatal("expected error for empty status output")
	}
	if err := evaluateHealthStatus("n1", "", "null", true); err == nil {
		t.Fatal("expected error for null status output")
	}
}

func TestValidateReadyStatus_SilentErrorBranch(t *testing.T) {
	oldCache := helper.CacheDir
	oldClusterPath := helper.ClusterPath
	helper.CacheDir = t.TempDir()
	helper.ClusterPath = t.TempDir()
	t.Cleanup(func() {
		helper.CacheDir = oldCache
		helper.ClusterPath = oldClusterPath
	})

	writeFakeTalos(t, func(args []string) (string, error) {
		if strings.Contains(strings.Join(args, " "), "jsonpath={.spec.status.ready}") {
			return "", errors.New("ready check failed")
		}
		return "running", nil
	})

	if err := validateReadyStatus("10.0.0.10", "running", true); err == nil {
		t.Fatal("expected validateReadyStatus error in silent branch")
	}
}

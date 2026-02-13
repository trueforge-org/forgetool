package nodestatus

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

func TestWaitForHealth_PeriodicCheckPath(t *testing.T) {
	oldCache := helper.CacheDir
	oldClusterPath := helper.ClusterPath
	oldInterval := waitForHealthCheckInterval
	oldMaxDuration := waitForHealthMaxDuration
	helper.CacheDir = t.TempDir()
	helper.ClusterPath = t.TempDir()
	waitForHealthCheckInterval = 2 * time.Millisecond
	waitForHealthMaxDuration = 200 * time.Millisecond
	t.Cleanup(func() {
		helper.CacheDir = oldCache
		helper.ClusterPath = oldClusterPath
		waitForHealthCheckInterval = oldInterval
		waitForHealthMaxDuration = oldMaxDuration
	})

	callCount := 0
	writeFakeTalos(t, func(args []string) (string, error) {
		argsStr := strings.Join(args, " ")
		if strings.Contains(argsStr, "jsonpath={.spec.stage}") {
			callCount++
			if callCount <= 2 {
				return "", fmt.Errorf("temporary failure")
			}
			return "running", nil
		}
		return "unknown", nil
	})

	got, err := WaitForHealth("10.0.0.7", []string{"running"})
	if err != nil {
		t.Fatalf("expected eventual success via periodic checks, got: %v", err)
	}
	if got != "running" {
		t.Fatalf("expected running status, got %q", got)
	}
	if callCount < 3 {
		t.Fatalf("expected periodic checks to run after initial failure, calls=%d", callCount)
	}
}

func TestWaitForHealth_TimeoutPath(t *testing.T) {
	oldCache := helper.CacheDir
	oldClusterPath := helper.ClusterPath
	oldInterval := waitForHealthCheckInterval
	oldMaxDuration := waitForHealthMaxDuration
	helper.CacheDir = t.TempDir()
	helper.ClusterPath = t.TempDir()
	waitForHealthCheckInterval = 2 * time.Millisecond
	waitForHealthMaxDuration = 20 * time.Millisecond
	t.Cleanup(func() {
		helper.CacheDir = oldCache
		helper.ClusterPath = oldClusterPath
		waitForHealthCheckInterval = oldInterval
		waitForHealthMaxDuration = oldMaxDuration
	})

	writeFakeTalos(t, func(args []string) (string, error) {
		return "", fmt.Errorf("still booting")
	})

	got, err := WaitForHealth("10.0.0.8", []string{"running"})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if got != "ERROR" {
		t.Fatalf("expected ERROR on timeout, got %q", got)
	}
	if !strings.Contains(err.Error(), "timeout waiting for Node to boot") {
		t.Fatalf("unexpected timeout error: %v", err)
	}
}

func TestCheckStatusAndNeedBootstrap_InsecureRetryError(t *testing.T) {
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
				return "retry output", fmt.Errorf("retry failed")
			}
		}
		return "certificate signed by unknown authority", fmt.Errorf("certificate signed by unknown authority")
	})

	status, err := CheckStatus("10.0.0.9")
	if err == nil {
		t.Fatal("expected CheckStatus insecure retry error, got nil")
	}
	if status != "ERROR" {
		t.Fatalf("expected ERROR status, got %q", status)
	}

	need, err := CheckNeedBootstrap("10.0.0.9")
	if err == nil {
		t.Fatal("expected CheckNeedBootstrap insecure retry error, got nil")
	}
	if need {
		t.Fatal("expected bootstrap to be false on insecure retry error")
	}
}

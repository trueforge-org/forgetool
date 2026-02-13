package nodestatus

import (
	"fmt"
	"strings"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

func TestCheckStatusError(t *testing.T) {
	oldCache := helper.CacheDir
	oldClusterPath := helper.ClusterPath
	helper.CacheDir = t.TempDir()
	helper.ClusterPath = t.TempDir()
	t.Cleanup(func() {
		helper.CacheDir = oldCache
		helper.ClusterPath = oldClusterPath
	})

	writeFakeTalos(t, func(args []string) (string, error) {
		return "", fmt.Errorf("error occurred")
	})

	status, err := CheckStatus("10.0.0.1")
	if err == nil {
		t.Fatal("expected error from CheckStatus, got nil")
	}
	if status != "ERROR" {
		t.Fatalf("expected ERROR status, got %q", status)
	}
}

func TestCheckStatusCertFallback(t *testing.T) {
	oldCache := helper.CacheDir
	oldClusterPath := helper.ClusterPath
	helper.CacheDir = t.TempDir()
	helper.ClusterPath = t.TempDir()
	t.Cleanup(func() {
		helper.CacheDir = oldCache
		helper.ClusterPath = oldClusterPath
	})

	writeFakeTalos(t, func(args []string) (string, error) {
		// Check if --insecure flag is present
		for _, arg := range args {
			if arg == "--insecure" {
				return "running", nil
			}
		}
		return "certificate signed by unknown authority", fmt.Errorf("certificate signed by unknown authority")
	})

	status, err := CheckStatus("10.0.0.1")
	if err != nil {
		t.Fatalf("expected fallback to succeed, got error: %v", err)
	}
	if !strings.Contains(status, "running") {
		t.Fatalf("expected running status from fallback, got %q", status)
	}
}

func TestCheckReadyStatusError(t *testing.T) {
	oldCache := helper.CacheDir
	oldClusterPath := helper.ClusterPath
	helper.CacheDir = t.TempDir()
	helper.ClusterPath = t.TempDir()
	t.Cleanup(func() {
		helper.CacheDir = oldCache
		helper.ClusterPath = oldClusterPath
	})

	writeFakeTalos(t, func(args []string) (string, error) {
		return "", fmt.Errorf("fail")
	})

	status, err := CheckReadyStatus("10.0.0.1", false)
	if err == nil {
		t.Fatal("expected error from CheckReadyStatus, got nil")
	}
	if status != "ERROR" {
		t.Fatalf("expected ERROR, got %q", status)
	}
}

func TestCheckReadyStatusSilent(t *testing.T) {
	oldCache := helper.CacheDir
	oldClusterPath := helper.ClusterPath
	helper.CacheDir = t.TempDir()
	helper.ClusterPath = t.TempDir()
	t.Cleanup(func() {
		helper.CacheDir = oldCache
		helper.ClusterPath = oldClusterPath
	})

	writeFakeTalos(t, func(args []string) (string, error) {
		return "", fmt.Errorf("fail")
	})

	_, err := CheckReadyStatus("10.0.0.1", true)
	if err == nil {
		t.Fatal("expected error from silent CheckReadyStatus, got nil")
	}
}

func TestCheckReadyStatusNotReady(t *testing.T) {
	oldCache := helper.CacheDir
	oldClusterPath := helper.ClusterPath
	helper.CacheDir = t.TempDir()
	helper.ClusterPath = t.TempDir()
	t.Cleanup(func() {
		helper.CacheDir = oldCache
		helper.ClusterPath = oldClusterPath
	})

	writeFakeTalos(t, func(args []string) (string, error) {
		argsStr := strings.Join(args, " ")
		if strings.Contains(argsStr, "jsonpath={.spec.status.ready}") {
			return "false", nil
		}
		return "running", nil
	})

	status, err := CheckReadyStatus("10.0.0.1", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(status, "false") {
		t.Fatalf("expected false in status, got %q", status)
	}
}

func TestCheckReadyStatusCertFallback(t *testing.T) {
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
				return "true", nil
			}
		}
		return "certificate signed by unknown authority", fmt.Errorf("certificate signed by unknown authority")
	})

	status, err := CheckReadyStatus("10.0.0.1", false)
	if err != nil {
		t.Fatalf("expected fallback success, got error: %v", err)
	}
	if !strings.Contains(status, "true") {
		t.Fatalf("expected true from fallback, got %q", status)
	}
}

func TestCheckReadyStatusCertOutputWithoutError(t *testing.T) {
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
				return "true", nil
			}
		}
		return "certificate signed by unknown authority", nil
	})

	status, err := CheckReadyStatus("10.0.0.1", false)
	if err != nil {
		t.Fatalf("expected fallback success, got error: %v", err)
	}
	if !strings.Contains(status, "true") {
		t.Fatalf("expected true from fallback, got %q", status)
	}
}

func TestCheckNeedBootstrapNoBootstrap(t *testing.T) {
	oldCache := helper.CacheDir
	oldClusterPath := helper.ClusterPath
	helper.CacheDir = t.TempDir()
	helper.ClusterPath = t.TempDir()
	t.Cleanup(func() {
		helper.CacheDir = oldCache
		helper.ClusterPath = oldClusterPath
	})

	// Success with "running" status - no bootstrap needed
	writeFakeTalos(t, func(args []string) (string, error) {
		return "running", nil
	})

	need, err := CheckNeedBootstrap("10.0.0.1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if need {
		t.Fatal("expected bootstrap not needed for running status")
	}
}

func TestCheckNeedBootstrapCertErrorNoMaintenance(t *testing.T) {
	oldCache := helper.CacheDir
	oldClusterPath := helper.ClusterPath
	helper.CacheDir = t.TempDir()
	helper.ClusterPath = t.TempDir()
	t.Cleanup(func() {
		helper.CacheDir = oldCache
		helper.ClusterPath = oldClusterPath
	})

	// Certificate error fallback yields "running" (not maintenance)
	writeFakeTalos(t, func(args []string) (string, error) {
		// Check if --insecure flag is present
		for _, arg := range args {
			if arg == "--insecure" {
				return "running", nil
			}
		}
		return "certificate signed by unknown authority", fmt.Errorf("certificate signed by unknown authority")
	})

	need, err := CheckNeedBootstrap("10.0.0.1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if need {
		t.Fatal("expected bootstrap not needed when running (not maintenance)")
	}
}

func TestCheckNeedBootstrapCertOutputWithoutError(t *testing.T) {
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
				return "maintenance", nil
			}
		}
		return "certificate signed by unknown authority", nil
	})

	need, err := CheckNeedBootstrap("10.0.0.1")
	if err != nil {
		t.Fatalf("expected fallback success, got error: %v", err)
	}
	if !need {
		t.Fatal("expected bootstrap needed for maintenance status from insecure fallback")
	}
}

func TestCheckNeedBootstrapNonCertError(t *testing.T) {
	oldCache := helper.CacheDir
	oldClusterPath := helper.ClusterPath
	helper.CacheDir = t.TempDir()
	helper.ClusterPath = t.TempDir()
	t.Cleanup(func() {
		helper.CacheDir = oldCache
		helper.ClusterPath = oldClusterPath
	})

	// Non-certificate error
	writeFakeTalos(t, func(args []string) (string, error) {
		return "", fmt.Errorf("connection refused")
	})

	_, err := CheckNeedBootstrap("10.0.0.1")
	if err == nil {
		t.Fatal("expected error for non-cert failure, got nil")
	}
}

func TestCheckStatusCertOutputWithoutError(t *testing.T) {
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
				return "running", nil
			}
		}
		return "certificate signed by unknown authority", nil
	})

	status, err := CheckStatus("10.0.0.1")
	if err != nil {
		t.Fatalf("expected fallback success, got error: %v", err)
	}
	if !strings.Contains(status, "running") {
		t.Fatalf("expected running status from fallback, got %q", status)
	}
}

func TestCheckHealthRunningWithReadyCheck(t *testing.T) {
	oldCache := helper.CacheDir
	oldClusterPath := helper.ClusterPath
	helper.CacheDir = t.TempDir()
	helper.ClusterPath = t.TempDir()
	t.Cleanup(func() {
		helper.CacheDir = oldCache
		helper.ClusterPath = oldClusterPath
	})

	// status="" + "running" output -> triggers CheckReadyStatus
	writeFakeTalos(t, func(args []string) (string, error) {
		argsStr := strings.Join(args, " ")
		if strings.Contains(argsStr, "jsonpath={.spec.stage}") {
			return "running", nil
		}
		if strings.Contains(argsStr, "jsonpath={.spec.status.ready}") {
			return "true", nil
		}
		return "unknown", nil
	})

	err := CheckHealth("10.0.0.1", "", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestCheckHealthRunningNotReady(t *testing.T) {
	oldCache := helper.CacheDir
	oldClusterPath := helper.ClusterPath
	helper.CacheDir = t.TempDir()
	helper.ClusterPath = t.TempDir()
	t.Cleanup(func() {
		helper.CacheDir = oldCache
		helper.ClusterPath = oldClusterPath
	})

	// status="" + "running" but ready check returns error
	writeFakeTalos(t, func(args []string) (string, error) {
		argsStr := strings.Join(args, " ")
		if strings.Contains(argsStr, "jsonpath={.spec.stage}") {
			return "running", nil
		}
		if strings.Contains(argsStr, "jsonpath={.spec.status.ready}") {
			return "", fmt.Errorf("fail")
		}
		return "unknown", nil
	})

	err := CheckHealth("10.0.0.1", "", false)
	if err == nil {
		t.Fatal("expected error when ready check fails, got nil")
	}
}

func TestCheckHealthSilentMode(t *testing.T) {
	oldCache := helper.CacheDir
	oldClusterPath := helper.ClusterPath
	helper.CacheDir = t.TempDir()
	helper.ClusterPath = t.TempDir()
	t.Cleanup(func() {
		helper.CacheDir = oldCache
		helper.ClusterPath = oldClusterPath
	})

	writeFakeTalos(t, func(args []string) (string, error) {
		return "running", nil
	})

	// Matching status, silent=true
	err := CheckHealth("10.0.0.1", "running", true)
	if err != nil {
		t.Fatalf("expected no error in silent mode, got: %v", err)
	}
}

func TestWaitForHealthMultipleStatuses(t *testing.T) {
	oldCache := helper.CacheDir
	oldClusterPath := helper.ClusterPath
	helper.CacheDir = t.TempDir()
	helper.ClusterPath = t.TempDir()
	t.Cleanup(func() {
		helper.CacheDir = oldCache
		helper.ClusterPath = oldClusterPath
	})

	writeFakeTalos(t, func(args []string) (string, error) {
		return "maintenance", nil
	})

	// "running" won't match, but "maintenance" will
	got, err := WaitForHealth("10.0.0.1", []string{"running", "maintenance"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got != "maintenance" {
		t.Fatalf("expected maintenance, got %q", got)
	}
}

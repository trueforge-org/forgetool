package nodestatus

import (
	"strings"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

func TestCheckHealthMatchingStatus(t *testing.T) {
	oldCache := helper.CacheDir
	oldClusterPath := helper.ClusterPath
	helper.CacheDir = t.TempDir()
	helper.ClusterPath = t.TempDir()
	t.Cleanup(func() {
		helper.CacheDir = oldCache
		helper.ClusterPath = oldClusterPath
	})

	script := "#!/bin/sh\necho running\n"
	writeFakeTalos(t, script)

	err := CheckHealth("10.0.0.1", "running", false)
	if err != nil {
		t.Fatalf("expected no error for matching status, got: %v", err)
	}
}

func TestCheckHealthMaintenanceMode(t *testing.T) {
	oldCache := helper.CacheDir
	oldClusterPath := helper.ClusterPath
	helper.CacheDir = t.TempDir()
	helper.ClusterPath = t.TempDir()
	t.Cleanup(func() {
		helper.CacheDir = oldCache
		helper.ClusterPath = oldClusterPath
	})

	script := "#!/bin/sh\necho maintenance\n"
	writeFakeTalos(t, script)

	err := CheckHealth("10.0.0.2", "", false)
	if err != nil {
		t.Fatalf("expected no error for maintenance mode with empty status, got: %v", err)
	}
}

func TestCheckHealthMismatchedStatus(t *testing.T) {
	oldCache := helper.CacheDir
	oldClusterPath := helper.ClusterPath
	helper.CacheDir = t.TempDir()
	helper.ClusterPath = t.TempDir()
	t.Cleanup(func() {
		helper.CacheDir = oldCache
		helper.ClusterPath = oldClusterPath
	})

	script := "#!/bin/sh\necho running\n"
	writeFakeTalos(t, script)

	err := CheckHealth("10.0.0.3", "maintenance", false)
	if err == nil {
		t.Fatal("expected error for mismatched status, got nil")
	}
	if !strings.Contains(err.Error(), "healthcheck failed") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestCheckHealthCheckStatusError(t *testing.T) {
	oldCache := helper.CacheDir
	oldClusterPath := helper.ClusterPath
	helper.CacheDir = t.TempDir()
	helper.ClusterPath = t.TempDir()
	t.Cleanup(func() {
		helper.CacheDir = oldCache
		helper.ClusterPath = oldClusterPath
	})

	script := "#!/bin/sh\necho 'connection refused' >&2\nexit 1\n"
	writeFakeTalos(t, script)

	// silent=true
	err := CheckHealth("10.0.0.4", "", true)
	if err == nil {
		t.Fatal("expected error when CheckStatus fails (silent=true), got nil")
	}
	if !strings.Contains(err.Error(), "healthcheck failed") {
		t.Fatalf("unexpected error message (silent=true): %v", err)
	}

	// silent=false
	err = CheckHealth("10.0.0.4", "", false)
	if err == nil {
		t.Fatal("expected error when CheckStatus fails (silent=false), got nil")
	}
	if !strings.Contains(err.Error(), "healthcheck failed") {
		t.Fatalf("unexpected error message (silent=false): %v", err)
	}
}

func TestWaitForHealthImmediateSuccess(t *testing.T) {
	oldCache := helper.CacheDir
	oldClusterPath := helper.ClusterPath
	helper.CacheDir = t.TempDir()
	helper.ClusterPath = t.TempDir()
	t.Cleanup(func() {
		helper.CacheDir = oldCache
		helper.ClusterPath = oldClusterPath
	})

	script := "#!/bin/sh\necho running\n"
	writeFakeTalos(t, script)

	got, err := WaitForHealth("10.0.0.5", []string{"running"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got != "running" {
		t.Fatalf("expected status 'running', got %q", got)
	}
}

func TestWaitForHealthEmptyStatusSlice(t *testing.T) {
	oldCache := helper.CacheDir
	oldClusterPath := helper.ClusterPath
	helper.CacheDir = t.TempDir()
	helper.ClusterPath = t.TempDir()
	t.Cleanup(func() {
		helper.CacheDir = oldCache
		helper.ClusterPath = oldClusterPath
	})

	// Empty status defaults to CheckHealth with status="" which accepts "maintenance"
	script := "#!/bin/sh\necho maintenance\n"
	writeFakeTalos(t, script)

	got, err := WaitForHealth("10.0.0.6", []string{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty status string (default), got %q", got)
	}
}

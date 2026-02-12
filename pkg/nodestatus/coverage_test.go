package nodestatus

import (
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

	script := "#!/bin/sh\necho 'error occurred' >&2\nexit 1\n"
	writeFakeTalos(t, script)

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

	script := "#!/bin/sh\nif echo \"$*\" | grep -q -- '--insecure'; then\n  echo running\n  exit 0\nfi\necho 'certificate signed by unknown authority' >&2\nexit 1\n"
	writeFakeTalos(t, script)

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

	script := "#!/bin/sh\necho 'fail' >&2\nexit 1\n"
	writeFakeTalos(t, script)

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

	script := "#!/bin/sh\necho 'fail' >&2\nexit 1\n"
	writeFakeTalos(t, script)

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

	script := "#!/bin/sh\ncase \"$*\" in\n  *\"jsonpath={.spec.status.ready}\"*) echo false ;;\n  *) echo running ;;\nesac\n"
	writeFakeTalos(t, script)

	status, err := CheckReadyStatus("10.0.0.1", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(status, "false") {
		t.Fatalf("expected false in status, got %q", status)
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
	script := "#!/bin/sh\necho running\n"
	writeFakeTalos(t, script)

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
	script := "#!/bin/sh\nif echo \"$*\" | grep -q -- '--insecure'; then\n  echo running\n  exit 0\nfi\necho 'certificate signed by unknown authority' >&2\nexit 1\n"
	writeFakeTalos(t, script)

	need, err := CheckNeedBootstrap("10.0.0.1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if need {
		t.Fatal("expected bootstrap not needed when running (not maintenance)")
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
	script := "#!/bin/sh\necho 'connection refused' >&2\nexit 1\n"
	writeFakeTalos(t, script)

	_, err := CheckNeedBootstrap("10.0.0.1")
	if err == nil {
		t.Fatal("expected error for non-cert failure, got nil")
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
	script := "#!/bin/sh\ncase \"$*\" in\n  *\"jsonpath={.spec.stage}\"*) echo running ;;\n  *\"jsonpath={.spec.status.ready}\"*) echo true ;;\n  *) echo unknown ;;\nesac\n"
	writeFakeTalos(t, script)

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
	script := "#!/bin/sh\ncase \"$*\" in\n  *\"jsonpath={.spec.stage}\"*) echo running ;;\n  *\"jsonpath={.spec.status.ready}\"*) echo 'fail' >&2; exit 1 ;;\nesac\n"
	writeFakeTalos(t, script)

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

	script := "#!/bin/sh\necho running\n"
	writeFakeTalos(t, script)

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

	script := "#!/bin/sh\necho maintenance\n"
	writeFakeTalos(t, script)

	// "running" won't match, but "maintenance" will
	got, err := WaitForHealth("10.0.0.1", []string{"running", "maintenance"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got != "maintenance" {
		t.Fatalf("expected maintenance, got %q", got)
	}
}

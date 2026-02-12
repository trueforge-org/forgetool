package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

func TestMainHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_MAIN_HELPER") != "1" {
		return
	}

	if p := os.Getenv("FORGETOOL_TEST_CACHE_DIR"); p != "" {
		helper.CacheDir = p
	}
	os.Args = []string{"forgetool"}
	main()
	os.Exit(0)
}

func TestMain_ExecutesToCacheStage(t *testing.T) {
	cacheDir := t.TempDir()

	cmd := exec.Command(os.Args[0], "-test.run=TestMainHelperProcess")
	cmd.Env = append(
		os.Environ(),
		"GO_WANT_MAIN_HELPER=1",
		"FORGETOOL_TEST_CACHE_DIR="+cacheDir,
		"DEBUG=1",
	)
	cmd.Dir = t.TempDir()
	_ = cmd.Run()

	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		t.Fatalf("failed to read cache dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("expected main to populate cache directory")
	}

	foundFile := false
	err = filepath.WalkDir(cacheDir, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			foundFile = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk cache dir failed: %v", err)
	}
	if !foundFile {
		t.Fatalf("expected cache to contain extracted files")
	}
}

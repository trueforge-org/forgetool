package embed

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

func TestGetTalosExec_NonEmptyWithCachePrefix(t *testing.T) {
	got := GetTalosExec()
	if got == "" {
		t.Fatal("expected non-empty path from GetTalosExec")
	}
	if !strings.HasPrefix(got, helper.CacheDir) {
		t.Fatalf("expected path to start with CacheDir %q, got %q", helper.CacheDir, got)
	}
}

func TestAllToCache_Idempotent(t *testing.T) {
	oldCache := helper.CacheDir
	helper.CacheDir = t.TempDir()
	t.Cleanup(func() { helper.CacheDir = oldCache })

	AllToCache()
	entries1, err := os.ReadDir(helper.CacheDir)
	if err != nil {
		t.Fatalf("first ReadDir failed: %v", err)
	}

	AllToCache()
	entries2, err := os.ReadDir(helper.CacheDir)
	if err != nil {
		t.Fatalf("second ReadDir failed: %v", err)
	}

	if len(entries1) != len(entries2) {
		t.Fatalf("expected same number of entries after second call, got %d vs %d", len(entries1), len(entries2))
	}
}

func TestAllToCache_SkipsExistingFiles(t *testing.T) {
	oldCache := helper.CacheDir
	helper.CacheDir = t.TempDir()
	t.Cleanup(func() { helper.CacheDir = oldCache })

	// First call to populate cache
	AllToCache()

	// Collect files and their sizes
	sizes := map[string]int64{}
	err := filepath.Walk(helper.CacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			sizes[path] = info.Size()
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk failed: %v", err)
	}

	// Second call should not error
	AllToCache()

	// Verify files still exist with same sizes
	for path, size := range sizes {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("file %s missing after second AllToCache: %v", path, err)
		}
		if info.Size() != size {
			t.Fatalf("file %s size changed: %d -> %d", path, size, info.Size())
		}
	}
}

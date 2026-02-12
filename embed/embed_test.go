package embed

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

func TestGetTalosExec_ReturnsPathForPlatform(t *testing.T) {
	got := GetTalosExec()
	if !strings.HasPrefix(got, helper.CacheDir) {
		t.Fatalf("expected path to start with cache dir %s, got %s", helper.CacheDir, got)
	}

	var expect string
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	switch goos {
	case "windows":
		if goarch == "amd64" {
			expect = "talosctl-windows-amd64.exe"
		} else {
			expect = "talosctl-windows-arm64.exe"
		}
	case "linux":
		if goarch == "amd64" {
			expect = "talosctl-linux-amd64"
		} else {
			expect = "talosctl-linux-arm64"
		}
	case "darwin":
		if goarch == "amd64" {
			expect = "talosctl-darwin-amd64"
		} else {
			expect = "talosctl-darwin-arm64"
		}
	case "freebsd":
		if goarch == "amd64" {
			expect = "talosctl-freebsd-amd64"
		} else {
			expect = "talosctl-freebsd-arm64"
		}
	default:
		t.Skipf("unsupported platform %s/%s for this test", goos, goarch)
	}

	if filepath.Base(got) != expect {
		t.Fatalf("expected exec basename %s, got %s", expect, filepath.Base(got))
	}
}

func TestFilesToCache_WritesGenericFiles(t *testing.T) {
	oldCache := helper.CacheDir
	helper.CacheDir = t.TempDir()
	t.Cleanup(func() {
		helper.CacheDir = oldCache
	})

	filesToCache(GenericFiles, "generic")

	foundFile := false
	err := filepath.WalkDir(helper.CacheDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			foundFile = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking cache failed: %v", err)
	}
	if !foundFile {
		t.Fatalf("expected generic embedded files to be written to cache")
	}

	entries, err := os.ReadDir(helper.CacheDir)
	if err != nil {
		t.Fatalf("reading cache dir failed: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("expected cache directory to contain entries")
	}
}

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

	AllToCache()

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

	AllToCache()

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

package deps

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/trueforge-org/forgetool/v4/pkg/helper"
)

func TestCopyDependency_Success(t *testing.T) {
	td := t.TempDir()
	oldHelmCache := helper.HelmCache
	helper.HelmCache = td
	defer func() { helper.HelmCache = oldHelmCache }()

	// Create a cached dependency file
	repoDir := "example.com/charts"
	cacheDir := filepath.Join(td, repoDir)
	os.MkdirAll(cacheDir, 0755)
	srcFile := filepath.Join(cacheDir, "myapp-1.0.0.tgz")
	os.WriteFile(srcFile, []byte("fake chart data"), 0644)

	chartFolder := filepath.Join(td, "charts", "stable", "myapp")
	os.MkdirAll(chartFolder, 0755)

	err := copyDependency(chartFolder, "example", repoDir, "myapp", "1.0.0")
	if err != nil {
		t.Fatalf("copyDependency failed: %v", err)
	}

	// Verify the file was copied
	destFile := filepath.Join(chartFolder, "charts", "myapp-1.0.0.tgz")
	if _, err := os.Stat(destFile); os.IsNotExist(err) {
		t.Fatal("expected dependency to be copied to charts/ subdirectory")
	}
}

func TestCopyDependency_SourceMissing(t *testing.T) {
	td := t.TempDir()
	oldHelmCache := helper.HelmCache
	helper.HelmCache = td
	defer func() { helper.HelmCache = oldHelmCache }()

	chartFolder := filepath.Join(td, "chart")
	os.MkdirAll(chartFolder, 0755)

	err := copyDependency(chartFolder, "example", "example.com/charts", "missing", "1.0.0")
	if err == nil {
		t.Fatal("expected error when source file is missing")
	}
}

func TestFetchIndexFile_OCI(t *testing.T) {
	td := t.TempDir()
	oldIndexCache := helper.IndexCache
	helper.IndexCache = td
	defer func() { helper.IndexCache = oldIndexCache }()

	// OCI repos should be skipped
	err := fetchIndexFile("oci://example.com/charts", "example.com/charts", "oci://example.com/charts")
	if err != nil {
		t.Fatalf("fetchIndexFile should skip OCI repos, got: %v", err)
	}
}

func TestFetchIndexFile_AlreadyCached(t *testing.T) {
	td := t.TempDir()
	oldIndexCache := helper.IndexCache
	helper.IndexCache = td
	defer func() { helper.IndexCache = oldIndexCache }()

	// Create a cached index file
	repoDir := "example.com/charts"
	indexDir := filepath.Join(td, repoDir)
	os.MkdirAll(indexDir, 0755)
	os.WriteFile(filepath.Join(indexDir, "index.yaml"), []byte("cached"), 0644)

	err := fetchIndexFile("https://example.com/charts", repoDir, "https://example.com/charts/index.yaml")
	if err != nil {
		t.Fatalf("fetchIndexFile should use cache, got: %v", err)
	}
}

func TestFetchDependency_AlreadyCached(t *testing.T) {
	td := t.TempDir()
	oldHelmCache := helper.HelmCache
	helper.HelmCache = td
	defer func() { helper.HelmCache = oldHelmCache }()

	// Create a cached dependency
	repoDir := "example.com/charts"
	cacheDir := filepath.Join(td, repoDir)
	os.MkdirAll(cacheDir, 0755)
	os.WriteFile(filepath.Join(cacheDir, "cached-1.0.0.tgz"), []byte("cached"), 0644)

	err := fetchDependency("https://example.com/charts", repoDir, "cached", "1.0.0", "https://example.com/charts/index.yaml")
	if err != nil {
		t.Fatalf("fetchDependency should use cache, got: %v", err)
	}
}

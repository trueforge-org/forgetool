package website

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// preservedIndexNames is the set of files at the root of a docs directory that
// must be preserved across a wipe-and-recreate cycle. The website repo
// maintains these by hand and they should not be regenerated.
var preservedIndexNames = []string{"index.mdx", "index.md"}

// PrepareContainerWebsite performs the global, one-time setup for the
// containerforge website docs tree. It is the Go equivalent of the bash
// preamble in containerforge's release.yaml workflow:
//
//   - ensures the public/img/hotlink-ok/{container-icons,container-icons-small,container-screenshots}
//     directories exist;
//   - preserves any maintained docs/containers/index.mdx (and index.md) file;
//   - removes the entire docs/containers directory so stale/renamed apps are
//     cleaned up;
//   - recreates the docs/containers directory and restores the preserved index
//     file.
//
// It must be called before iterating over apps with ProcessApp.
func PrepareContainerWebsite(opts ContainerOptions) error {
	opts.applyDefaults()
	root := opts.WebsiteDir
	if root == "" {
		return errors.New("website: WebsiteDir must be set")
	}

	for _, sub := range []string{
		filepath.Join("containerforge", "public", "img", "hotlink-ok", "container-icons"),
		filepath.Join("containerforge", "public", "img", "hotlink-ok", "container-icons-small"),
		filepath.Join("containerforge", "public", "img", "hotlink-ok", "container-screenshots"),
	} {
		if err := os.MkdirAll(filepath.Join(root, sub), 0o755); err != nil {
			return fmt.Errorf("create %s: %w", sub, err)
		}
	}

	docsDir := filepath.Join(root, "containerforge", "src", "content", "docs", "containers")
	return wipeAndRestore(docsDir)
}

// FinalizeContainerWebsite copies the contents of changelogsDir into the
// containerforge docs/containers directory. It is the Go equivalent of the
// "Copy changelogs to docs" step in containerforge's release.yaml workflow.
//
// When changelogsDir is empty or does not exist, the function is a no-op. When
// it exists but contains no subdirectories, no copy is performed (parity with
// the bash check `find ./changelogs -mindepth 1 -type d`).
func FinalizeContainerWebsite(opts ContainerOptions, changelogsDir string) error {
	opts.applyDefaults()
	dst := filepath.Join(opts.WebsiteDir, "containerforge", "src", "content", "docs", "containers")
	return copyChangelogs(changelogsDir, dst)
}

// PrepareChartWebsite performs the global, one-time setup for the truecharts
// website docs tree. It is the Go equivalent of the bash preamble in
// truecharts' charts-release.yaml workflow:
//
//   - ensures the public/img/hotlink-ok/{chart-icons,chart-icons-small} and
//     src/assets directories exist;
//   - preserves any maintained docs/charts/index.mdx file;
//   - removes every train subdirectory under docs/charts;
//   - recreates the docs/charts directory and restores the preserved index
//     file.
//
// chartsDir is the source charts directory (e.g. "charts") and is used to
// determine which train subdirectories to remove. Subdirectories that do not
// correspond to a source train are left untouched.
func PrepareChartWebsite(opts ChartOptions) error {
	opts.applyDefaults()
	root := opts.WebsiteDir
	if root == "" {
		return errors.New("website: WebsiteDir must be set")
	}

	for _, sub := range []string{
		filepath.Join("truecharts", "public", "img", "hotlink-ok", "chart-icons"),
		filepath.Join("truecharts", "public", "img", "hotlink-ok", "chart-icons-small"),
		filepath.Join("truecharts", "src", "assets"),
	} {
		if err := os.MkdirAll(filepath.Join(root, sub), 0o755); err != nil {
			return fmt.Errorf("create %s: %w", sub, err)
		}
	}

	docsDir := filepath.Join(root, "truecharts", "src", "content", "docs", "charts")
	return wipeAndRestore(docsDir)
}

// FinalizeChartWebsite copies the contents of changelogsDir into the
// truecharts docs/charts directory. Behaves like FinalizeContainerWebsite.
func FinalizeChartWebsite(opts ChartOptions, changelogsDir string) error {
	opts.applyDefaults()
	dst := filepath.Join(opts.WebsiteDir, "truecharts", "src", "content", "docs", "charts")
	return copyChangelogs(changelogsDir, dst)
}

// wipeAndRestore preserves index.mdx / index.md at the root of docsDir, removes
// docsDir entirely, recreates it, and restores the preserved files.
func wipeAndRestore(docsDir string) error {
	preserved := map[string][]byte{}
	for _, name := range preservedIndexNames {
		p := filepath.Join(docsDir, name)
		data, err := os.ReadFile(p)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("preserve %s: %w", name, err)
		}
		preserved[name] = data
	}

	if err := os.RemoveAll(docsDir); err != nil {
		return fmt.Errorf("remove %s: %w", docsDir, err)
	}
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		return fmt.Errorf("recreate %s: %w", docsDir, err)
	}

	for name, data := range preserved {
		if err := os.WriteFile(filepath.Join(docsDir, name), data, 0o644); err != nil {
			return fmt.Errorf("restore %s: %w", name, err)
		}
	}
	return nil
}

// copyChangelogs implements the bash:
//
//	mkdir -p $changelogsDir
//	if [ -n "$(find $changelogsDir -mindepth 1 -type d)" ]; then
//	  cp -r $changelogsDir/** $dst/
//	fi
//
// It is a no-op when changelogsDir is empty or contains no subdirectories.
func copyChangelogs(changelogsDir, dst string) error {
	if changelogsDir == "" {
		return nil
	}
	entries, err := os.ReadDir(changelogsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read %s: %w", changelogsDir, err)
	}
	hasDir := false
	for _, e := range entries {
		if e.IsDir() {
			hasDir = true
			break
		}
	}
	if !hasDir {
		return nil
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	return copyTreeContents(changelogsDir, dst)
}

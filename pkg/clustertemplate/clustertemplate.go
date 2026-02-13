package clustertemplate

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

// VersionEnv overrides the cluster-template release tag used for cache population.
const VersionEnv = "FORGETOOL_CLUSTER_TEMPLATE_VERSION"

// releaseTagPattern matches release tags like v1.2.3, 1.2.3, or v1.2.3-rc.1.
var releaseTagPattern = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+([-.+][A-Za-z0-9._-]+)?$`)

var latestReleaseURL = "https://api.github.com/repos/trueforge-org/cluster-template/releases/latest"
var releaseArchiveURL = "https://codeload.github.com/trueforge-org/cluster-template/tar.gz/refs/tags/%s"
var httpClient = &http.Client{Timeout: 10 * time.Second}
var resolveVersionHook = resolveVersion
var downloadReleaseHook = downloadRelease
var mkdirAllHook = os.MkdirAll
var openFileHook = func(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
	return os.OpenFile(name, flag, perm)
}
var copyHook = io.Copy
var closeHook = func(c io.Closer) error { return c.Close() }
var absPathHook = filepath.Abs
var isWithinCacheHook = isWithinCache

func ToCache() error {
	version, err := resolveVersionHook()
	if err != nil {
		return err
	}

	return downloadReleaseHook(version)
}

func resolveVersion() (string, error) {
	if version := strings.TrimSpace(os.Getenv(VersionEnv)); version != "" {
		if !releaseTagPattern.MatchString(version) {
			return "", fmt.Errorf("invalid %s value %q", VersionEnv, version)
		}
		// Keep a defensive traversal check even with pattern validation.
		if strings.Contains(version, "..") {
			return "", fmt.Errorf("invalid %s value %q", VersionEnv, version)
		}

		return version, nil
	}

	req, err := http.NewRequest(http.MethodGet, latestReleaseURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status for latest release: %s", resp.Status)
	}

	var latestRelease struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&latestRelease); err != nil {
		return "", err
	}
	if latestRelease.TagName == "" {
		return "", fmt.Errorf("latest release tag_name is empty")
	}

	return latestRelease.TagName, nil
}

func downloadRelease(version string) error {
	downloadURL := fmt.Sprintf(releaseArchiveURL, url.PathEscape(version))
	resp, err := httpClient.Get(downloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status for release archive %q: %s", version, resp.Status)
	}

	return extractArchive(resp.Body)
}

func extractArchive(reader io.Reader) error {
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	cacheDir, err := absPathHook(helper.CacheDir)
	if err != nil {
		return err
	}
	cacheDir = filepath.Clean(cacheDir)
	cachePrefix := cacheDir + string(os.PathSeparator)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		name := filepath.ToSlash(header.Name)
		nameParts := strings.SplitN(name, "/", 2)
		if len(nameParts) < 2 || nameParts[1] == "" {
			continue
		}
		// Prevent directory traversal by rejecting any archive entry
		// whose relative path contains ".." path elements.
		if strings.Contains(nameParts[1], "..") {
			return fmt.Errorf("invalid archive path (contains '..'): %s", header.Name)
		}

		targetPath := filepath.Join(cacheDir, filepath.FromSlash(nameParts[1]))
		cleanTargetPath := filepath.Clean(targetPath)
		if !isWithinCacheHook(cacheDir, cachePrefix, cleanTargetPath) {
			return fmt.Errorf("invalid archive path: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := mkdirAllHook(cleanTargetPath, os.ModePerm); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := mkdirAllHook(filepath.Dir(cleanTargetPath), os.ModePerm); err != nil {
				return err
			}

			file, err := openFileHook(cleanTargetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return err
			}

			if _, err := copyHook(file, tarReader); err != nil {
				_ = closeHook(file)
				return err
			}
			if err := closeHook(file); err != nil {
				return err
			}
		}
	}
}

func isWithinCache(cacheDir, cachePrefix, cleanTargetPath string) bool {
	return cleanTargetPath == cacheDir || strings.HasPrefix(cleanTargetPath, cachePrefix)
}

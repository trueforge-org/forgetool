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
	cacheDir := filepath.Clean(helper.CacheDir)
	cachePrefix := cacheDir + string(os.PathSeparator)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if header == nil {
			continue
		}

		name := filepath.ToSlash(header.Name)
		nameParts := strings.SplitN(name, "/", 2)
		if len(nameParts) < 2 || nameParts[1] == "" {
			continue
		}

		targetPath := filepath.Join(cacheDir, filepath.FromSlash(nameParts[1]))
		cleanTargetPath := filepath.Clean(targetPath)
		if cleanTargetPath != cacheDir && !strings.HasPrefix(cleanTargetPath, cachePrefix) {
			return fmt.Errorf("invalid archive path: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(cleanTargetPath, os.ModePerm); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(cleanTargetPath), os.ModePerm); err != nil {
				return err
			}

			file, err := os.OpenFile(cleanTargetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return err
			}

			if _, err := io.Copy(file, tarReader); err != nil {
				_ = file.Close()
				return err
			}
			if err := file.Close(); err != nil {
				return err
			}
		}
	}
}

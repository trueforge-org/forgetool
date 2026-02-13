package embed

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

	"github.com/rs/zerolog/log"
	"github.com/trueforge-org/forgetool/pkg/helper"
)

const clusterTemplateVersionEnv = "FORGETOOL_CLUSTER_TEMPLATE_VERSION"

var clusterTemplateVersionPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

var clusterTemplateLatestReleaseURL = "https://api.github.com/repos/trueforge-org/cluster-template/releases/latest"
var clusterTemplateReleaseArchiveURL = "https://codeload.github.com/trueforge-org/cluster-template/tar.gz/refs/tags/%s"
var clusterTemplateHTTPClient = &http.Client{Timeout: 10 * time.Second}
var resolveClusterTemplateVersionHook = resolveClusterTemplateVersion
var downloadClusterTemplateReleaseHook = downloadClusterTemplateRelease

func clusterTemplateToCache() {
	version, err := resolveClusterTemplateVersionHook()
	if err != nil {
		log.Info().Msgf("Error resolving cluster-template release version: %v", err)
		filesToCache(GenericFiles, "generic")
		return
	}

	if err := downloadClusterTemplateReleaseHook(version); err != nil {
		log.Info().Msgf("Error downloading cluster-template release %q: %v", version, err)
		filesToCache(GenericFiles, "generic")
	}
}

func resolveClusterTemplateVersion() (string, error) {
	if version := strings.TrimSpace(os.Getenv(clusterTemplateVersionEnv)); version != "" {
		if !clusterTemplateVersionPattern.MatchString(version) {
			return "", fmt.Errorf("invalid %s value %q", clusterTemplateVersionEnv, version)
		}

		return version, nil
	}

	req, err := http.NewRequest(http.MethodGet, clusterTemplateLatestReleaseURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := clusterTemplateHTTPClient.Do(req)
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

func downloadClusterTemplateRelease(version string) error {
	downloadURL := fmt.Sprintf(clusterTemplateReleaseArchiveURL, url.PathEscape(version))
	resp, err := clusterTemplateHTTPClient.Get(downloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status for release archive %q: %s", version, resp.Status)
	}

	return extractClusterTemplateArchive(resp.Body)
}

func extractClusterTemplateArchive(reader io.Reader) error {
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

			mode := os.FileMode(header.Mode)
			if mode.Perm() == 0 {
				mode = 0o644
			}

			file, err := os.OpenFile(cleanTargetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
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

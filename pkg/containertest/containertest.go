package containertest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Paths          []string    `yaml:"paths"`
	Files          []FileCheck `yaml:"files"`
	URLs           []URLCheck  `yaml:"urls"`
	TimeoutSeconds int         `yaml:"timeoutSeconds"`
}

type FileCheck struct {
	Path        string   `yaml:"path"`
	Contains    []string `yaml:"contains"`
	NotContains []string `yaml:"notContains"`
}

type URLCheck struct {
	URL      string   `yaml:"url"`
	Status   int      `yaml:"status"`
	Contains []string `yaml:"contains"`
}

const defaultTimeoutSeconds = 10

var (
	readFileFn          = os.ReadFile
	statFn              = os.Stat
	newRequestWithCtxFn = http.NewRequestWithContext
	httpDoFn            = func(client *http.Client, req *http.Request) (*http.Response, error) { return client.Do(req) }
)

func RunFromConfigFile(configPath string) error {
	configData, err := readFileFn(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file %q: %w", configPath, err)
	}

	cfg := Config{}
	if err := yaml.Unmarshal(configData, &cfg); err != nil {
		return fmt.Errorf("failed to parse config file %q: %w", configPath, err)
	}

	return Run(cfg)
}

func Run(cfg Config) error {
	timeout := defaultTimeoutSeconds * time.Second
	if cfg.TimeoutSeconds > 0 {
		timeout = time.Duration(cfg.TimeoutSeconds) * time.Second
	}

	client := &http.Client{Timeout: timeout}
	failures := make([]string, 0)

	for _, path := range cfg.Paths {
		if _, err := statFn(path); err != nil {
			failures = append(failures, fmt.Sprintf("path check failed for %q: %v", path, err))
		}
	}

	for _, fileCheck := range cfg.Files {
		if fileCheck.Path == "" {
			failures = append(failures, "file check failed: path is required")
			continue
		}

		contents, err := readFileFn(fileCheck.Path)
		if err != nil {
			failures = append(failures, fmt.Sprintf("file check failed for %q: %v", fileCheck.Path, err))
			continue
		}

		contentsString := string(contents)
		for _, value := range fileCheck.Contains {
			if !strings.Contains(contentsString, value) {
				failures = append(failures, fmt.Sprintf("file check failed for %q: missing %q", fileCheck.Path, value))
			}
		}
		for _, value := range fileCheck.NotContains {
			if strings.Contains(contentsString, value) {
				failures = append(failures, fmt.Sprintf("file check failed for %q: found forbidden value %q", fileCheck.Path, value))
			}
		}
	}

	for _, urlCheck := range cfg.URLs {
		if urlCheck.URL == "" {
			failures = append(failures, "url check failed: url is required")
			continue
		}

		parsedURL, err := url.ParseRequestURI(urlCheck.URL)
		if err != nil {
			failures = append(failures, fmt.Sprintf("url check failed for %q: %v", urlCheck.URL, err))
			continue
		}
		if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			failures = append(failures, fmt.Sprintf("url check failed for %q: unsupported scheme %q", urlCheck.URL, parsedURL.Scheme))
			continue
		}

		req, err := newRequestWithCtxFn(context.Background(), http.MethodGet, urlCheck.URL, nil)
		if err != nil {
			failures = append(failures, fmt.Sprintf("url check failed for %q: %v", urlCheck.URL, err))
			continue
		}

		bodyBytes, statusCode, err := func() ([]byte, int, error) {
			resp, reqErr := httpDoFn(client, req)
			if reqErr != nil {
				return nil, 0, reqErr
			}
			defer resp.Body.Close()

			body, readErr := io.ReadAll(resp.Body)
			if readErr != nil {
				return nil, 0, readErr
			}

			return body, resp.StatusCode, nil
		}()
		if err != nil {
			failures = append(failures, fmt.Sprintf("url check failed for %q: %v", urlCheck.URL, err))
			continue
		}

		expectedStatus := http.StatusOK
		if urlCheck.Status > 0 {
			expectedStatus = urlCheck.Status
		}
		if statusCode != expectedStatus {
			failures = append(failures, fmt.Sprintf("url check failed for %q: expected status %d got %d", urlCheck.URL, expectedStatus, statusCode))
		}

		bodyString := string(bodyBytes)
		for _, value := range urlCheck.Contains {
			if !strings.Contains(bodyString, value) {
				failures = append(failures, fmt.Sprintf("url check failed for %q: missing %q", urlCheck.URL, value))
			}
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf("container tests failed:\n- %s", strings.Join(failures, "\n- "))
	}

	return nil
}

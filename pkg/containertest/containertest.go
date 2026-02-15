package containertest

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Image           string            `yaml:"image"`
	Env             map[string]string `yaml:"env"`
	Paths           []string          `yaml:"paths"`
	ExternalStorage []string          `yaml:"externalStorage"`
	Files           []FileCheck       `yaml:"files"`
	URLs            []URLCheck        `yaml:"urls"`
	TCP             []TCPCheck        `yaml:"tcp"`
	Commands        []string          `yaml:"commands"`
	TimeoutSeconds  int               `yaml:"timeoutSeconds"`
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

type TCPCheck struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type ContainerConfig struct {
	Env map[string]string
}

type HTTPTestConfig struct {
	Port       string
	Path       string
	StatusCode int
}

const defaultTimeoutSeconds = 10

var (
	readFileFn = os.ReadFile

	runContainerFn = func(ctx context.Context, image string, opts ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return testcontainers.Run(ctx, image, opts...)
	}
	terminateContainerFn = func(ctx context.Context, c testcontainers.Container) error {
		return c.Terminate(ctx)
	}
	containerExitCodeFn = func(ctx context.Context, c testcontainers.Container) (int, error) {
		state, err := c.State(ctx)
		if err != nil {
			return 0, err
		}
		return state.ExitCode, nil
	}

	runPathCheckFn = runPathCheck
	runFileCheckFn = runFileCheck
	runURLCheckFn  = runURLCheck
	runTCPCheckFn  = runTCPCheck
	runCommandFn   = runCommand
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

	image := GetTestImage(cfg.Image)
	containerCfg := &ContainerConfig{Env: cfg.Env}
	failures := make([]string, 0)

	hasChecks := len(cfg.Paths) > 0 || len(cfg.ExternalStorage) > 0 || len(cfg.Files) > 0 || len(cfg.URLs) > 0 || len(cfg.TCP) > 0 || len(cfg.Commands) > 0
	if hasChecks && image == "" {
		return fmt.Errorf("container tests failed:\n- image is required (set `image` in config or TEST_IMAGE env var)")
	}

	for _, path := range cfg.Paths {
		if err := runPathCheckFn(image, path, containerCfg, timeout); err != nil {
			failures = append(failures, fmt.Sprintf("path check failed for %q: %v", path, err))
		}
	}

	externalStorage := append([]string{}, cfg.ExternalStorage...)
	if len(externalStorage) > 0 && !containsString(externalStorage, "/config") {
		externalStorage = append(externalStorage, "/config")
	}
	for _, path := range externalStorage {
		if err := runPathCheckFn(image, path, containerCfg, timeout); err != nil {
			failures = append(failures, fmt.Sprintf("external storage check failed for %q: %v", path, err))
		}
	}

	for _, fileCheck := range cfg.Files {
		if err := runFileCheckFn(image, fileCheck, containerCfg, timeout); err != nil {
			failures = append(failures, fmt.Sprintf("file check failed for %q: %v", fileCheck.Path, err))
		}
	}

	for _, urlCheck := range cfg.URLs {
		if err := runURLCheckFn(image, urlCheck, containerCfg, timeout); err != nil {
			failures = append(failures, fmt.Sprintf("url check failed for %q: %v", urlCheck.URL, err))
		}
	}

	for _, tcpCheck := range cfg.TCP {
		if err := runTCPCheckFn(image, tcpCheck, containerCfg, timeout); err != nil {
			failures = append(failures, fmt.Sprintf("tcp check failed for %q: %v", netJoinHostPort(tcpCheck.Host, tcpCheck.Port), err))
		}
	}

	for _, command := range cfg.Commands {
		trimmedCommand := strings.TrimSpace(command)
		if trimmedCommand == "" {
			failures = append(failures, "command check failed: command is required")
			continue
		}

		output, err := runCommandFn(image, cfg.Env, trimmedCommand, timeout)
		if err != nil {
			if strings.TrimSpace(output) != "" {
				failures = append(failures, fmt.Sprintf("command check failed for %q: %v (output: %s)", trimmedCommand, err, strings.TrimSpace(output)))
				continue
			}
			failures = append(failures, fmt.Sprintf("command check failed for %q: %v", trimmedCommand, err))
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf("container tests failed:\n- %s", strings.Join(failures, "\n- "))
	}

	return nil
}

func GetTestImage(defaultImage string) string {
	image := os.Getenv("TEST_IMAGE")
	if image == "" {
		return defaultImage
	}
	return image
}

func applyContainerConfig(config *ContainerConfig) []testcontainers.ContainerCustomizer {
	var opts []testcontainers.ContainerCustomizer
	if config == nil {
		return opts
	}
	if len(config.Env) > 0 {
		opts = append(opts, testcontainers.WithEnv(config.Env))
	}
	return opts
}

func runContainer(ctx context.Context, image string, opts ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
	return runContainerFn(ctx, image, opts...)
}

func assertExitZero(ctx context.Context, c testcontainers.Container, what string) error {
	exitCode, err := containerExitCodeFn(ctx, c)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		return fmt.Errorf("%s (exit code %d)", what, exitCode)
	}
	return nil
}

func runPathCheck(image string, path string, config *ContainerConfig, timeout time.Duration) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("path is required")
	}
	_, err := runCommand(image, config.Env, "test -d "+path, timeout)
	return err
}

func runFileCheck(image string, fileCheck FileCheck, config *ContainerConfig, timeout time.Duration) error {
	if strings.TrimSpace(fileCheck.Path) == "" {
		return fmt.Errorf("path is required")
	}

	if _, err := runCommand(image, config.Env, "test -f "+fileCheck.Path, timeout); err != nil {
		return err
	}

	for _, value := range fileCheck.Contains {
		command := fmt.Sprintf("sh -c %q", "grep -F -- "+strconv.Quote(value)+" "+strconv.Quote(fileCheck.Path)+" >/dev/null")
		if _, err := runCommand(image, config.Env, command, timeout); err != nil {
			return fmt.Errorf("missing %q", value)
		}
	}
	for _, value := range fileCheck.NotContains {
		command := fmt.Sprintf("sh -c %q", "! grep -F -- "+strconv.Quote(value)+" "+strconv.Quote(fileCheck.Path)+" >/dev/null")
		if _, err := runCommand(image, config.Env, command, timeout); err != nil {
			return fmt.Errorf("found forbidden value %q", value)
		}
	}

	return nil
}

func runURLCheck(image string, urlCheck URLCheck, config *ContainerConfig, timeout time.Duration) error {
	if strings.TrimSpace(urlCheck.URL) == "" {
		return fmt.Errorf("url is required")
	}

	parsedURL, err := url.ParseRequestURI(urlCheck.URL)
	if err != nil {
		return err
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("unsupported scheme %q", parsedURL.Scheme)
	}

	port := parsedURL.Port()
	if port == "" {
		if parsedURL.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}

	path := parsedURL.EscapedPath()
	if path == "" {
		path = "/"
	}

	httpConfig := HTTPTestConfig{Port: port, Path: path, StatusCode: urlCheck.Status}
	return testHTTPEndpoint(image, httpConfig, config, timeout)
}

func runTCPCheck(image string, tcpCheck TCPCheck, config *ContainerConfig, timeout time.Duration) error {
	if tcpCheck.Port <= 0 {
		return fmt.Errorf("port must be greater than 0")
	}
	return testListeningPort(image, strconv.Itoa(tcpCheck.Port), config, timeout)
}

func testHTTPEndpoint(image string, httpConfig HTTPTestConfig, containerConfig *ContainerConfig, timeout time.Duration) error {
	if httpConfig.Path == "" {
		httpConfig.Path = "/"
	}
	if httpConfig.StatusCode == 0 {
		httpConfig.StatusCode = 200
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	portStr := httpConfig.Port + "/tcp"
	portTCP := nat.Port(portStr)

	opts := []testcontainers.ContainerCustomizer{
		testcontainers.WithExposedPorts(portStr),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort(portTCP),
			wait.ForHTTP(httpConfig.Path).WithPort(portTCP).WithStatusCodeMatcher(func(status int) bool {
				return status == httpConfig.StatusCode
			}),
		),
	}
	opts = append(opts, applyContainerConfig(containerConfig)...)

	c, err := runContainer(ctx, image, opts...)
	if err != nil {
		return err
	}
	defer func() { _ = terminateContainerFn(ctx, c) }()

	return nil
}

func testListeningPort(image string, port string, containerConfig *ContainerConfig, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	portStr := port + "/tcp"
	portTCP := nat.Port(portStr)

	opts := []testcontainers.ContainerCustomizer{
		testcontainers.WithExposedPorts(portStr),
		testcontainers.WithWaitStrategy(wait.ForListeningPort(portTCP)),
	}
	opts = append(opts, applyContainerConfig(containerConfig)...)

	c, err := runContainer(ctx, image, opts...)
	if err != nil {
		return err
	}
	defer func() { _ = terminateContainerFn(ctx, c) }()

	return nil
}

func runCommand(image string, env map[string]string, command string, timeout time.Duration) (string, error) {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return "", fmt.Errorf("command is required")
	}
	if image == "" {
		return "", fmt.Errorf("image is required for command checks (set config image or TEST_IMAGE)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	opts := []testcontainers.ContainerCustomizer{
		testcontainers.WithEntrypoint(fields[0]),
		testcontainers.WithWaitStrategy(wait.ForExit()),
	}
	if len(env) > 0 {
		opts = append(opts, testcontainers.WithEnv(env))
	}
	if len(fields) > 1 {
		opts = append(opts, testcontainers.WithEntrypointArgs(fields[1:]...))
	}

	c, err := runContainer(ctx, image, opts...)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("timed out after %s", timeout)
		}
		return "", err
	}
	defer func() { _ = terminateContainerFn(ctx, c) }()

	if err := assertExitZero(ctx, c, fmt.Sprintf("command %q should succeed", command)); err != nil {
		return "", err
	}

	return "", nil
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func netJoinHostPort(host string, port int) string {
	if strings.TrimSpace(host) == "" {
		host = "127.0.0.1"
	}
	return host + ":" + strconv.Itoa(port)
}

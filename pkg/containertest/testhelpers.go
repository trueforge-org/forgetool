package containertest

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	colorReset  = "\033[0m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"

	defaultWaitStartupTimeout = 2 * time.Minute
)

var runContainerBackend = func(ctx context.Context, image string, opts ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
	return testcontainers.Run(ctx, image, opts...)
}

var mkdirTempFn = os.MkdirTemp

func envTruthy(name string) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	switch value {
	case "1", "true", "yes", "on", "y":
		return true
	default:
		return false
	}
}

func colorsEnabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	if envTruthy("FORCE_COLOR") || envTruthy("CLICOLOR_FORCE") {
		return true
	}

	if strings.EqualFold(strings.TrimSpace(os.Getenv("TERM")), "dumb") {
		return false
	}

	return true
}

func debugEnabled() bool {
	return envTruthy("TESTHELPERS_DEBUG")
}

func shouldDumpContainerLogs(testFailed bool) bool {
	mode := strings.TrimSpace(strings.ToLower(os.Getenv("TESTHELPERS_CONTAINER_LOGS")))
	switch mode {
	case "always", "all":
		return true
	case "never", "off", "none", "0", "false", "no":
		return false
	case "success", "passed", "pass":
		return !testFailed
	case "failure", "fail", "failed", "onfail", "on-fail", "error", "errors", "":
		return testFailed
	default:
		logWarn("Unknown TESTHELPERS_CONTAINER_LOGS mode %q, defaulting to failure-only", mode)
		return testFailed
	}
}

func logPrefix(level string) string {
	if !colorsEnabled() {
		switch level {
		case "DEBUG":
			return "🐛 [DEBUG]"
		case "WARN":
			return "⚠️ [WARN]"
		case "ERROR":
			return "❌ [ERROR]"
		case "OK":
			return "✅ [OK]"
		default:
			return "ℹ️ [INFO]"
		}
	}

	switch level {
	case "DEBUG":
		return colorCyan + "🐛 [DEBUG]" + colorReset
	case "WARN":
		return colorYellow + "⚠️ [WARN]" + colorReset
	case "ERROR":
		return colorRed + "❌ [ERROR]" + colorReset
	case "OK":
		return colorGreen + "✅ [OK]" + colorReset
	default:
		return colorBlue + "ℹ️ [INFO]" + colorReset
	}
}

func logInfo(format string, args ...any) {
	fmt.Printf("%s %s %s\n", time.Now().Format("15:04:05"), logPrefix("INFO"), fmt.Sprintf(format, args...))
}

func logDebug(format string, args ...any) {
	if !debugEnabled() {
		return
	}
	fmt.Printf("%s %s %s\n", time.Now().Format("15:04:05"), logPrefix("DEBUG"), fmt.Sprintf(format, args...))
}

func logWarn(format string, args ...any) {
	fmt.Printf("%s %s %s\n", time.Now().Format("15:04:05"), logPrefix("WARN"), fmt.Sprintf(format, args...))
}

func logError(format string, args ...any) {
	fmt.Printf("%s %s %s\n", time.Now().Format("15:04:05"), logPrefix("ERROR"), fmt.Sprintf(format, args...))
}

func logOK(format string, args ...any) {
	fmt.Printf("%s %s %s\n", time.Now().Format("15:04:05"), logPrefix("OK"), fmt.Sprintf(format, args...))
}

func separatorLine(runeChar string, count int) string {
	if count <= 0 {
		count = 72
	}
	return strings.Repeat(runeChar, count)
}

func logSection(title string) {
	line := separatorLine("=", 72)
	if colorsEnabled() {
		fmt.Printf("%s %s\n", colorCyan+line+colorReset, colorCyan+title+colorReset)
		fmt.Printf("%s\n", colorCyan+line+colorReset)
		return
	}
	fmt.Printf("%s\n%s\n%s\n", line, title, line)
}

func envSummary(env map[string]string) string {
	if len(env) == 0 {
		return "none"
	}

	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	return fmt.Sprintf("%d vars [%s]", len(keys), strings.Join(keys, ", "))
}

func commandString(entrypoint string, args []string) string {
	if len(args) == 0 {
		return entrypoint
	}

	return entrypoint + " " + strings.Join(args, " ")
}

func terminateContainer(ctx context.Context, container testcontainers.Container, label string) error {
	logDebug("🧹 Cleaning up container for %s", label)
	if err := container.Terminate(ctx); err != nil {
		logError("Failed to terminate container for %s: %v", label, err)
		return err
	}
	logDebug("Container terminated for %s", label)
	return nil
}

func dumpContainerLogs(ctx context.Context, c testcontainers.Container, label string) {
	logSection(fmt.Sprintf("📦 Container Logs (%s)", label))

	reader, err := c.Logs(ctx)
	if err != nil {
		logWarn("Unable to fetch container logs for %s: %v", label, err)
		return
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		logWarn("Unable to read container logs for %s: %v", label, err)
		return
	}

	text := strings.TrimSpace(string(content))
	if text == "" {
		logInfo("No container logs were emitted for %s", label)
		return
	}

	fmt.Println(text)
	logSection(fmt.Sprintf("✅ End Container Logs (%s)", label))
}

// GetTestImage returns the image to test from TEST_IMAGE env var or falls back to the default
func GetTestImage(defaultImage string) string {
	image := os.Getenv("TEST_IMAGE")
	if image == "" {
		logInfo("Using default test image: %s", defaultImage)
		return defaultImage
	}
	logInfo("Using TEST_IMAGE override: %s", image)
	return image
}

// MountConfig defines a folder to be mounted from a host tmp dir into the container.
type MountConfig struct {
	Path  string `yaml:"path"`  // target path inside the container (required)
	Chmod string `yaml:"chmod"` // optional permissions in octal notation (e.g. "755")
	Chown string `yaml:"chown"` // optional ownership in "uid:gid" notation (e.g. "568:568")
}

// ContainerConfig holds optional container configuration
type ContainerConfig struct {
	Env            map[string]string // Environment variables to set in the container
	Command        []string          // Override the container entrypoint via WithEntrypoint / WithEntrypointArgs.
	ReadOnlyRootfs bool              // Mount the container root filesystem as read-only
	Mounts         []MountConfig     // Folders to mount from host tmp dirs into the container
}

// parseChown splits a "uid:gid" string into integer uid and gid values.
func parseChown(chown string) (int, int, error) {
	parts := strings.SplitN(chown, ":", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid chown format %q (expected uid:gid)", chown)
	}
	uid, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid uid %q: %w", parts[0], err)
	}
	if uid < 0 {
		return 0, 0, fmt.Errorf("uid must be >= 0, got %d", uid)
	}
	gid, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid gid %q: %w", parts[1], err)
	}
	if gid < 0 {
		return 0, 0, fmt.Errorf("gid must be >= 0, got %d", gid)
	}
	return uid, gid, nil

}

// applyContainerConfig applies optional container configuration and returns a cleanup
// function that removes any host temp directories created for mounts.
func applyContainerConfig(config *ContainerConfig) ([]testcontainers.ContainerCustomizer, func()) {
	var opts []testcontainers.ContainerCustomizer
	var tmpDirs []string

	if config == nil {
		logDebug("No extra container config provided")
		return opts, func() {}
	}

	// Apply environment variables
	if len(config.Env) > 0 {
		opts = append(opts, testcontainers.WithEnv(config.Env))
		logInfo("Applying container environment: %s", envSummary(config.Env))
	} else {
		logDebug("Container config provided without env vars")
	}

	// Apply entrypoint override
	if len(config.Command) > 0 {
		entrypoint := config.Command[0]
		if strings.TrimSpace(entrypoint) != "" {
			opts = append(opts, testcontainers.WithEntrypoint(entrypoint))
			if len(config.Command) > 1 {
				opts = append(opts, testcontainers.WithEntrypointArgs(config.Command[1:]...))
			}
			logInfo("Applying container entrypoint: %s", strings.Join(config.Command, " "))
		}
	}

	// Apply read-only root filesystem
	if config.ReadOnlyRootfs {
		opts = append(opts, testcontainers.WithHostConfigModifier(func(hc *container.HostConfig) {
			hc.ReadonlyRootfs = true
		}))
		logInfo("Applying read-only root filesystem")
	}

	// Apply mounts: create a tmp dir on the host for each mount entry
	for _, mount := range config.Mounts {
		mountPath := strings.TrimSpace(mount.Path)
		if mountPath == "" {
			logWarn("Skipping mount with empty path")
			continue
		}
		if !strings.HasPrefix(mountPath, "/") {
			logWarn("Skipping mount with non-absolute path %q", mountPath)
			continue
		}

		tmpDir, err := mkdirTempFn("", "containertest-mount-*")
		if err != nil {
			logWarn("Failed to create temp dir for mount %s: %v", mountPath, err)
			continue
		}
		tmpDirs = append(tmpDirs, tmpDir)

		if mount.Chmod != "" {
			mode, parseErr := strconv.ParseUint(mount.Chmod, 8, 32)
			if parseErr != nil {
				logWarn("Invalid chmod %q for mount %s: %v", mount.Chmod, mountPath, parseErr)
			} else if chmodErr := os.Chmod(tmpDir, os.FileMode(mode)); chmodErr != nil {
				logWarn("Failed to chmod temp dir for mount %s: %v", mountPath, chmodErr)
			}
		}

		if mount.Chown != "" {
			uid, gid, chownParseErr := parseChown(mount.Chown)
			if chownParseErr != nil {
				logWarn("Invalid chown %q for mount %s: %v", mount.Chown, mountPath, chownParseErr)
			} else if lchownErr := os.Lchown(tmpDir, uid, gid); lchownErr != nil {
				logWarn("Failed to chown temp dir for mount %s: %v", mountPath, lchownErr)
			}
		}

		logInfo("Mounting tmp dir %s -> %s (chmod=%q chown=%q)", tmpDir, mountPath, mount.Chmod, mount.Chown)
		opts = append(opts, testcontainers.WithMounts(testcontainers.BindMount(tmpDir, testcontainers.ContainerMountTarget(mountPath))))
	}

	cleanup := func() {
		for _, dir := range tmpDirs {
			logDebug("Removing mount tmp dir %s", dir)
			if removeErr := os.RemoveAll(dir); removeErr != nil {
				logWarn("Failed to remove mount tmp dir %s: %v", dir, removeErr)
			}
		}
	}

	return opts, cleanup
}

// runContainer is a tiny helper to start a container with common patterns centralized.
func runContainer(ctx context.Context, image string, opts ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
	runStart := time.Now()
	logInfo("🚀 Starting container workflow: image=%s customizers=%d", image, len(opts))
	logInfo("Container workflow step: resolving image locally / pulling if needed")

	phaseStart := runStart
	shortContainerID := func(c testcontainers.Container) string {
		if c == nil {
			return "unknown"
		}

		id := c.GetContainerID()
		if len(id) == 0 {
			return "unknown"
		}
		if len(id) > 12 {
			return id[:12]
		}

		return id
	}

	lifecycleLogs := testcontainers.WithAdditionalLifecycleHooks(testcontainers.ContainerLifecycleHooks{
		PostCreates: []testcontainers.ContainerHook{
			func(_ context.Context, c testcontainers.Container) error {
				elapsed := time.Since(phaseStart).Round(time.Millisecond)
				total := time.Since(runStart).Round(time.Millisecond)
				logInfo("Container created: id=%s elapsed=%s total=%s", shortContainerID(c), elapsed, total)
				phaseStart = time.Now()
				return nil
			},
		},
		PreStarts: []testcontainers.ContainerHook{
			func(_ context.Context, c testcontainers.Container) error {
				elapsed := time.Since(phaseStart).Round(time.Millisecond)
				total := time.Since(runStart).Round(time.Millisecond)
				logInfo("Container starting: id=%s pull/createElapsed=%s total=%s", shortContainerID(c), elapsed, total)
				phaseStart = time.Now()
				return nil
			},
		},
		PostStarts: []testcontainers.ContainerHook{
			func(_ context.Context, c testcontainers.Container) error {
				elapsed := time.Since(phaseStart).Round(time.Millisecond)
				total := time.Since(runStart).Round(time.Millisecond)
				logInfo("Container started: id=%s startElapsed=%s total=%s", shortContainerID(c), elapsed, total)
				logInfo("Container readiness phase starting: executing configured wait strategies (if any)")
				phaseStart = time.Now()
				return nil
			},
		},
		PostReadies: []testcontainers.ContainerHook{
			func(_ context.Context, c testcontainers.Container) error {
				elapsed := time.Since(phaseStart).Round(time.Millisecond)
				total := time.Since(runStart).Round(time.Millisecond)
				logInfo("Container ready: id=%s waitElapsed=%s total=%s", shortContainerID(c), elapsed, total)
				return nil
			},
		},
	})

	allOpts := make([]testcontainers.ContainerCustomizer, 0, len(opts)+1)
	allOpts = append(allOpts, lifecycleLogs)
	allOpts = append(allOpts, opts...)

	logDebug("Invoking testcontainers.Run for image=%s", image)
	container, err := runContainerBackend(ctx, image, allOpts...)
	if err != nil {
		logError("Container startup workflow failed for image=%s after %s: %v", image, time.Since(runStart).Round(time.Millisecond), err)
		if container != nil {
			failureCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if shouldDumpContainerLogs(true) {
				dumpContainerLogs(failureCtx, container, "startup failure")
			} else {
				logDebug("Skipping container logs for startup failure (mode=%q)", strings.TrimSpace(strings.ToLower(os.Getenv("TESTHELPERS_CONTAINER_LOGS"))))
			}

			if termErr := terminateContainer(failureCtx, container, "startup failure"); termErr != nil {
				logWarn("Failed to terminate container after startup failure: %v", termErr)
			}
		}
		return nil, err
	}
	logOK("Container is up: image=%s totalElapsed=%s", image, time.Since(runStart).Round(time.Millisecond))
	return container, nil
}

func readContainerLogs(ctx context.Context, c testcontainers.Container) (string, error) {
	reader, err := c.Logs(ctx)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// assertExitCode waits for container exit (via wait strategy set by caller) and verifies the exit code.
func assertExitCode(ctx context.Context, c testcontainers.Container, what string, expectedExitCode int) error {
	logInfo("Checking container exit code for %s", what)
	state, err := c.State(ctx)
	if err != nil {
		logError("Failed to read container state for %s: %v", what, err)
		return fmt.Errorf("failed to get container state: %w", err)
	}
	logDebug("Container state for %s: running=%t, status=%s, exitCode=%d", what, state.Running, state.Status, state.ExitCode)
	if state.ExitCode != expectedExitCode {
		logError("Container exit code mismatch for %s: expected=%d actual=%d", what, expectedExitCode, state.ExitCode)
		return fmt.Errorf("%s: exit code %d", what, state.ExitCode)
	}
	logOK("Container exit code is %d for %s", expectedExitCode, what)
	return nil
}

// assertExitZero waits for container exit (via wait strategy set by caller) and verifies the exit code is zero.
func assertExitZero(ctx context.Context, c testcontainers.Container, what string) error {
	return assertExitCode(ctx, c, what, 0)
}

// HTTPTestConfig holds the configuration for HTTP endpoint tests
type HTTPTestConfig struct {
	Port              string `yaml:"port"`
	Path              string `yaml:"path"`
	StatusCode        int    `yaml:"statusCode"`
	StatusCodeMatcher func(int) bool
}

// TCPTestConfig holds the configuration for TCP wait checks.
type TCPTestConfig struct {
	Port string `yaml:"port"`
}

// CommandTestConfig holds optional configuration for command checks.
type CommandTestConfig struct {
	Command          string `yaml:"command"`
	ExpectedExitCode *int   `yaml:"expectedExitCode"`
	ExpectedContent  string `yaml:"expectedContent"`
	MatchContent     bool   `yaml:"matchContent"`
}

// HealthCommandTestConfig holds configuration for exec-based health commands.
type HealthCommandTestConfig struct {
	Command          string `yaml:"command"`
	ExpectedExitCode *int   `yaml:"expectedExitCode"`
	ExpectedContent  string `yaml:"expectedContent"`
	MatchContent     bool   `yaml:"matchContent"`
}

func describeWaitStrategies(strategies []wait.Strategy) []string {
	descriptions := make([]string, 0, len(strategies))
	for _, strategy := range strategies {
		if strategy == nil {
			descriptions = append(descriptions, "<nil>")
			continue
		}

		if stringer, ok := strategy.(fmt.Stringer); ok {
			descriptions = append(descriptions, stringer.String())
			continue
		}

		descriptions = append(descriptions, fmt.Sprintf("%T", strategy))
	}

	return descriptions
}

func logWaitStrategiesStart(label string, strategies []wait.Strategy) {
	descriptions := describeWaitStrategies(strategies)
	if len(descriptions) == 0 {
		logInfo("%s: no explicit wait strategies configured", label)
		return
	}

	for index, description := range descriptions {
		logInfo("%s: wait[%d/%d] starting: %s", label, index+1, len(descriptions), description)
	}
}

func normalizeHTTPConfig(httpConfig HTTPTestConfig) HTTPTestConfig {
	if httpConfig.Path == "" {
		httpConfig.Path = "/"
	}
	if httpConfig.StatusCode == 0 {
		httpConfig.StatusCode = 200
	}
	if httpConfig.StatusCodeMatcher == nil {
		httpConfig.StatusCodeMatcher = func(status int) bool {
			return status == httpConfig.StatusCode
		}
	}

	return httpConfig
}

func appendHTTPWaitStrategies(httpConfigs []HTTPTestConfig, portsSet map[string]struct{}, tcpWaitStrategies []wait.Strategy, httpWaitStrategies []wait.Strategy, waitStartupTimeout time.Duration) ([]wait.Strategy, []wait.Strategy, error) {
	for index, httpConfig := range httpConfigs {
		httpConfig = normalizeHTTPConfig(httpConfig)
		if strings.TrimSpace(httpConfig.Port) == "" {
			return nil, nil, fmt.Errorf("http wait #%d missing port", index+1)
		}

		portStr := strings.TrimSpace(httpConfig.Port) + "/tcp"
		portTCP := nat.Port(portStr)
		portsSet[portStr] = struct{}{}

		statusCodeMatcher := httpConfig.StatusCodeMatcher
		logInfo("Adding HTTP wait #%d: port=%s path=%s expectedStatus=%d startupTimeout=%s", index+1, portStr, httpConfig.Path, httpConfig.StatusCode, waitStartupTimeout.Round(time.Second))
		tcpWaitStrategies = append(tcpWaitStrategies,
			wait.ForListeningPort(portTCP).WithStartupTimeout(waitStartupTimeout),
		)
		httpWaitStrategies = append(httpWaitStrategies,
			wait.ForHTTP(httpConfig.Path).WithPort(portTCP).WithStatusCodeMatcher(func(status int) bool {
				return statusCodeMatcher(status)
			}).WithStartupTimeout(waitStartupTimeout),
		)
	}

	return tcpWaitStrategies, httpWaitStrategies, nil
}

func appendTCPWaitStrategies(tcpConfigs []TCPTestConfig, portsSet map[string]struct{}, tcpWaitStrategies []wait.Strategy, waitStartupTimeout time.Duration) ([]wait.Strategy, error) {
	for index, tcpConfig := range tcpConfigs {
		if strings.TrimSpace(tcpConfig.Port) == "" {
			return nil, fmt.Errorf("tcp wait #%d missing port", index+1)
		}

		portStr := strings.TrimSpace(tcpConfig.Port) + "/tcp"
		portTCP := nat.Port(portStr)
		portsSet[portStr] = struct{}{}
		logInfo("Adding TCP wait #%d: port=%s startupTimeout=%s", index+1, portStr, waitStartupTimeout.Round(time.Second))

		tcpWaitStrategies = append(tcpWaitStrategies, wait.ForListeningPort(portTCP).WithStartupTimeout(waitStartupTimeout))
	}

	return tcpWaitStrategies, nil
}

func appendFileExecWaitStrategies(filePaths []string, fileWaitStrategies []wait.Strategy) ([]wait.Strategy, error) {
	for index, filePath := range filePaths {
		trimmedPath := strings.TrimSpace(filePath)
		if trimmedPath == "" {
			return nil, fmt.Errorf("file check #%d missing file path", index+1)
		}

		logInfo("Adding file exec wait #%d: path=%s", index+1, trimmedPath)
		fileWaitStrategies = append(fileWaitStrategies, wait.ForExec([]string{"test", "-f", trimmedPath}).WithExitCode(0))
	}

	return fileWaitStrategies, nil
}

// CheckHealth waits for the container's Docker HEALTHCHECK to report healthy.
func CheckHealth(ctx context.Context, image string, containerConfig *ContainerConfig) (err error) {
	logInfo("🧪 Health check: image=%s", image)
	if containerConfig != nil {
		logInfo("Health check container config: env=%s", envSummary(containerConfig.Env))
	}

	waitStartupTimeout := defaultWaitStartupTimeout
	if deadline, hasDeadline := ctx.Deadline(); hasDeadline {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return context.DeadlineExceeded
		}
		waitStartupTimeout = remaining
	}

	waits := []wait.Strategy{wait.ForHealthCheck().WithStartupTimeout(waitStartupTimeout)}
	logInfo("Health check details: startupTimeout=%s", waitStartupTimeout.Round(time.Second))
	logWaitStrategiesStart("health check", waits)

	opts := []testcontainers.ContainerCustomizer{
		testcontainers.WithWaitStrategy(waits...),
	}

	configOpts, cleanupMounts := applyContainerConfig(containerConfig)
	opts = append(opts, configOpts...)

	container, err := runContainer(ctx, image, opts...)
	if err != nil {
		cleanupMounts()
		return err
	}
	defer func() {
		if shouldDumpContainerLogs(err != nil) {
			dumpContainerLogs(ctx, container, "health check")
		} else {
			logDebug("Skipping container logs for health check (mode=%q, failed=%t)", strings.TrimSpace(strings.ToLower(os.Getenv("TESTHELPERS_CONTAINER_LOGS"))), err != nil)
		}
		termErr := terminateContainer(ctx, container, "health check")
		if err == nil && termErr != nil {
			err = fmt.Errorf("failed to terminate container: %w", termErr)
		}
		cleanupMounts()

	}()

	logInfo("Health check completed successfully for image=%s", image)

	return nil
}

// CheckWaits verifies HTTP and TCP waits within one container start/stop lifecycle.
func CheckWaits(ctx context.Context, image string, httpConfigs []HTTPTestConfig, tcpConfigs []TCPTestConfig, containerConfig *ContainerConfig) (err error) {
	if len(httpConfigs) == 0 && len(tcpConfigs) == 0 {
		return fmt.Errorf("at least one HTTP or TCP wait must be provided")
	}

	logInfo("🧪 Wait checks: image=%s http=%d tcp=%d", image, len(httpConfigs), len(tcpConfigs))
	if containerConfig != nil {
		logInfo("Wait checks container config: env=%s", envSummary(containerConfig.Env))
	}

	portsSet := map[string]struct{}{}
	var tcpWaitStrategies []wait.Strategy
	var httpWaitStrategies []wait.Strategy

	waitStartupTimeout := defaultWaitStartupTimeout
	if deadline, hasDeadline := ctx.Deadline(); hasDeadline {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return context.DeadlineExceeded
		}
		waitStartupTimeout = remaining
	}

	var errBuild error
	tcpWaitStrategies, httpWaitStrategies, errBuild = appendHTTPWaitStrategies(httpConfigs, portsSet, tcpWaitStrategies, httpWaitStrategies, waitStartupTimeout)
	if errBuild != nil {
		return errBuild
	}

	tcpWaitStrategies, errBuild = appendTCPWaitStrategies(tcpConfigs, portsSet, tcpWaitStrategies, waitStartupTimeout)
	if errBuild != nil {
		return errBuild
	}

	// Global invariant: run all TCP waits before any HTTP waits.
	waitStrategies := append(tcpWaitStrategies, httpWaitStrategies...)
	logWaitStrategiesStart("wait checks", waitStrategies)

	exposedPorts := make([]string, 0, len(portsSet))
	for port := range portsSet {
		exposedPorts = append(exposedPorts, port)
	}
	sort.Strings(exposedPorts)
	logInfo("Wait checks details: startupTimeout=%s exposedPorts=%s", waitStartupTimeout.Round(time.Second), strings.Join(exposedPorts, ", "))

	opts := []testcontainers.ContainerCustomizer{
		testcontainers.WithExposedPorts(exposedPorts...),
		testcontainers.WithWaitStrategy(waitStrategies...),
	}

	// Apply optional container config
	configOpts, cleanupMounts := applyContainerConfig(containerConfig)
	opts = append(opts, configOpts...)

	container, err := runContainer(ctx, image, opts...)
	if err != nil {
		cleanupMounts()
		return fmt.Errorf("wait checks failed (startupTimeout=%s, exposedPorts=%s): %w", waitStartupTimeout.Round(time.Second), strings.Join(exposedPorts, ", "), err)
	}
	defer func() {
		if shouldDumpContainerLogs(err != nil) {
			dumpContainerLogs(ctx, container, "wait checks")
		} else {
			logDebug("Skipping container logs for wait checks (mode=%q, failed=%t)", strings.TrimSpace(strings.ToLower(os.Getenv("TESTHELPERS_CONTAINER_LOGS"))), err != nil)
		}
		termErr := terminateContainer(ctx, container, "wait checks")
		if err == nil && termErr != nil {
			err = fmt.Errorf("failed to terminate container: %w", termErr)
		}
		cleanupMounts()
	}()

	logInfo("Wait checks completed successfully for image=%s", image)

	return nil
}

// CheckHealthAndWaits verifies health/TCP/HTTP waits and file exec waits within one container start/stop lifecycle.
func CheckHealthAndWaits(ctx context.Context, image string, httpConfigs []HTTPTestConfig, tcpConfigs []TCPTestConfig, filePaths []string, containerConfig *ContainerConfig) (err error) {
	if len(httpConfigs) == 0 && len(tcpConfigs) == 0 && len(filePaths) == 0 {
		return fmt.Errorf("at least one HTTP, TCP, or file path wait must be provided")
	}

	logInfo("🧪 Combined health+wait checks: image=%s http=%d tcp=%d filePaths=%d", image, len(httpConfigs), len(tcpConfigs), len(filePaths))
	if containerConfig != nil {
		logInfo("Combined health+wait checks container config: env=%s", envSummary(containerConfig.Env))
	}

	portsSet := map[string]struct{}{}
	var tcpWaitStrategies []wait.Strategy
	var httpWaitStrategies []wait.Strategy
	var fileWaitStrategies []wait.Strategy

	waitStartupTimeout := defaultWaitStartupTimeout
	if deadline, hasDeadline := ctx.Deadline(); hasDeadline {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return context.DeadlineExceeded
		}
		waitStartupTimeout = remaining
	}

	var errBuild error
	tcpWaitStrategies, httpWaitStrategies, errBuild = appendHTTPWaitStrategies(httpConfigs, portsSet, tcpWaitStrategies, httpWaitStrategies, waitStartupTimeout)
	if errBuild != nil {
		return errBuild
	}

	tcpWaitStrategies, errBuild = appendTCPWaitStrategies(tcpConfigs, portsSet, tcpWaitStrategies, waitStartupTimeout)
	if errBuild != nil {
		return errBuild
	}

	fileWaitStrategies, errBuild = appendFileExecWaitStrategies(filePaths, fileWaitStrategies)
	if errBuild != nil {
		return errBuild
	}

	waitStrategies := make([]wait.Strategy, 0, 1+len(tcpWaitStrategies)+len(httpWaitStrategies)+len(fileWaitStrategies))
	if len(httpConfigs) > 0 || len(tcpConfigs) > 0 {
		healthWaitStrategy := wait.ForHealthCheck().WithStartupTimeout(waitStartupTimeout)
		waitStrategies = append(waitStrategies, healthWaitStrategy)
	}
	// Global invariant: optional health first, then all TCP waits, then all HTTP waits, then file exec waits.
	waitStrategies = append(waitStrategies, tcpWaitStrategies...)
	waitStrategies = append(waitStrategies, httpWaitStrategies...)
	waitStrategies = append(waitStrategies, fileWaitStrategies...)
	logWaitStrategiesStart("combined health+wait checks", waitStrategies)

	exposedPorts := make([]string, 0, len(portsSet))
	for port := range portsSet {
		exposedPorts = append(exposedPorts, port)
	}
	sort.Strings(exposedPorts)
	logInfo("Combined health+wait checks details: startupTimeout=%s exposedPorts=%s", waitStartupTimeout.Round(time.Second), strings.Join(exposedPorts, ", "))

	opts := []testcontainers.ContainerCustomizer{
		testcontainers.WithExposedPorts(exposedPorts...),
		testcontainers.WithWaitStrategy(waitStrategies...),
	}

	configOpts, cleanupMounts := applyContainerConfig(containerConfig)
	opts = append(opts, configOpts...)

	container, err := runContainer(ctx, image, opts...)
	if err != nil {
		cleanupMounts()
		return fmt.Errorf("combined health+wait checks failed (startupTimeout=%s, exposedPorts=%s): %w", waitStartupTimeout.Round(time.Second), strings.Join(exposedPorts, ", "), err)
	}
	defer func() {
		if shouldDumpContainerLogs(err != nil) {
			dumpContainerLogs(ctx, container, "combined health+wait checks")
		} else {
			logDebug("Skipping container logs for combined health+wait checks (mode=%q, failed=%t)", strings.TrimSpace(strings.ToLower(os.Getenv("TESTHELPERS_CONTAINER_LOGS"))), err != nil)
		}
		termErr := terminateContainer(ctx, container, "combined health+wait checks")
		if err == nil && termErr != nil {
			err = fmt.Errorf("failed to terminate container: %w", termErr)
		}
		cleanupMounts()
	}()

	logInfo("Combined health+wait checks completed successfully for image=%s", image)

	return nil
}

// CheckHTTPEndpoint verifies that an HTTP endpoint is accessible and returns the expected status code.
func CheckHTTPEndpoint(ctx context.Context, image string, httpConfig HTTPTestConfig, containerConfig *ContainerConfig) (err error) {
	httpConfig = normalizeHTTPConfig(httpConfig)

	logInfo("🧪 HTTP endpoint check: image=%s port=%s/tcp path=%s expected=%d", image, httpConfig.Port, httpConfig.Path, httpConfig.StatusCode)
	logDebug("HTTP endpoint checks always include mandatory TCP listening wait first")
	if httpConfig.StatusCodeMatcher != nil {
		logDebug("Custom HTTP status matcher configured")
	}

	return CheckWaits(ctx, image, []HTTPTestConfig{httpConfig}, nil, containerConfig)
}

// CheckTCPListening verifies that a TCP port is listening in the container.
func CheckTCPListening(ctx context.Context, image string, port string, config *ContainerConfig) (err error) {
	logInfo("🧪 TCP listening check: image=%s port=%s/tcp", image, port)

	return CheckWaits(ctx, image, nil, []TCPTestConfig{{Port: port}}, config)
}

// CheckFileExists verifies a file exists in the container.
func CheckFileExists(ctx context.Context, image string, filePath string, config *ContainerConfig) error {
	return CheckCommandSucceeds(ctx, image, config, "test", "-f", filePath)
}

// CheckFilesExist verifies that all provided files exist in the container.
func CheckFilesExist(ctx context.Context, image string, filePaths []string, config *ContainerConfig) error {
	if len(filePaths) == 0 {
		return fmt.Errorf("at least one file path must be provided")
	}

	for index, filePath := range filePaths {
		if strings.TrimSpace(filePath) == "" {
			return fmt.Errorf("file check #%d missing file path", index+1)
		}
		if err := CheckFileExists(ctx, image, filePath, config); err != nil {
			return fmt.Errorf("file check #%d failed: %w", index+1, err)
		}
	}

	return nil
}

// CheckStandardRun verifies the container can be started without altering entrypoint or args.
func CheckStandardRun(ctx context.Context, image string, config *ContainerConfig) (err error) {
	logInfo("🧪 Standard run check: image=%s", image)
	opts, cleanupMounts := applyContainerConfig(config)

	container, err := runContainer(ctx, image, opts...)
	if err != nil {
		cleanupMounts()
		return err
	}
	defer func() {
		if shouldDumpContainerLogs(err != nil) {
			dumpContainerLogs(ctx, container, "standard run check")
		} else {
			logDebug("Skipping container logs for standard run check (mode=%q, failed=%t)", strings.TrimSpace(strings.ToLower(os.Getenv("TESTHELPERS_CONTAINER_LOGS"))), err != nil)
		}
		termErr := terminateContainer(ctx, container, "standard run check")
		if err == nil && termErr != nil {
			err = fmt.Errorf("failed to terminate container: %w", termErr)
		}
		cleanupMounts()
	}()

	logInfo("Standard run check completed successfully for image=%s", image)
	return nil
}

// CheckCommand verifies that a command runs with optional expected exit code and output content checks.
func CheckCommand(ctx context.Context, image string, containerConfig *ContainerConfig, commandConfig *CommandTestConfig, entrypoint string, args ...string) (err error) {
	var expectedExitCode *int
	if commandConfig != nil {
		expectedExitCode = commandConfig.ExpectedExitCode
	} else {
		exitCodeZero := 0
		expectedExitCode = &exitCodeZero
	}

	fullCommand := commandString(entrypoint, args)
	if expectedExitCode != nil {
		logInfo("🧪 Command check: image=%s command=%q expectedExitCode=%d", image, fullCommand, *expectedExitCode)
	} else {
		logInfo("🧪 Command check: image=%s command=%q expectedExitCode=<any>", image, fullCommand)
	}
	if containerConfig != nil {
		logInfo("Command check container config: env=%s", envSummary(containerConfig.Env))
	}
	if commandConfig != nil && commandConfig.MatchContent {
		logInfo("Command check output match enabled")
	}
	waits := []wait.Strategy{wait.ForExit()}
	logWaitStrategiesStart("command check", waits)

	opts := []testcontainers.ContainerCustomizer{
		testcontainers.WithEntrypoint(entrypoint),
		testcontainers.WithWaitStrategy(waits...),
	}

	if len(args) > 0 {
		opts = append(opts, testcontainers.WithEntrypointArgs(args...))
	}

	// Apply optional container config
	configOpts, cleanupMounts := applyContainerConfig(containerConfig)
	opts = append(opts, configOpts...)

	container, err := runContainer(ctx, image, opts...)
	if err != nil {
		cleanupMounts()
		return err
	}
	defer func() {
		if shouldDumpContainerLogs(err != nil) {
			dumpContainerLogs(ctx, container, "command check")
		} else {
			logDebug("Skipping container logs for command check (mode=%q, failed=%t)", strings.TrimSpace(strings.ToLower(os.Getenv("TESTHELPERS_CONTAINER_LOGS"))), err != nil)
		}
		termErr := terminateContainer(ctx, container, "command check")
		if err == nil && termErr != nil {
			err = fmt.Errorf("failed to terminate container: %w", termErr)
		}
		cleanupMounts()
	}()

	if expectedExitCode != nil {
		if err := assertExitCode(ctx, container, fmt.Sprintf("command %q", fullCommand), *expectedExitCode); err != nil {
			logWarn("Command check failed: %q", fullCommand)
			return err
		}
	} else {
		logInfo("Command check: skipping exit code assertion (expectedExitCode unset)")
	}

	if commandConfig != nil && commandConfig.MatchContent {
		output, logErr := readContainerLogs(ctx, container)
		if logErr != nil {
			return fmt.Errorf("failed reading command output: %w", logErr)
		}

		actual := strings.TrimSpace(output)
		expected := strings.TrimSpace(commandConfig.ExpectedContent)
		if !strings.Contains(actual, expected) {
			return fmt.Errorf("command %q output mismatch: expected content %q not found in %q", fullCommand, expected, actual)
		}
		logOK("Command output contains expected content")
	}

	logInfo("Command check completed successfully: %q", fullCommand)

	return nil
}

// CheckCommands verifies that all provided commands pass using the command backend checks.
func CheckCommands(ctx context.Context, image string, containerConfig *ContainerConfig, commands []CommandTestConfig) error {
	if len(commands) == 0 {
		return fmt.Errorf("at least one command must be provided")
	}

	for index, command := range commands {
		if strings.TrimSpace(command.Command) == "" {
			return fmt.Errorf("command check #%d missing command", index+1)
		}

		commandConfig := command
		if err := CheckCommand(ctx, image, containerConfig, &commandConfig, "sh", "-c", command.Command); err != nil {
			return fmt.Errorf("command check #%d failed: %w", index+1, err)
		}
	}

	return nil
}

// CheckHealthCommands verifies that exec-based health commands succeed on a running container.
func CheckHealthCommands(ctx context.Context, image string, containerConfig *ContainerConfig, commands []HealthCommandTestConfig) (err error) {
	if len(commands) == 0 {
		return fmt.Errorf("at least one health command must be provided")
	}

	logInfo("🧪 HealthCommand checks: image=%s commands=%d", image, len(commands))
	if containerConfig != nil {
		logInfo("HealthCommand checks container config: env=%s", envSummary(containerConfig.Env))
	}

	waits := []wait.Strategy{wait.ForHealthCheck()}
	for index, command := range commands {
		trimmedCommand := strings.TrimSpace(command.Command)
		if trimmedCommand == "" {
			return fmt.Errorf("healthCommand check #%d missing command", index+1)
		}

		waitStrategy := wait.ForExec([]string{"sh", "-c", trimmedCommand})
		if command.ExpectedExitCode != nil {
			waitStrategy = waitStrategy.WithExitCode(*command.ExpectedExitCode)
		} else {
			waitStrategy = waitStrategy.WithExitCodeMatcher(func(int) bool { return true })
		}
		if command.MatchContent {
			expectedContent := command.ExpectedContent
			waitStrategy = waitStrategy.WithResponseMatcher(func(body io.Reader) bool {
				content, readErr := io.ReadAll(body)
				if readErr != nil {
					return false
				}
				return strings.Contains(string(content), expectedContent)
			})
		}

		waits = append(waits, waitStrategy)
	}
	logWaitStrategiesStart("healthCommand checks", waits)

	opts := []testcontainers.ContainerCustomizer{
		testcontainers.WithWaitStrategy(waits...),
	}

	configOpts, cleanupMounts := applyContainerConfig(containerConfig)
	opts = append(opts, configOpts...)

	container, err := runContainer(ctx, image, opts...)
	if err != nil {
		cleanupMounts()
		return err
	}
	defer func() {
		if shouldDumpContainerLogs(err != nil) {
			dumpContainerLogs(ctx, container, "healthCommand checks")
		} else {
			logDebug("Skipping container logs for healthCommand checks (mode=%q, failed=%t)", strings.TrimSpace(strings.ToLower(os.Getenv("TESTHELPERS_CONTAINER_LOGS"))), err != nil)
		}
		termErr := terminateContainer(ctx, container, "healthCommand checks")
		if err == nil && termErr != nil {
			err = fmt.Errorf("failed to terminate container: %w", termErr)
		}
		cleanupMounts()
	}()

	logInfo("HealthCommand checks completed successfully for image=%s", image)

	return nil
}

// CheckCommandSucceeds verifies that a command runs successfully in the container (exit code 0).
func CheckCommandSucceeds(ctx context.Context, image string, config *ContainerConfig, entrypoint string, args ...string) error {
	return CheckCommand(ctx, image, config, nil, entrypoint, args...)
}

// CheckRunnerOutput runs the container and verifies its output contains expectedOutput.
// When command is non-empty, it is treated as an entrypoint override for this run.
// When command is empty, the container runs with its default entrypoint.
// The container must exit for logs to be collected; wait.ForExit() is used as the wait strategy.
// When expectedExitCode is non-nil, the runner output check also verifies container exit code.
func CheckRunnerOutput(ctx context.Context, image string, containerConfig *ContainerConfig, command string, expectedOutput string, expectedExitCode *int) (err error) {
	logInfo("🧪 Runner output check: image=%s command=%q", image, command)
	if containerConfig != nil {
		logInfo("Runner output check container config: env=%s", envSummary(containerConfig.Env))
	}
	waits := []wait.Strategy{wait.ForExit()}
	logWaitStrategiesStart("runner output check", waits)

	opts := []testcontainers.ContainerCustomizer{
		testcontainers.WithWaitStrategy(waits...),
	}

	trimmedCommand := strings.TrimSpace(command)
	if trimmedCommand != "" {
		parts := strings.Fields(trimmedCommand)
		opts = append(opts, testcontainers.WithEntrypoint(parts[0]))
		if len(parts) > 1 {
			opts = append(opts, testcontainers.WithEntrypointArgs(parts[1:]...))
		}
	}

	configOpts, cleanupMounts := applyContainerConfig(containerConfig)
	opts = append(opts, configOpts...)

	c, err := runContainer(ctx, image, opts...)
	if err != nil {
		cleanupMounts()
		return err
	}
	defer func() {
		if shouldDumpContainerLogs(err != nil) {
			dumpContainerLogs(ctx, c, "runner output check")
		} else {
			logDebug("Skipping container logs for runner output check (mode=%q, failed=%t)", strings.TrimSpace(strings.ToLower(os.Getenv("TESTHELPERS_CONTAINER_LOGS"))), err != nil)
		}
		termErr := terminateContainer(ctx, c, "runner output check")
		if err == nil && termErr != nil {
			err = fmt.Errorf("failed to terminate container: %w", termErr)
		}
		cleanupMounts()
	}()

	output, readErr := readContainerLogs(ctx, c)
	if readErr != nil {
		return fmt.Errorf("failed to read container output: %w", readErr)
	}

	if expectedExitCode != nil {
		if err := assertExitCode(ctx, c, "runner output check", *expectedExitCode); err != nil {
			return err
		}
	}

	actual := strings.TrimSpace(output)
	expected := strings.TrimSpace(expectedOutput)
	matched := strings.Contains(actual, expected)
	logInfo("Runner output check details: expectedLen=%d actualLen=%d matched=%t", len(expected), len(actual), matched)
	preview := actual
	const previewLimit = 512
	if len(preview) > previewLimit {
		preview = preview[:previewLimit] + "… (truncated)"
	}
	logInfo("Runner output preview: %q", preview)

	if !matched {
		return fmt.Errorf("runner output check: expected %q not found in output %q", expected, actual)
	}

	logOK("Runner output check passed")
	return nil
}

package containertest

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	tcexec "github.com/testcontainers/testcontainers-go/exec"
	"github.com/testcontainers/testcontainers-go/wait"
)

type fakeStrategyTarget struct {
	host string
	port nat.Port

	execExitCode int
	execReader   io.Reader
	execErr      error
}

func (f *fakeStrategyTarget) Host(context.Context) (string, error) { return f.host, nil }
func (f *fakeStrategyTarget) Inspect(context.Context) (*container.InspectResponse, error) {
	return &container.InspectResponse{
		ContainerJSONBase: &container.ContainerJSONBase{State: &container.State{Running: true}},
	}, nil
}
func (f *fakeStrategyTarget) Ports(context.Context) (nat.PortMap, error) { return nat.PortMap{}, nil }
func (f *fakeStrategyTarget) MappedPort(context.Context, nat.Port) (nat.Port, error) {
	return f.port, nil
}
func (f *fakeStrategyTarget) Logs(context.Context) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}
func (f *fakeStrategyTarget) Exec(context.Context, []string, ...tcexec.ProcessOption) (int, io.Reader, error) {
	if f.execErr != nil {
		return 0, nil, f.execErr
	}
	if f.execReader != nil {
		return f.execExitCode, f.execReader, nil
	}
	return f.execExitCode, strings.NewReader(""), nil
}
func (f *fakeStrategyTarget) State(context.Context) (*container.State, error) {
	return &container.State{Running: true, Health: &container.Health{Status: "healthy"}}, nil
}
func (f *fakeStrategyTarget) CopyFileFromContainer(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

type fakeContainer struct {
	terminateErr error
	terminated   bool
	logsErr      error
	logsReader   io.ReadCloser
	state        *container.State
	stateErr     error
}

func (f *fakeContainer) GetContainerID() string                           { return "id" }
func (f *fakeContainer) Endpoint(context.Context, string) (string, error) { return "", nil }
func (f *fakeContainer) PortEndpoint(context.Context, nat.Port, string) (string, error) {
	return "", nil
}
func (f *fakeContainer) Host(context.Context) (string, error) { return "", nil }
func (f *fakeContainer) Inspect(context.Context) (*container.InspectResponse, error) {
	return &container.InspectResponse{}, nil
}
func (f *fakeContainer) MappedPort(context.Context, nat.Port) (nat.Port, error) { return "", nil }
func (f *fakeContainer) Ports(context.Context) (nat.PortMap, error)             { return nat.PortMap{}, nil }
func (f *fakeContainer) SessionID() string                                      { return "session" }
func (f *fakeContainer) IsRunning() bool                                        { return false }
func (f *fakeContainer) Start(context.Context) error                            { return nil }
func (f *fakeContainer) Stop(context.Context, *time.Duration) error             { return nil }
func (f *fakeContainer) Terminate(context.Context, ...testcontainers.TerminateOption) error {
	f.terminated = true
	return f.terminateErr
}
func (f *fakeContainer) Logs(context.Context) (io.ReadCloser, error) {
	if f.logsErr != nil {
		return nil, f.logsErr
	}
	if f.logsReader != nil {
		return f.logsReader, nil
	}
	return io.NopCloser(strings.NewReader("")), nil
}
func (f *fakeContainer) FollowOutput(testcontainers.LogConsumer) {}
func (f *fakeContainer) StartLogProducer(context.Context, ...testcontainers.LogProductionOption) error {
	return nil
}
func (f *fakeContainer) StopLogProducer() error                          { return nil }
func (f *fakeContainer) Name(context.Context) (string, error)            { return "", nil }
func (f *fakeContainer) State(context.Context) (*container.State, error) { return f.state, f.stateErr }
func (f *fakeContainer) Networks(context.Context) ([]string, error)      { return nil, nil }
func (f *fakeContainer) NetworkAliases(context.Context) (map[string][]string, error) {
	return nil, nil
}
func (f *fakeContainer) Exec(context.Context, []string, ...tcexec.ProcessOption) (int, io.Reader, error) {
	return 0, strings.NewReader(""), nil
}
func (f *fakeContainer) ContainerIP(context.Context) (string, error)                      { return "", nil }
func (f *fakeContainer) ContainerIPs(context.Context) ([]string, error)                   { return nil, nil }
func (f *fakeContainer) CopyToContainer(context.Context, []byte, string, int64) error     { return nil }
func (f *fakeContainer) CopyDirToContainer(context.Context, string, string, int64) error  { return nil }
func (f *fakeContainer) CopyFileToContainer(context.Context, string, string, int64) error { return nil }
func (f *fakeContainer) CopyFileFromContainer(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}
func (f *fakeContainer) GetLogProductionErrorChannel() <-chan error {
	ch := make(chan error)
	close(ch)
	return ch
}

type errReadCloser struct{}

func (e *errReadCloser) Read([]byte) (int, error) { return 0, errors.New("read boom") }
func (e *errReadCloser) Close() error             { return nil }

type fakeWaitStrategyNoStringer struct{}

func (f fakeWaitStrategyNoStringer) WaitUntilReady(context.Context, wait.StrategyTarget) error {
	return nil
}

func withEnv(t *testing.T, key, value string) {
	t.Helper()
	old, had := os.LookupEnv(key)
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, old)
			return
		}
		_ = os.Unsetenv(key)
	})
	if value == "" {
		_ = os.Unsetenv(key)
		return
	}
	_ = os.Setenv(key, value)
}

func setRunBackend(t *testing.T, fn func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error)) {
	t.Helper()
	old := runContainerBackend
	runContainerBackend = fn
	t.Cleanup(func() { runContainerBackend = old })
}

func TestEnvAndColorHelpers(t *testing.T) {
	withEnv(t, "BOOL_ON", " YeS ")
	if !envTruthy("BOOL_ON") {
		t.Fatalf("expected envTruthy true")
	}
	if envTruthy("DOES_NOT_EXIST") {
		t.Fatalf("expected envTruthy false")
	}

	withEnv(t, "NO_COLOR", "1")
	if colorsEnabled() {
		t.Fatalf("expected colors disabled when NO_COLOR is set")
	}

	withEnv(t, "NO_COLOR", "")
	withEnv(t, "FORCE_COLOR", "true")
	if !colorsEnabled() {
		t.Fatalf("expected colors enabled with FORCE_COLOR")
	}

	withEnv(t, "FORCE_COLOR", "")
	withEnv(t, "TERM", "dumb")
	if colorsEnabled() {
		t.Fatalf("expected colors disabled for TERM=dumb")
	}
}

func TestLoggingAndFormattingHelpers(t *testing.T) {
	withEnv(t, "NO_COLOR", "1")
	if got := logPrefix("WARN"); !strings.Contains(got, "[WARN]") {
		t.Fatalf("unexpected plain log prefix: %q", got)
	}
	if got := logPrefix("ERROR"); !strings.Contains(got, "[ERROR]") {
		t.Fatalf("unexpected plain error prefix: %q", got)
	}
	if got := logPrefix("DEBUG"); !strings.Contains(got, "[DEBUG]") {
		t.Fatalf("unexpected plain debug prefix: %q", got)
	}
	if got := logPrefix("OK"); !strings.Contains(got, "[OK]") {
		t.Fatalf("unexpected plain ok prefix: %q", got)
	}
	if got := logPrefix("SOMETHING"); !strings.Contains(got, "[INFO]") {
		t.Fatalf("unexpected plain default prefix: %q", got)
	}
	logSection("section-no-color")

	withEnv(t, "NO_COLOR", "")
	withEnv(t, "FORCE_COLOR", "1")
	if got := logPrefix("ERROR"); !strings.Contains(got, "[ERROR]") {
		t.Fatalf("unexpected color log prefix: %q", got)
	}
	if got := logPrefix("WARN"); !strings.Contains(got, "[WARN]") {
		t.Fatalf("unexpected color warn prefix: %q", got)
	}
	if got := logPrefix("DEBUG"); !strings.Contains(got, "[DEBUG]") {
		t.Fatalf("unexpected color debug prefix: %q", got)
	}
	if got := logPrefix("OK"); !strings.Contains(got, "[OK]") {
		t.Fatalf("unexpected color ok prefix: %q", got)
	}
	if got := logPrefix("SOMETHING"); !strings.Contains(got, "[INFO]") {
		t.Fatalf("unexpected color default prefix: %q", got)
	}

	if separatorLine("*", 3) != "***" {
		t.Fatalf("separator line mismatch")
	}
	if separatorLine("-", 0) != strings.Repeat("-", 72) {
		t.Fatalf("separator line default length mismatch")
	}

	if envSummary(nil) != "none" {
		t.Fatalf("expected envSummary none for nil")
	}
	if got := envSummary(map[string]string{"B": "2", "A": "1"}); got != "2 vars [A, B]" {
		t.Fatalf("unexpected env summary: %q", got)
	}

	if commandString("echo", nil) != "echo" {
		t.Fatalf("expected entrypoint only")
	}
	if commandString("echo", []string{"hi", "there"}) != "echo hi there" {
		t.Fatalf("expected joined command string")
	}

	withEnv(t, "TESTHELPERS_DEBUG", "")
	logDebug("debug disabled")
	withEnv(t, "TESTHELPERS_DEBUG", "1")
	logDebug("debug enabled")
	logInfo("info")
	logWarn("warn")
	logError("error")
	logOK("ok")
	logSection("section-color")
}

func TestShouldDumpContainerLogsModes(t *testing.T) {
	tests := []struct {
		mode       string
		failed     bool
		expectsLog bool
	}{
		{mode: "always", failed: false, expectsLog: true},
		{mode: "never", failed: true, expectsLog: false},
		{mode: "success", failed: false, expectsLog: true},
		{mode: "success", failed: true, expectsLog: false},
		{mode: "failure", failed: true, expectsLog: true},
		{mode: "failure", failed: false, expectsLog: false},
		{mode: "unknown-mode", failed: true, expectsLog: true},
		{mode: "", failed: false, expectsLog: false},
	}

	for _, test := range tests {
		withEnv(t, "TESTHELPERS_CONTAINER_LOGS", test.mode)
		if got := shouldDumpContainerLogs(test.failed); got != test.expectsLog {
			t.Fatalf("mode=%q failed=%t expected %t got %t", test.mode, test.failed, test.expectsLog, got)
		}
	}
}

func TestApplyAndNormalizeConfigHelpers(t *testing.T) {
	if got, _ := applyContainerConfig(nil); len(got) != 0 {
		t.Fatalf("expected no opts for nil config")
	}
	if got, _ := applyContainerConfig(&ContainerConfig{}); len(got) != 0 {
		t.Fatalf("expected no opts for empty env")
	}
	if got, _ := applyContainerConfig(&ContainerConfig{Env: map[string]string{"A": "1"}}); len(got) != 1 {
		t.Fatalf("expected one env opt")
	}
	if got, _ := applyContainerConfig(&ContainerConfig{ReadOnlyRootfs: true}); len(got) != 1 {
		t.Fatalf("expected one opt for ReadOnlyRootfs")
	}
	if got, _ := applyContainerConfig(&ContainerConfig{ReadOnlyRootfs: true}); len(got) == 1 {
		req := testcontainers.GenericContainerRequest{}
		if err := got[0].Customize(&req); err != nil {
			t.Fatalf("unexpected customize error: %v", err)
		}
		if req.HostConfigModifier == nil {
			t.Fatalf("expected HostConfigModifier to be set")
		}
		hc := &container.HostConfig{}
		req.HostConfigModifier(hc)
		if !hc.ReadonlyRootfs {
			t.Fatalf("expected HostConfigModifier to set ReadonlyRootfs=true")
		}
	}
	if got, _ := applyContainerConfig(&ContainerConfig{Env: map[string]string{"A": "1"}, ReadOnlyRootfs: true}); len(got) != 2 {
		t.Fatalf("expected two opts for env+ReadOnlyRootfs")
	}
	if got, _ := applyContainerConfig(&ContainerConfig{Command: []string{"myapp", "--version"}}); len(got) != 2 {
		t.Fatalf("expected two opts for Command entrypoint+args")
	}
	if got, _ := applyContainerConfig(&ContainerConfig{Env: map[string]string{"A": "1"}, Command: []string{"cmd"}, ReadOnlyRootfs: true}); len(got) != 3 {
		t.Fatalf("expected three opts for env+command+ReadOnlyRootfs, got %d", len(got))
	}

	normalized := normalizeHTTPConfig(HTTPTestConfig{Port: "8080"})
	if normalized.Path != "/" || normalized.StatusCode != 200 || normalized.StatusCodeMatcher == nil {
		t.Fatalf("unexpected normalized http config: %+v", normalized)
	}
	if !normalized.StatusCodeMatcher(200) || normalized.StatusCodeMatcher(500) {
		t.Fatalf("unexpected default status matcher behavior")
	}

	matcher := func(status int) bool { return status >= 200 && status < 300 }
	custom := normalizeHTTPConfig(HTTPTestConfig{Port: "8080", Path: "/health", StatusCode: 204, StatusCodeMatcher: matcher})
	if custom.Path != "/health" || custom.StatusCode != 204 || !custom.StatusCodeMatcher(201) {
		t.Fatalf("expected custom matcher and values to be preserved")
	}
}

func TestApplyContainerConfigMounts(t *testing.T) {
	old := mkdirTempFn
	t.Cleanup(func() { mkdirTempFn = old })

	// Seam: mkdirTempFn fails
	mkdirTempFn = func(string, string) (string, error) {
		return "", errors.New("mkdir boom")
	}
	opts, cleanup := applyContainerConfig(&ContainerConfig{Mounts: []MountConfig{{Path: "/config"}}})
	if len(opts) != 0 {
		t.Fatalf("expected no opts when mkdir fails, got %d", len(opts))
	}
	cleanup() // should be a no-op

	// Seam: mkdirTempFn succeeds; verify one mount customizer per mount entry and cleanup removes the dir
	realTmpDir := t.TempDir()
	createdDir := ""
	mkdirTempFn = func(string, string) (string, error) {
		createdDir = realTmpDir
		return realTmpDir, nil
	}
	opts, cleanup = applyContainerConfig(&ContainerConfig{
		Mounts: []MountConfig{
			{Path: "/config", Chmod: "755", Chown: "0:0"},
		},
	})
	if len(opts) != 1 {
		t.Fatalf("expected one mount opt, got %d", len(opts))
	}
	// Verify cleanup removes the dir
	if _, statErr := os.Stat(createdDir); statErr != nil {
		t.Fatalf("expected tmp dir to exist before cleanup: %v", statErr)
	}
	cleanup()
	if _, statErr := os.Stat(createdDir); !os.IsNotExist(statErr) {
		t.Fatalf("expected tmp dir to be removed after cleanup")
	}

	// Invalid chmod should warn and skip but not fail; mount is still added
	mkdirTempFn = func(string, string) (string, error) {
		return realTmpDir, nil
	}
	opts, cleanup = applyContainerConfig(&ContainerConfig{
		Mounts: []MountConfig{{Path: "/data", Chmod: "invalid"}},
	})
	if len(opts) != 1 {
		t.Fatalf("expected one mount opt even with invalid chmod, got %d", len(opts))
	}
	cleanup()

	// Invalid chown should warn and skip but not fail; mount is still added
	opts, cleanup = applyContainerConfig(&ContainerConfig{
		Mounts: []MountConfig{{Path: "/data", Chown: "notanumber:0"}},
	})
	if len(opts) != 1 {
		t.Fatalf("expected one mount opt even with invalid chown, got %d", len(opts))
	}
	cleanup()

	// Multiple mounts produce multiple opts (one per mount)
	opts, cleanup = applyContainerConfig(&ContainerConfig{
		Env:    map[string]string{"K": "V"},
		Mounts: []MountConfig{{Path: "/a"}, {Path: "/b"}},
	})
	if len(opts) != 3 { // 1 env + 2 mounts
		t.Fatalf("expected 3 opts (1 env + 2 mounts), got %d", len(opts))
	}
	cleanup()

	// Non-absolute mount path should be skipped.
	opts, cleanup = applyContainerConfig(&ContainerConfig{
		Mounts: []MountConfig{{Path: "relative/path"}},
	})
	if len(opts) != 0 {
		t.Fatalf("expected no opts for non-absolute mount path, got %d", len(opts))
	}
	cleanup()

	// Empty mount path should be skipped.
	opts, cleanup = applyContainerConfig(&ContainerConfig{
		Mounts: []MountConfig{{Path: "   "}},
	})
	if len(opts) != 0 {
		t.Fatalf("expected no opts for empty mount path, got %d", len(opts))
	}
	cleanup()

	// Invalid tmp path exercises chmod/remove cleanup warning paths.
	mkdirTempFn = func(string, string) (string, error) {
		return "bad\x00path", nil
	}
	opts, cleanup = applyContainerConfig(&ContainerConfig{
		Mounts: []MountConfig{{Path: "/bad", Chmod: "755"}},
	})
	if len(opts) != 1 {
		t.Fatalf("expected one mount opt for invalid tmp path case, got %d", len(opts))
	}
	cleanup()
}

func TestParseChown(t *testing.T) {
	uid, gid, err := parseChown("568:568")
	if err != nil || uid != 568 || gid != 568 {
		t.Fatalf("unexpected parseChown result: uid=%d gid=%d err=%v", uid, gid, err)
	}

	uid, gid, err = parseChown("0:0")
	if err != nil || uid != 0 || gid != 0 {
		t.Fatalf("unexpected parseChown zero result: uid=%d gid=%d err=%v", uid, gid, err)
	}

	if _, _, err := parseChown("nocolon"); err == nil {
		t.Fatalf("expected error for missing colon")
	}

	if _, _, err := parseChown("abc:0"); err == nil {
		t.Fatalf("expected error for non-numeric uid")
	}

	if _, _, err := parseChown("0:xyz"); err == nil {
		t.Fatalf("expected error for non-numeric gid")
	}

	if _, _, err := parseChown("-1:0"); err == nil {
		t.Fatalf("expected error for negative uid")
	}

	if _, _, err := parseChown("0:-1"); err == nil {
		t.Fatalf("expected error for negative gid")
	}
}

func TestCheckStandardRun(t *testing.T) {
	ctx := context.Background()

	t.Run("success with logs", func(t *testing.T) {
		setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
			return &fakeContainer{logsReader: io.NopCloser(strings.NewReader("container log line"))}, nil
		})
		withEnv(t, "TESTHELPERS_CONTAINER_LOGS", "always")

		if err := CheckStandardRun(ctx, "img", nil); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("terminate error returned", func(t *testing.T) {
		setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
			return &fakeContainer{terminateErr: errors.New("terminate boom")}, nil
		})
		withEnv(t, "TESTHELPERS_CONTAINER_LOGS", "never")

		err := CheckStandardRun(ctx, "img", nil)
		if err == nil || !strings.Contains(err.Error(), "failed to terminate container") {
			t.Fatalf("expected termination error, got %v", err)
		}
	})

	t.Run("startup error cleans up mounts", func(t *testing.T) {
		oldMkdirTempFn := mkdirTempFn
		t.Cleanup(func() { mkdirTempFn = oldMkdirTempFn })

		tmpDir := t.TempDir()
		mkdirTempFn = func(string, string) (string, error) {
			return tmpDir, nil
		}

		setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
			return nil, errors.New("start boom")
		})

		err := CheckStandardRun(ctx, "img", &ContainerConfig{Mounts: []MountConfig{{Path: "/config"}}})
		if err == nil {
			t.Fatalf("expected startup error")
		}

		if _, statErr := os.Stat(tmpDir); !os.IsNotExist(statErr) {
			t.Fatalf("expected mount temp dir to be removed after startup failure")
		}
	})
}

func TestWaitStrategyBuilders(t *testing.T) {
	if _, _, err := appendHTTPWaitStrategies([]HTTPTestConfig{{Path: "/"}}, map[string]struct{}{}, nil, nil, 5*time.Second); err == nil {
		t.Fatalf("expected error for missing http port")
	}

	tcpWaits, httpWaits, err := appendHTTPWaitStrategies([]HTTPTestConfig{{Port: "8080", Path: "/ready"}}, map[string]struct{}{}, nil, nil, 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tcpWaits) != 1 || len(httpWaits) != 1 {
		t.Fatalf("expected one tcp and one http wait strategy")
	}

	matcherCalled := false
	_, customHTTPWaits, err := appendHTTPWaitStrategies([]HTTPTestConfig{{
		Port: "8080",
		Path: "/ready",
		StatusCodeMatcher: func(status int) bool {
			matcherCalled = true
			return status == http.StatusNoContent
		},
	}}, map[string]struct{}{}, nil, nil, 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected custom matcher append error: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	host := strings.TrimPrefix(server.URL, "http://")
	hostParts := strings.Split(host, ":")
	if len(hostParts) != 2 {
		t.Fatalf("unexpected server host format: %q", host)
	}
	portValue, err := strconv.Atoi(hostParts[1])
	if err != nil {
		t.Fatalf("failed parsing server port: %v", err)
	}

	waitCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := customHTTPWaits[0].WaitUntilReady(waitCtx, &fakeStrategyTarget{host: hostParts[0], port: nat.Port(strconv.Itoa(portValue))}); err != nil {
		t.Fatalf("unexpected http wait execution error: %v", err)
	}
	if !matcherCalled {
		t.Fatalf("expected status matcher to be invoked")
	}

	if _, err := appendTCPWaitStrategies([]TCPTestConfig{{Port: ""}}, map[string]struct{}{}, nil, 5*time.Second); err == nil {
		t.Fatalf("expected error for missing tcp port")
	}

	tcpOnly, err := appendTCPWaitStrategies([]TCPTestConfig{{Port: "9090"}}, map[string]struct{}{}, nil, 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tcpOnly) != 1 {
		t.Fatalf("expected one tcp wait strategy")
	}
}

func TestContainerLifecycleHelpers(t *testing.T) {
	ctx := context.Background()
	failTerm := &fakeContainer{terminateErr: errors.New("terminate boom")}
	if err := terminateContainer(ctx, failTerm, "x"); err == nil {
		t.Fatalf("expected terminate error")
	}

	okTerm := &fakeContainer{}
	if err := terminateContainer(ctx, okTerm, "x"); err != nil {
		t.Fatalf("unexpected terminate error: %v", err)
	}

	logErr := &fakeContainer{logsErr: errors.New("logs boom")}
	dumpContainerLogs(ctx, logErr, "l1")

	readErr := &fakeContainer{logsReader: &errReadCloser{}}
	dumpContainerLogs(ctx, readErr, "l2")

	emptyLogs := &fakeContainer{logsReader: io.NopCloser(strings.NewReader("   \n"))}
	dumpContainerLogs(ctx, emptyLogs, "l3")

	contentLogs := &fakeContainer{logsReader: io.NopCloser(strings.NewReader("hello world"))}
	dumpContainerLogs(ctx, contentLogs, "l4")
}

func TestRunContainerAndReadLogs(t *testing.T) {
	ctx := context.Background()

	defaultBackend := runContainerBackend
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	func() {
		defer func() {
			_ = recover()
		}()
		_, _ = defaultBackend(cancelledCtx, "invalid-image-for-coverage")
	}()

	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return nil, errors.New("run boom")
	})
	if _, err := runContainer(ctx, "img"); err == nil {
		t.Fatalf("expected runContainer error")
	}

	withEnv(t, "TESTHELPERS_CONTAINER_LOGS", "always")
	failedContainer := &fakeContainer{logsReader: io.NopCloser(strings.NewReader("startup logs"))}
	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return failedContainer, errors.New("run boom with container")
	})
	if _, err := runContainer(ctx, "img"); err == nil {
		t.Fatalf("expected runContainer error when backend returns container + error")
	}
	if !failedContainer.terminated {
		t.Fatalf("expected failed-start container to be terminated")
	}

	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return &fakeContainer{}, nil
	})
	if _, err := runContainer(ctx, "img"); err != nil {
		t.Fatalf("unexpected runContainer error: %v", err)
	}

	if _, err := readContainerLogs(ctx, &fakeContainer{logsErr: errors.New("logs")}); err == nil {
		t.Fatalf("expected readContainerLogs logs error")
	}
	if _, err := readContainerLogs(ctx, &fakeContainer{logsReader: &errReadCloser{}}); err == nil {
		t.Fatalf("expected readContainerLogs read error")
	}
	output, err := readContainerLogs(ctx, &fakeContainer{logsReader: io.NopCloser(bytes.NewBufferString("abc"))})
	if err != nil || output != "abc" {
		t.Fatalf("expected successful log read, got %q err=%v", output, err)
	}
}

func TestCheckHealthCommandsResponseMatcherExecution(t *testing.T) {
	ctx := context.Background()

	setRunBackend(t, func(waitCtx context.Context, _ string, opts ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		req := testcontainers.GenericContainerRequest{}
		for _, opt := range opts {
			if err := opt.Customize(&req); err != nil {
				return nil, err
			}
		}
		if req.WaitingFor == nil {
			return nil, errors.New("missing waiting strategy")
		}

		target := &fakeStrategyTarget{
			execExitCode: 0,
			execReader:   strings.NewReader("service is healthy: ok"),
		}
		if err := req.WaitingFor.WaitUntilReady(waitCtx, target); err != nil {
			return nil, err
		}

		return &fakeContainer{state: &container.State{ExitCode: 0}}, nil
	})

	if err := CheckHealthCommands(ctx, "img", nil, []HealthCommandTestConfig{{
		Command:          "echo ok",
		ExpectedExitCode: intPtr(0),
		ExpectedContent:  "ok",
		MatchContent:     true,
	}}); err != nil {
		t.Fatalf("expected successful health command response match, got %v", err)
	}
}

func TestCheckHealthCommandsResponseMatcherReadError(t *testing.T) {
	ctx := context.Background()

	setRunBackend(t, func(waitCtx context.Context, _ string, opts ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		req := testcontainers.GenericContainerRequest{}
		for _, opt := range opts {
			if err := opt.Customize(&req); err != nil {
				return nil, err
			}
		}
		if req.WaitingFor == nil {
			return nil, errors.New("missing waiting strategy")
		}

		target := &fakeStrategyTarget{
			execExitCode: 0,
			execReader:   &errReadCloser{},
		}
		if err := req.WaitingFor.WaitUntilReady(waitCtx, target); err != nil {
			return nil, err
		}

		return &fakeContainer{state: &container.State{ExitCode: 0}}, nil
	})

	err := CheckHealthCommands(ctx, "img", nil, []HealthCommandTestConfig{{
		Command:          "echo ok",
		ExpectedExitCode: intPtr(0),
		ExpectedContent:  "ok",
		MatchContent:     true,
	}})
	if err == nil {
		t.Fatalf("expected health command response matcher read error to fail")
	}
}

func TestAssertExitCode(t *testing.T) {
	ctx := context.Background()
	if err := assertExitCode(ctx, &fakeContainer{stateErr: errors.New("state")}, "cmd", 0); err == nil {
		t.Fatalf("expected state error")
	}
	if err := assertExitCode(ctx, &fakeContainer{state: &container.State{ExitCode: 2}}, "cmd", 0); err == nil {
		t.Fatalf("expected exit mismatch")
	}
	if err := assertExitCode(ctx, &fakeContainer{state: &container.State{ExitCode: 0}}, "cmd", 0); err != nil {
		t.Fatalf("unexpected assertExitCode error: %v", err)
	}
	if err := assertExitZero(ctx, &fakeContainer{state: &container.State{ExitCode: 0}}, "cmd"); err != nil {
		t.Fatalf("unexpected assertExitZero error: %v", err)
	}
}

func TestGetTestImage(t *testing.T) {
	withEnv(t, "TEST_IMAGE", "")
	if got := GetTestImage("default"); got != "default" {
		t.Fatalf("expected default image")
	}
	withEnv(t, "TEST_IMAGE", "custom")
	if got := GetTestImage("default"); got != "custom" {
		t.Fatalf("expected override image")
	}
}

func TestWaitStrategyHelpers(t *testing.T) {
	descriptions := describeWaitStrategies([]wait.Strategy{nil, wait.ForHealthCheck(), fakeWaitStrategyNoStringer{}})
	if len(descriptions) != 3 {
		t.Fatalf("expected 3 strategy descriptions, got %d", len(descriptions))
	}
	if descriptions[0] != "<nil>" {
		t.Fatalf("expected nil marker, got %q", descriptions[0])
	}
	if !strings.Contains(descriptions[1], "health") && !strings.Contains(strings.ToLower(descriptions[1]), "health") {
		t.Fatalf("expected health strategy description, got %q", descriptions[1])
	}
	if !strings.Contains(descriptions[2], "fakeWaitStrategyNoStringer") {
		t.Fatalf("expected type-based description, got %q", descriptions[2])
	}

	logWaitStrategiesStart("empty", nil)
	logWaitStrategiesStart("filled", []wait.Strategy{wait.ForHealthCheck()})
}

func TestCheckHealthAndWaitsWithoutDocker(t *testing.T) {
	ctx := context.Background()
	withEnv(t, "TESTHELPERS_CONTAINER_LOGS", "never")

	if err := CheckHealthAndWaits(ctx, "img", nil, nil, nil, nil); err == nil {
		t.Fatalf("expected empty combined waits error")
	}

	deadlineCtx, cancel := context.WithDeadline(ctx, time.Now().Add(-time.Second))
	defer cancel()
	if err := CheckHealthAndWaits(deadlineCtx, "img", []HTTPTestConfig{{Port: "8080"}}, nil, nil, nil); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}

	if err := CheckHealthAndWaits(ctx, "img", []HTTPTestConfig{{Path: "/"}}, nil, nil, nil); err == nil {
		t.Fatalf("expected missing HTTP port error")
	}

	if err := CheckHealthAndWaits(ctx, "img", nil, []TCPTestConfig{{Port: " "}}, nil, nil); err == nil {
		t.Fatalf("expected missing TCP port error")
	}

	if err := CheckHealthAndWaits(ctx, "img", nil, nil, []string{" "}, nil); err == nil {
		t.Fatalf("expected blank file path error")
	}

	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return nil, errors.New("start failed")
	})
	if err := CheckHealthAndWaits(ctx, "img", []HTTPTestConfig{{Port: "8080"}}, nil, nil, nil); err == nil {
		t.Fatalf("expected wrapped backend startup error")
	}

	setRunBackend(t, func(waitCtx context.Context, _ string, opts ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		req := testcontainers.GenericContainerRequest{}
		for _, opt := range opts {
			if err := opt.Customize(&req); err != nil {
				return nil, err
			}
		}
		if req.WaitingFor == nil {
			return nil, errors.New("missing waiting strategy")
		}
		if len(req.ExposedPorts) != 2 {
			return nil, errors.New("expected exposed ports to be configured")
		}

		return &fakeContainer{state: &container.State{ExitCode: 0}, terminateErr: errors.New("term fail")}, nil
	})
	if err := CheckHealthAndWaits(ctx, "img", []HTTPTestConfig{{Port: "8080", Path: "/ready"}}, []TCPTestConfig{{Port: "9090"}}, []string{"/etc/hosts"}, nil); err == nil || !strings.Contains(err.Error(), "failed to terminate container") {
		t.Fatalf("expected terminate failure, got %v", err)
	}

	setRunBackend(t, func(waitCtx context.Context, _ string, opts ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		req := testcontainers.GenericContainerRequest{}
		for _, opt := range opts {
			if err := opt.Customize(&req); err != nil {
				return nil, err
			}
		}
		if req.WaitingFor == nil {
			return nil, errors.New("missing waiting strategy")
		}
		if len(req.ExposedPorts) != 0 {
			return nil, errors.New("expected no exposed ports for file-only checks")
		}

		return &fakeContainer{state: &container.State{ExitCode: 0}}, nil
	})
	if err := CheckHealthAndWaits(ctx, "img", nil, nil, []string{"/bin/sh"}, &ContainerConfig{Env: map[string]string{"A": "1"}}); err != nil {
		t.Fatalf("expected combined waits success, got %v", err)
	}
}

func TestRunContainerLifecycleHooksExecute(t *testing.T) {
	ctx := context.Background()
	withEnv(t, "TESTHELPERS_CONTAINER_LOGS", "never")

	setRunBackend(t, func(runCtx context.Context, _ string, opts ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		req := testcontainers.GenericContainerRequest{}
		for _, opt := range opts {
			if err := opt.Customize(&req); err != nil {
				return nil, err
			}
		}

		c := &fakeContainer{}
		for _, hooks := range req.LifecycleHooks {
			for _, hook := range hooks.PostCreates {
				if err := hook(runCtx, nil); err != nil {
					return nil, err
				}
				if err := hook(runCtx, c); err != nil {
					return nil, err
				}
			}
			for _, hook := range hooks.PreStarts {
				if err := hook(runCtx, c); err != nil {
					return nil, err
				}
			}
			for _, hook := range hooks.PostStarts {
				if err := hook(runCtx, c); err != nil {
					return nil, err
				}
			}
			for _, hook := range hooks.PostReadies {
				if err := hook(runCtx, c); err != nil {
					return nil, err
				}
			}
		}

		return c, nil
	})

	if _, err := runContainer(ctx, "img"); err != nil {
		t.Fatalf("expected lifecycle hook execution success, got %v", err)
	}
}

func TestHighLevelChecksWithoutDocker(t *testing.T) {
	ctx := context.Background()
	withEnv(t, "TESTHELPERS_CONTAINER_LOGS", "never")

	if err := CheckWaits(ctx, "img", nil, nil, nil); err == nil {
		t.Fatalf("expected CheckWaits to reject empty waits")
	}
	if err := CheckHTTPEndpoint(ctx, "img", HTTPTestConfig{Path: "/"}, nil); err == nil {
		t.Fatalf("expected http check error for missing port")
	}
	if err := CheckTCPListening(ctx, "img", "", nil); err == nil {
		t.Fatalf("expected tcp check error for missing port")
	}

	if err := CheckFilesExist(ctx, "img", nil, nil); err == nil {
		t.Fatalf("expected files empty error")
	}
	if err := CheckFilesExist(ctx, "img", []string{" "}, nil); err == nil {
		t.Fatalf("expected files blank path error")
	}

	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return nil, errors.New("start failed")
	})
	if err := CheckFileExists(ctx, "img", "/x", nil); err == nil {
		t.Fatalf("expected file check failure from run backend")
	}
	if err := CheckFilesExist(ctx, "img", []string{"/x"}, nil); err == nil {
		t.Fatalf("expected wrapped file check failure")
	}

	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return &fakeContainer{state: &container.State{ExitCode: 0}}, nil
	})
	if err := CheckFilesExist(ctx, "img", []string{"/x", "/y"}, nil); err != nil {
		t.Fatalf("expected successful file checks, got: %v", err)
	}
}

func TestCheckCommandAndCommands(t *testing.T) {
	ctx := context.Background()
	withEnv(t, "TESTHELPERS_CONTAINER_LOGS", "never")

	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return nil, errors.New("run failed")
	})
	if err := CheckCommand(ctx, "img", nil, nil, "echo", "ok"); err == nil {
		t.Fatalf("expected run failure")
	}

	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return &fakeContainer{stateErr: errors.New("state failed")}, nil
	})
	if err := CheckCommand(ctx, "img", nil, nil, "echo", "ok"); err == nil {
		t.Fatalf("expected state failure")
	}

	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return &fakeContainer{state: &container.State{ExitCode: 0}, logsErr: errors.New("logs failed")}, nil
	})
	if err := CheckCommand(ctx, "img", nil, &CommandTestConfig{MatchContent: true, ExpectedContent: "x"}, "sh", "-c", "echo x"); err == nil {
		t.Fatalf("expected logs read failure")
	}

	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return &fakeContainer{state: &container.State{ExitCode: 0}, logsReader: io.NopCloser(strings.NewReader("actual"))}, nil
	})
	if err := CheckCommand(ctx, "img", nil, &CommandTestConfig{MatchContent: true, ExpectedContent: "missing"}, "sh", "-c", "echo x"); err == nil {
		t.Fatalf("expected content mismatch")
	}

	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return &fakeContainer{state: &container.State{ExitCode: 0}, logsReader: io.NopCloser(strings.NewReader("contains expected"))}, nil
	})
	if err := CheckCommand(ctx, "img", nil, &CommandTestConfig{MatchContent: true, ExpectedContent: "expected"}, "sh", "-c", "echo x"); err != nil {
		t.Fatalf("unexpected command success error: %v", err)
	}

	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return &fakeContainer{state: &container.State{ExitCode: 7}}, nil
	})
	exitCodeSeven := 7
	if err := CheckCommand(ctx, "img", nil, &CommandTestConfig{ExpectedExitCode: &exitCodeSeven}, "echo", "ok"); err != nil {
		t.Fatalf("unexpected non-zero expected exit code error: %v", err)
	}

	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return &fakeContainer{state: &container.State{ExitCode: 9}}, nil
	})
	if err := CheckCommand(ctx, "img", nil, &CommandTestConfig{}, "echo", "ok"); err != nil {
		t.Fatalf("unexpected command success when expectedExitCode is unset: %v", err)
	}

	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return &fakeContainer{state: &container.State{ExitCode: 0}}, nil
	})
	if err := CheckCommand(ctx, "img", &ContainerConfig{Env: map[string]string{"A": "1"}}, nil, "true"); err != nil {
		t.Fatalf("unexpected command success for no-args path: %v", err)
	}

	withEnv(t, "TESTHELPERS_CONTAINER_LOGS", "always")
	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return &fakeContainer{state: &container.State{ExitCode: 0}, logsReader: io.NopCloser(strings.NewReader("command logs"))}, nil
	})
	if err := CheckCommand(ctx, "img", nil, nil, "echo", "ok"); err != nil {
		t.Fatalf("unexpected command success with logs enabled: %v", err)
	}
	withEnv(t, "TESTHELPERS_CONTAINER_LOGS", "never")

	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return &fakeContainer{state: &container.State{ExitCode: 0}, terminateErr: errors.New("term failed")}, nil
	})
	if err := CheckCommand(ctx, "img", nil, nil, "echo", "ok"); err == nil {
		t.Fatalf("expected terminate failure")
	}

	if err := CheckCommands(ctx, "img", nil, nil); err == nil {
		t.Fatalf("expected empty commands error")
	}
	if err := CheckCommands(ctx, "img", nil, []CommandTestConfig{{Command: " "}}); err == nil {
		t.Fatalf("expected blank command error")
	}

	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return &fakeContainer{state: &container.State{ExitCode: 0}}, nil
	})
	if err := CheckCommands(ctx, "img", nil, []CommandTestConfig{{Command: "echo one"}, {Command: "echo two"}}); err != nil {
		t.Fatalf("unexpected commands success error: %v", err)
	}

	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return &fakeContainer{state: &container.State{ExitCode: 1}}, nil
	})
	exitCodeZero := 0
	if err := CheckCommands(ctx, "img", nil, []CommandTestConfig{{Command: "false", ExpectedExitCode: &exitCodeZero}}); err == nil {
		t.Fatalf("expected wrapped command failure")
	}

	if err := CheckHealthCommands(ctx, "img", nil, nil); err == nil {
		t.Fatalf("expected empty healthCommands error")
	}
	if err := CheckHealthCommands(ctx, "img", nil, []HealthCommandTestConfig{{Command: " "}}); err == nil {
		t.Fatalf("expected blank healthCommand error")
	}

	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return nil, errors.New("start failed")
	})
	if err := CheckHealthCommands(ctx, "img", nil, []HealthCommandTestConfig{{Command: "mycommand"}}); err == nil {
		t.Fatalf("expected run backend failure for healthCommands")
	}

	withEnv(t, "TESTHELPERS_CONTAINER_LOGS", "always")
	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return &fakeContainer{state: &container.State{ExitCode: 0}, logsReader: io.NopCloser(strings.NewReader("logs"))}, nil
	})
	if err := CheckHealthCommands(ctx, "img", &ContainerConfig{Env: map[string]string{"A": "1"}}, []HealthCommandTestConfig{{
		Command:          "mycommand",
		ExpectedExitCode: &exitCodeSeven,
		ExpectedContent:  "ok",
		MatchContent:     true,
	}}); err != nil {
		t.Fatalf("unexpected healthCommands success error: %v", err)
	}

	withEnv(t, "TESTHELPERS_CONTAINER_LOGS", "never")
	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return &fakeContainer{terminateErr: errors.New("terminate failed")}, nil
	})
	if err := CheckHealthCommands(ctx, "img", nil, []HealthCommandTestConfig{{Command: "mycommand"}}); err == nil {
		t.Fatalf("expected terminate failure for healthCommands")
	}

	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return &fakeContainer{state: &container.State{ExitCode: 0}}, nil
	})
	if err := CheckCommandSucceeds(ctx, "img", nil, "echo", "ok"); err != nil {
		t.Fatalf("unexpected command succeeds error: %v", err)
	}
}

func TestCheckWaitsExecutionPaths(t *testing.T) {
	ctx := context.Background()

	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return nil, errors.New("start failed")
	})
	if err := CheckWaits(ctx, "img", []HTTPTestConfig{{Port: "8080"}}, nil, nil); err == nil {
		t.Fatalf("expected run backend failure")
	}

	withEnv(t, "TESTHELPERS_CONTAINER_LOGS", "always")
	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return &fakeContainer{state: &container.State{ExitCode: 0}, logsReader: io.NopCloser(strings.NewReader("logs"))}, nil
	})
	if err := CheckWaits(ctx, "img", []HTTPTestConfig{{Port: "8080", Path: "/ready"}}, []TCPTestConfig{{Port: "9090"}}, &ContainerConfig{Env: map[string]string{"A": "1"}}); err != nil {
		t.Fatalf("unexpected waits success error: %v", err)
	}

	withEnv(t, "TESTHELPERS_CONTAINER_LOGS", "never")
	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return &fakeContainer{terminateErr: errors.New("terminate failed")}, nil
	})
	if err := CheckWaits(ctx, "img", []HTTPTestConfig{{Port: "8080"}}, nil, nil); err == nil {
		t.Fatalf("expected terminate failure")
	}
}

func TestCheckHealthExecutionPaths(t *testing.T) {
	ctx := context.Background()

	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return nil, errors.New("start failed")
	})
	if err := CheckHealth(ctx, "img", nil); err == nil {
		t.Fatalf("expected run backend failure")
	}

	withEnv(t, "TESTHELPERS_CONTAINER_LOGS", "always")
	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return &fakeContainer{state: &container.State{ExitCode: 0}, logsReader: io.NopCloser(strings.NewReader("logs"))}, nil
	})
	if err := CheckHealth(ctx, "img", &ContainerConfig{Env: map[string]string{"A": "1"}}); err != nil {
		t.Fatalf("unexpected health success error: %v", err)
	}

	withEnv(t, "TESTHELPERS_CONTAINER_LOGS", "never")
	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return &fakeContainer{terminateErr: errors.New("terminate failed")}, nil
	})
	if err := CheckHealth(ctx, "img", nil); err == nil {
		t.Fatalf("expected terminate failure")
	}
}

func TestCheckRunnerOutput(t *testing.T) {
	ctx := context.Background()
	withEnv(t, "TESTHELPERS_CONTAINER_LOGS", "never")

	// Run backend fails
	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return nil, errors.New("start failed")
	})
	if err := CheckRunnerOutput(ctx, "img", nil, "myapp --version", "v1", nil); err == nil {
		t.Fatalf("expected run failure")
	}

	// Log read fails
	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return &fakeContainer{logsErr: errors.New("logs boom")}, nil
	})
	if err := CheckRunnerOutput(ctx, "img", nil, "", "v1", nil); err == nil {
		t.Fatalf("expected log read failure")
	}

	// Output does not contain expected string
	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return &fakeContainer{logsReader: io.NopCloser(strings.NewReader("v2.0"))}, nil
	})
	if err := CheckRunnerOutput(ctx, "img", nil, "myapp --version", "v1", nil); err == nil {
		t.Fatalf("expected content mismatch")
	}

	// Output contains expected string
	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return &fakeContainer{logsReader: io.NopCloser(strings.NewReader("myapp v1.2.3"))}, nil
	})
	if err := CheckRunnerOutput(ctx, "img", nil, "myapp --version", "v1.2.3", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No command (default entrypoint)
	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return &fakeContainer{logsReader: io.NopCloser(strings.NewReader("started"))}, nil
	})
	if err := CheckRunnerOutput(ctx, "img", nil, "", "started", nil); err != nil {
		t.Fatalf("unexpected error with no command: %v", err)
	}

	// Exit code matches expected
	expectedExitCode := 7
	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return &fakeContainer{
			logsReader: io.NopCloser(strings.NewReader("started")),
			state:      &container.State{ExitCode: 7},
		}, nil
	})
	if err := CheckRunnerOutput(ctx, "img", nil, "", "started", &expectedExitCode); err != nil {
		t.Fatalf("unexpected error with matching exit code: %v", err)
	}

	// Exit code mismatch
	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return &fakeContainer{
			logsReader: io.NopCloser(strings.NewReader("started")),
			state:      &container.State{ExitCode: 1},
		}, nil
	})
	if err := CheckRunnerOutput(ctx, "img", nil, "", "started", &expectedExitCode); err == nil {
		t.Fatalf("expected exit code mismatch")
	}

	// Terminate failure
	withEnv(t, "TESTHELPERS_CONTAINER_LOGS", "never")
	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return &fakeContainer{
			logsReader:   io.NopCloser(strings.NewReader("v1")),
			terminateErr: errors.New("terminate failed"),
		}, nil
	})
	if err := CheckRunnerOutput(ctx, "img", nil, "", "v1", nil); err == nil {
		t.Fatalf("expected terminate failure")
	}

	// Logs dumped on success when TESTHELPERS_CONTAINER_LOGS=always
	withEnv(t, "TESTHELPERS_CONTAINER_LOGS", "always")
	setRunBackend(t, func(context.Context, string, ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return &fakeContainer{logsReader: io.NopCloser(strings.NewReader("v1"))}, nil
	})
	if err := CheckRunnerOutput(ctx, "img", &ContainerConfig{Env: map[string]string{"A": "1"}}, "cmd", "v1", nil); err != nil {
		t.Fatalf("unexpected error with env config: %v", err)
	}
}

package containertest

import "context"

type testFailureReporter interface {
	Helper()
	Fatalf(format string, args ...any)
}

// RequireHTTPEndpoint runs an HTTP endpoint check and fails the test on error.
func RequireHTTPEndpoint(t testFailureReporter, ctx context.Context, image string, httpConfig HTTPTestConfig, containerConfig *ContainerConfig) {
	t.Helper()
	if err := CheckHTTPEndpoint(ctx, image, httpConfig, containerConfig); err != nil {
		t.Fatalf("HTTP endpoint check failed: %v", err)
	}
}

// RequireTCPListening runs a TCP listening check and fails the test on error.
func RequireTCPListening(t testFailureReporter, ctx context.Context, image string, port string, containerConfig *ContainerConfig) {
	t.Helper()
	if err := CheckTCPListening(ctx, image, port, containerConfig); err != nil {
		t.Fatalf("TCP listening check failed: %v", err)
	}
}

// RequireWaits runs combined HTTP/TCP waits and fails the test on error.
func RequireWaits(t testFailureReporter, ctx context.Context, image string, httpConfigs []HTTPTestConfig, tcpConfigs []TCPTestConfig, containerConfig *ContainerConfig) {
	t.Helper()
	if err := CheckWaits(ctx, image, httpConfigs, tcpConfigs, containerConfig); err != nil {
		t.Fatalf("wait checks failed: %v", err)
	}
}

// RequireFileExists runs a file existence check and fails the test on error.
func RequireFileExists(t testFailureReporter, ctx context.Context, image string, filePath string, containerConfig *ContainerConfig) {
	t.Helper()
	if err := CheckFileExists(ctx, image, filePath, containerConfig); err != nil {
		t.Fatalf("file existence check failed: %v", err)
	}
}

// RequireFilesExist runs list-based file existence checks and fails the test on error.
func RequireFilesExist(t testFailureReporter, ctx context.Context, image string, filePaths []string, containerConfig *ContainerConfig) {
	t.Helper()
	if err := CheckFilesExist(ctx, image, filePaths, containerConfig); err != nil {
		t.Fatalf("file existence checks failed: %v", err)
	}
}

// RequireCommandSucceeds runs a command check and fails the test on error.
func RequireCommandSucceeds(t testFailureReporter, ctx context.Context, image string, containerConfig *ContainerConfig, entrypoint string, args ...string) {
	t.Helper()
	if err := CheckCommandSucceeds(ctx, image, containerConfig, entrypoint, args...); err != nil {
		t.Fatalf("command check failed: %v", err)
	}
}

// RequireCommand runs a command check with optional expected exit code and output content checks.
func RequireCommand(t testFailureReporter, ctx context.Context, image string, containerConfig *ContainerConfig, commandConfig *CommandTestConfig, entrypoint string, args ...string) {
	t.Helper()
	if err := CheckCommand(ctx, image, containerConfig, commandConfig, entrypoint, args...); err != nil {
		t.Fatalf("command check failed: %v", err)
	}
}

// RequireCommands runs list-based command checks and fails the test on error.
func RequireCommands(t testFailureReporter, ctx context.Context, image string, containerConfig *ContainerConfig, commands []CommandTestConfig) {
	t.Helper()
	if err := CheckCommands(ctx, image, containerConfig, commands); err != nil {
		t.Fatalf("command checks failed: %v", err)
	}
}

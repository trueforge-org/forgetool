package containertest

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
)

func TestRunFromConfigFileSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "container-test.yaml")
	config := strings.Join([]string{
		"image: docker.io/library/alpine:3.22",
		"paths:",
		"  - /app",
		"commands:",
		"  - echo ok",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	oldRunPathCheckFn := runPathCheckFn
	oldRunCommandFn := runCommandFn
	t.Cleanup(func() {
		runPathCheckFn = oldRunPathCheckFn
		runCommandFn = oldRunCommandFn
	})

	runPathCheckFn = func(image, path string, cfg *ContainerConfig, timeout time.Duration) error {
		return nil
	}
	runCommandFn = func(image string, env map[string]string, command string, timeout time.Duration) (string, error) {
		return "", nil
	}

	if err := RunFromConfigFile(configPath); err != nil {
		t.Fatalf("RunFromConfigFile returned error: %v", err)
	}
}

func TestRunRequiresImageWhenChecksProvided(t *testing.T) {
	t.Setenv("TEST_IMAGE", "")
	err := Run(Config{Paths: []string{"/app"}})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "image is required") {
		t.Fatalf("expected image required error, got %v", err)
	}
}

func TestRunExternalStorageIncludesConfig(t *testing.T) {
	oldRunPathCheckFn := runPathCheckFn
	t.Cleanup(func() {
		runPathCheckFn = oldRunPathCheckFn
	})

	paths := make([]string, 0)
	runPathCheckFn = func(image, path string, cfg *ContainerConfig, timeout time.Duration) error {
		paths = append(paths, path)
		return nil
	}

	err := Run(Config{Image: "docker.io/library/alpine:3.22", ExternalStorage: []string{"/mnt/external"}})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(paths) != 2 || paths[0] != "/mnt/external" || paths[1] != "/config" {
		t.Fatalf("unexpected path checks: %#v", paths)
	}
}

func TestRunCommandsSuccess(t *testing.T) {
	oldRunCommandFn := runCommandFn
	t.Cleanup(func() {
		runCommandFn = oldRunCommandFn
	})

	calledCommands := make([]string, 0)
	runCommandFn = func(image string, env map[string]string, command string, timeout time.Duration) (string, error) {
		calledCommands = append(calledCommands, command)
		if image != "docker.io/library/alpine:3.22" {
			t.Fatalf("expected image %q, got %q", "docker.io/library/alpine:3.22", image)
		}
		if env["TEST_VAR"] != "yes" {
			t.Fatalf("expected env TEST_VAR=yes, got %#v", env)
		}
		if timeout != defaultTimeoutSeconds*time.Second {
			t.Fatalf("expected default timeout %v, got %v", defaultTimeoutSeconds*time.Second, timeout)
		}
		return "", nil
	}

	err := Run(Config{
		Image:    "docker.io/library/alpine:3.22",
		Env:      map[string]string{"TEST_VAR": "yes"},
		Commands: []string{"echo ok", "test -d /tmp"},
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if len(calledCommands) != 2 {
		t.Fatalf("expected 2 commands to run, got %d", len(calledCommands))
	}
}

func TestRunCommandsFailure(t *testing.T) {
	oldRunCommandFn := runCommandFn
	t.Cleanup(func() {
		runCommandFn = oldRunCommandFn
	})

	runCommandFn = func(image string, env map[string]string, command string, timeout time.Duration) (string, error) {
		if command == "failing command" {
			return "boom", errors.New("exit status 1")
		}
		return "", nil
	}

	err := Run(Config{Image: "docker.io/library/alpine:3.22", Commands: []string{"failing command"}})
	if err == nil {
		t.Fatal("expected command failure")
	}
	if !strings.Contains(err.Error(), "command check failed") {
		t.Fatalf("expected command check failure, got %v", err)
	}
	if !strings.Contains(err.Error(), "output: boom") {
		t.Fatalf("expected command output in failure, got %v", err)
	}
}

func TestRunCommandsPrintsOutput(t *testing.T) {
	oldRunCommandFn := runCommandFn
	t.Cleanup(func() {
		runCommandFn = oldRunCommandFn
	})

	runCommandFn = func(image string, env map[string]string, command string, timeout time.Duration) (string, error) {
		return "hello from command\n", nil
	}

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = oldStdout
		if closeErr := r.Close(); closeErr != nil {
			t.Errorf("failed to close stdout reader: %v", closeErr)
		}
	})

	err = Run(Config{Image: "docker.io/library/alpine:3.22", Commands: []string{"echo hello"}})
	_ = w.Close()
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	outputBytes, readErr := io.ReadAll(r)
	if readErr != nil {
		t.Fatalf("failed to read stdout: %v", readErr)
	}
	if got := string(outputBytes); !strings.Contains(got, "hello from command") {
		t.Fatalf("expected command output to be printed, got %q", got)
	}
}

func TestRunCommandsRejectsEmptyCommand(t *testing.T) {
	err := Run(Config{Image: "docker.io/library/alpine:3.22", Commands: []string{"  "}})
	if err == nil {
		t.Fatal("expected command validation failure")
	}
	if !strings.Contains(err.Error(), "command is required") {
		t.Fatalf("expected empty command validation failure, got %v", err)
	}
}

func TestRunCommandReturnsContainerOutput(t *testing.T) {
	oldRunContainerFn := runContainerFn
	oldTerminateContainerFn := terminateContainerFn
	oldContainerExitCodeFn := containerExitCodeFn
	oldContainerOutputFn := containerOutputFn
	t.Cleanup(func() {
		runContainerFn = oldRunContainerFn
		terminateContainerFn = oldTerminateContainerFn
		containerExitCodeFn = oldContainerExitCodeFn
		containerOutputFn = oldContainerOutputFn
	})

	runContainerFn = func(ctx context.Context, image string, opts ...testcontainers.ContainerCustomizer) (testcontainers.Container, error) {
		return nil, nil
	}
	terminateContainerFn = func(ctx context.Context, c testcontainers.Container) error { return nil }
	containerExitCodeFn = func(ctx context.Context, c testcontainers.Container) (int, error) { return 0, nil }
	containerOutputFn = func(ctx context.Context, c testcontainers.Container) (string, error) {
		return "container output\n", nil
	}

	output, err := runCommand("docker.io/library/alpine:3.22", nil, "echo hello", time.Second)
	if err != nil {
		t.Fatalf("runCommand returned error: %v", err)
	}
	if output != "container output\n" {
		t.Fatalf("expected output %q, got %q", "container output\n", output)
	}
}

func TestGetTestImage(t *testing.T) {
	t.Setenv("TEST_IMAGE", "")
	if got := GetTestImage("docker.io/library/alpine:3.22"); got != "docker.io/library/alpine:3.22" {
		t.Fatalf("expected default image, got %q", got)
	}

	t.Setenv("TEST_IMAGE", "ghcr.io/example/app:latest")
	if got := GetTestImage("docker.io/library/alpine:3.22"); got != "ghcr.io/example/app:latest" {
		t.Fatalf("expected env image override, got %q", got)
	}
}

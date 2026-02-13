package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/trueforge-org/forgetool/pkg/helper"
)

func TestMainHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_MAIN_HELPER") != "1" {
		return
	}

	if p := os.Getenv("FORGETOOL_TEST_CACHE_DIR"); p != "" {
		helper.CacheDir = p
	}
	os.Args = []string{"forgetool"}
	main()
	os.Exit(0)
}

func TestRunCommandHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_RUNCOMMAND_HELPER") != "1" {
		return
	}

	os.Args = []string{"forgetool", "definitely-not-a-real-command"}
	runCommand()
	os.Exit(0)
}

func TestMain_ExecutesToCacheStage(t *testing.T) {
	cacheDir := t.TempDir()

	cmd := exec.Command(os.Args[0], "-test.run=TestMainHelperProcess")
	cmd.Env = append(
		os.Environ(),
		"GO_WANT_MAIN_HELPER=1",
		"FORGETOOL_TEST_CACHE_DIR="+cacheDir,
		"DEBUG=1",
	)
	cmd.Dir = t.TempDir()
	_ = cmd.Run()

	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		t.Fatalf("failed to read cache dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("expected main to populate cache directory")
	}

	foundFile := false
	err = filepath.WalkDir(cacheDir, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			foundFile = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk cache dir failed: %v", err)
	}
	if !foundFile {
		t.Fatalf("expected cache to contain extracted files")
	}
}

func TestParseLogLevel_AllCases(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected zerolog.Level
	}{
		{name: "trace", input: "trace", expected: zerolog.TraceLevel},
		{name: "debug", input: "debug", expected: zerolog.DebugLevel},
		{name: "warn", input: "warn", expected: zerolog.WarnLevel},
		{name: "error", input: "error", expected: zerolog.ErrorLevel},
		{name: "fatal", input: "fatal", expected: zerolog.FatalLevel},
		{name: "panic", input: "panic", expected: zerolog.PanicLevel},
		{name: "info", input: "info", expected: zerolog.InfoLevel},
		{name: "default", input: "unknown", expected: zerolog.InfoLevel},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseLogLevel(tc.input)
			if got != tc.expected {
				t.Fatalf("parseLogLevel(%q) = %v, expected %v", tc.input, got, tc.expected)
			}
		})
	}
}

func TestRunCommand_ExitsOnExecuteError(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestRunCommandHelperProcess")
	cmd.Env = append(os.Environ(), "GO_WANT_RUNCOMMAND_HELPER=1")

	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected runCommand helper to exit with error")
	}
}

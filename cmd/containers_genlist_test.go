package cmd

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trueforge-org/forgetool/v4/pkg/containers/website"
)

func TestRunContainersGenListErrorPaths(t *testing.T) {
	oldWalk := containersGenListWalk
	oldFactory := containersGenListOptionsFactory
	oldGet := containersGenListGetData
	oldWrite := containersGenListWrite
	t.Cleanup(func() {
		containersGenListWalk = oldWalk
		containersGenListOptionsFactory = oldFactory
		containersGenListGetData = oldGet
		containersGenListWrite = oldWrite
	})

	containersGenListOptionsFactory = func() *website.ContainerListOptions {
		return &website.ContainerListOptions{OutputPath: "ignored"}
	}
	containersGenListGetData = func(_ *website.ContainerListOptions) fs.WalkDirFunc {
		return func(string, fs.DirEntry, error) error { return nil }
	}

	walkErr := errors.New("walk failed")
	containersGenListWalk = func([]string, fs.WalkDirFunc) error { return walkErr }
	if err := runContainersGenList([]string{"./apps"}); err == nil || !strings.Contains(err.Error(), "failed to generate container list") {
		t.Fatalf("expected wrapped walk error, got %v", err)
	}

	containersGenListWalk = func([]string, fs.WalkDirFunc) error { return nil }
	writeErr := errors.New("write failed")
	containersGenListWrite = func(_ *website.ContainerListOptions) error { return writeErr }
	if err := runContainersGenList([]string{"./apps"}); err == nil || !strings.Contains(err.Error(), "failed to write container list") {
		t.Fatalf("expected wrapped write error, got %v", err)
	}
}

func TestRunContainersGenListSuccess(t *testing.T) {
	oldWalk := containersGenListWalk
	oldFactory := containersGenListOptionsFactory
	oldGet := containersGenListGetData
	oldWrite := containersGenListWrite
	t.Cleanup(func() {
		containersGenListWalk = oldWalk
		containersGenListOptionsFactory = oldFactory
		containersGenListGetData = oldGet
		containersGenListWrite = oldWrite
	})

	calledWalk := false
	calledWrite := false
	containersGenListOptionsFactory = func() *website.ContainerListOptions {
		return &website.ContainerListOptions{OutputPath: "ignored"}
	}
	containersGenListGetData = func(_ *website.ContainerListOptions) fs.WalkDirFunc {
		return func(string, fs.DirEntry, error) error { return nil }
	}
	containersGenListWalk = func([]string, fs.WalkDirFunc) error {
		calledWalk = true
		return nil
	}
	containersGenListWrite = func(_ *website.ContainerListOptions) error {
		calledWrite = true
		return nil
	}

	if err := runContainersGenList([]string{"./apps"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !calledWalk || !calledWrite {
		t.Fatalf("expected both walk and write to be called")
	}
}

func TestGenContainersListCommandRunCallsRunner(t *testing.T) {
	oldRunner := containersGenListRunner
	t.Cleanup(func() { containersGenListRunner = oldRunner })

	called := false
	containersGenListRunner = func(args []string) error {
		called = true
		if len(args) != 1 || args[0] != "./apps" {
			t.Fatalf("unexpected args: %#v", args)
		}
		return nil
	}

	genContainersListCmd.Run(genContainersListCmd, []string{"./apps"})
	if !called {
		t.Fatalf("expected command Run to call runner")
	}
}

func TestGenContainersListCommandRunCallsOnError(t *testing.T) {
	oldRunner := containersGenListRunner
	oldOnError := containersGenListOnError
	t.Cleanup(func() {
		containersGenListRunner = oldRunner
		containersGenListOnError = oldOnError
	})

	want := errors.New("boom")
	containersGenListRunner = func([]string) error { return want }
	called := false
	containersGenListOnError = func(err error) {
		called = true
		if !errors.Is(err, want) {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	genContainersListCmd.Run(genContainersListCmd, []string{"./apps"})
	if !called {
		t.Fatalf("expected command Run to call error handler")
	}
}

func TestRunContainersGenListWithFixture(t *testing.T) {
	oldWalk := containersGenListWalk
	oldFactory := containersGenListOptionsFactory
	oldGet := containersGenListGetData
	oldWrite := containersGenListWrite
	t.Cleanup(func() {
		containersGenListWalk = oldWalk
		containersGenListOptionsFactory = oldFactory
		containersGenListGetData = oldGet
		containersGenListWrite = oldWrite
	})

	containersGenListWalk = defaultContainersGenListWalk
	containersGenListGetData = defaultContainersGenListGetData
	containersGenListWrite = defaultContainersGenListWrite

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp failed: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	containersGenListOptionsFactory = func() *website.ContainerListOptions {
		return &website.ContainerListOptions{OutputPath: filepath.Join(tmp, "containers.json")}
	}

	fixtureRoot, err := filepath.Abs(filepath.Join(oldWD, "..", "testdata", "cmd", "containers"))
	if err != nil {
		t.Fatalf("fixture abs path failed: %v", err)
	}

	if err := runContainersGenList([]string{fixtureRoot}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmp, "containers.json"))
	if err != nil {
		t.Fatalf("read containers.json failed: %v", err)
	}

	var out website.ContainerList
	if err := json.Unmarshal(content, &out); err != nil {
		t.Fatalf("parse containers.json failed: %v", err)
	}
	if out.TotalCount < 1 {
		t.Fatalf("expected at least one container in output")
	}
	if len(out.Containers) < 1 {
		t.Fatalf("expected at least one container entry")
	}
	if out.Containers[0].Name != "sample-app" {
		t.Fatalf("expected container name sample-app, got %s", out.Containers[0].Name)
	}
}

func TestDefaultContainersGenListWrappers(t *testing.T) {
	opts := defaultContainersGenListOptionsFactory()
	if opts.OutputPath != "./containers.json" {
		t.Fatalf("unexpected default output path: %s", opts.OutputPath)
	}

	if defaultContainersGenListGetData(opts) == nil {
		t.Fatalf("expected container data walker function")
	}

	opts.OutputPath = filepath.Join(t.TempDir(), "containers.json")
	if err := defaultContainersGenListWrite(opts); err == nil {
		t.Fatalf("expected write error when container list is nil")
	}
}

func TestDefaultContainersGenListWalk_DefaultPath(t *testing.T) {
	called := false
	walkFn := func(path string, d fs.DirEntry, err error) error {
		called = true
		return nil
	}
	// Empty paths should use ./apps which doesn't exist, so it should error
	err := defaultContainersGenListWalk(nil, walkFn)
	if err == nil {
		// Only fail if neither errored nor walked something
		if !called {
			t.Fatalf("expected either an error or the walk function to be called")
		}
	}
}

package cmd

import (
	"errors"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/trueforge-org/forgetool/v4/pkg/helper"
)

func TestRunGenToolDocs(t *testing.T) {
	oldMkdir := genToolDocsMkdirAll
	oldMarkdown := genToolDocsMarkdown
	oldWriter := genToolDocsWriter
	t.Cleanup(func() {
		genToolDocsMkdirAll = oldMkdir
		genToolDocsMarkdown = oldMarkdown
		genToolDocsWriter = oldWriter
	})

	mkdirCalled := false
	genToolDocsMkdirAll = func(path string, _ fs.FileMode) error {
		mkdirCalled = path == helper.DocsCache
		return nil
	}
	markdownCalled := false
	genToolDocsMarkdown = func(cmd *cobra.Command, path string) error {
		markdownCalled = cmd == RootCmd && path == helper.DocsCache
		return nil
	}
	writerCalled := false
	genToolDocsWriter = func(tmpDir string, outputDir string) {
		writerCalled = tmpDir == helper.DocsCache && outputDir == filepath.Clean("./docs")
	}

	if err := runGenToolDocs(filepath.Clean("./docs")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mkdirCalled || !markdownCalled || !writerCalled {
		t.Fatalf("expected mkdir, markdown and writer to all be called")
	}

	mkdirErr := errors.New("mkdir failed")
	genToolDocsMkdirAll = func(string, fs.FileMode) error { return mkdirErr }
	if err := runGenToolDocs("./docs"); !errors.Is(err, mkdirErr) {
		t.Fatalf("expected mkdir error, got %v", err)
	}

	genToolDocsMkdirAll = func(string, fs.FileMode) error { return nil }
	markdownErr := errors.New("markdown failed")
	genToolDocsMarkdown = func(*cobra.Command, string) error { return markdownErr }
	if err := runGenToolDocs("./docs"); !errors.Is(err, markdownErr) {
		t.Fatalf("expected markdown error, got %v", err)
	}
}

func TestGenDocsCommandRunCallsRunner(t *testing.T) {
	oldRunner := genToolDocsRunner
	t.Cleanup(func() { genToolDocsRunner = oldRunner })

	called := false
	genToolDocsRunner = func(outdir string) error {
		called = true
		if outdir != "./docs" {
			t.Fatalf("unexpected outdir: %s", outdir)
		}
		return nil
	}

	gendocsCmd.Run(gendocsCmd, []string{"./docs"})

	if !called {
		t.Fatalf("expected gendocs command run to call runner")
	}
}

func TestGenDocsCommandRunCallsOnError(t *testing.T) {
	oldRunner := genToolDocsRunner
	oldOnError := genToolDocsOnError
	t.Cleanup(func() {
		genToolDocsRunner = oldRunner
		genToolDocsOnError = oldOnError
	})

	want := errors.New("boom")
	genToolDocsRunner = func(string) error { return want }
	called := false
	genToolDocsOnError = func(err error) {
		called = true
		if !errors.Is(err, want) {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	gendocsCmd.Run(gendocsCmd, []string{"./docs"})

	if !called {
		t.Fatalf("expected gendocs command run to call error handler")
	}
}

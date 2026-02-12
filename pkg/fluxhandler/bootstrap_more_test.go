package fluxhandler

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

func TestCheckGitRepo(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	if err := checkGitRepo(); err != nil {
		t.Fatalf("expected current behavior (nil error) outside git repo, got: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(td, ".git"), 0755); err != nil {
		t.Fatalf("mkdir .git failed: %v", err)
	}
	if err := checkGitRepo(); err != nil {
		t.Fatalf("expected checkGitRepo success inside git repo: %v", err)
	}
}

func TestSetupFluxCDAndRepositoriesErrorPaths(t *testing.T) {
	ctx := context.Background()
	if err := setupFluxCD(ctx, filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatalf("expected setupFluxCD to fail on missing path")
	}
	if err := setupRepositories(ctx, filepath.Join(t.TempDir(), "missing-repos")); err == nil {
		t.Fatalf("expected setupRepositories to fail on missing files")
	}
}

func TestBootstrapFluxCDErrorAndNoopFluxBootstrap(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	oldEnv := helper.TalEnv
	helper.TalEnv = map[string]string{}
	defer func() { helper.TalEnv = oldEnv }()

	// No repository configured -> should no-op
	FluxBootstrap(context.Background())

	// bootstrapFluxCD should fail outside git repo
	if err := bootstrapFluxCD(context.Background()); err == nil {
		t.Fatalf("expected bootstrapFluxCD to fail outside git repo")
	}
}

package kubectlcmds

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKubectlApplyKustomize_NonExistingPathMessage(t *testing.T) {
	ctx := context.TODO()
	p := filepath.Join("/tmp", "nonexistent-dir-for-test-82734")
	err := KubectlApplyKustomize(ctx, p)
	if err == nil {
		t.Fatal("expected error for non-existing path, got nil")
	}
	if !strings.Contains(err.Error(), "path does not exist") {
		t.Fatalf("expected 'path does not exist' in error, got: %s", err.Error())
	}
}

func TestKubectlApplyKustomize_ExistingDirNoKustomization(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kustomize-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ctx := context.TODO()
	err = KubectlApplyKustomize(ctx, tmpDir)
	if err == nil {
		t.Fatal("expected error for directory without kustomization.yaml, got nil")
	}
	if !strings.Contains(err.Error(), "kustomize") {
		t.Fatalf("expected kustomize-related error, got: %s", err.Error())
	}
}

func TestKubectlApplyKustomize_FilePathUsesParentDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kustomize-file-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, "somefile.yaml")
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	ctx := context.TODO()
	err = KubectlApplyKustomize(ctx, tmpFile)
	if err == nil {
		t.Fatal("expected error for file in directory without kustomization.yaml, got nil")
	}
	if !strings.Contains(err.Error(), "kustomize") {
		t.Fatalf("expected kustomize-related error, got: %s", err.Error())
	}
}

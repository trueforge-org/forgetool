package kubectlcmds

import (
	"context"
	"path/filepath"
	"testing"
)

func TestKubectlApply_NonExistingFile(t *testing.T) {
	ctx := context.TODO()
	p := filepath.Join("/tmp", "this-file-should-not-exist-281937.yaml")
	if err := KubectlApply(ctx, p); err == nil {
		t.Fatalf("expected error for non-existing file, got nil")
	}
}

func TestKubectlApplyKustomize_NonExistingPath(t *testing.T) {
	ctx := context.TODO()
	p := filepath.Join("/tmp", "this-dir-should-not-exist-281937")
	if err := KubectlApplyKustomize(ctx, p); err == nil {
		t.Fatalf("expected error for non-existing kustomize path, got nil")
	}
}

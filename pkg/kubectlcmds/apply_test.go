package kubectlcmds

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestKubectlApply_NonExistingFileMessage(t *testing.T) {
	ctx := context.TODO()
	p := "/tmp/nonexistent-file-for-test-82734.yaml"
	err := KubectlApply(ctx, p)
	if err == nil {
		t.Fatal("expected error for non-existing file, got nil")
	}
	if !strings.Contains(err.Error(), "file does not exist") {
		t.Fatalf("expected 'file does not exist' in error, got: %s", err.Error())
	}
}

func TestKubectlApply_ExistingFileNoKubeconfig(t *testing.T) {
	forceNoKubeconfigEnv(t)

	tmpFile, err := os.CreateTemp("", "kubectl-apply-test-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test\n")); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	ctx := context.TODO()
	err = KubectlApply(ctx, tmpFile.Name())
	if err == nil {
		t.Fatal("expected error when no kubeconfig is available, got nil")
	}
	if !strings.Contains(err.Error(), "kubeconfig") {
		t.Fatalf("expected kubeconfig-related error, got: %s", err.Error())
	}
}

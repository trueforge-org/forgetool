package kubectlcmds

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKubectlApply_NonExistingFileMessage(t *testing.T) {
	ctx := context.TODO()
	p := filepath.Join("/tmp", "nonexistent-file-for-test-82734.yaml")
	err := KubectlApply(ctx, p)
	if err == nil {
		t.Fatal("expected error for non-existing file, got nil")
	}
	if !strings.Contains(err.Error(), "file does not exist") {
		t.Fatalf("expected 'file does not exist' in error, got: %s", err.Error())
	}
}

func TestKubectlApply_ExistingFileNoKubeconfig(t *testing.T) {
	// Create a temporary YAML file so the file-existence check passes,
	// but the kubeconfig loading should fail since no cluster is available.
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
	// Create a temporary directory without a kustomization.yaml.
	// The kustomize run should fail.
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
	// Create a temp dir with a file inside. When a file path is given,
	// KubectlApplyKustomize uses its parent directory, which also lacks
	// kustomization.yaml, so it should fail at kustomize run.
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

func TestGetKubeClient_NoKubeconfig(t *testing.T) {
	_, err := getKubeClient()
	if err == nil {
		t.Fatal("expected error when no kubeconfig is available, got nil")
	}
	if !strings.Contains(err.Error(), "kubeconfig") {
		t.Fatalf("expected kubeconfig-related error, got: %s", err.Error())
	}
}

func TestGetClientset_NoKubeconfig(t *testing.T) {
	_, err := GetClientset()
	if err == nil {
		t.Fatal("expected error when no kubeconfig is available, got nil")
	}
}

func TestCheckStatus_NoKubeconfig(t *testing.T) {
	// With no kubeconfig and no in-cluster config, CheckStatus should
	// fail immediately when trying to load the kubeconfig.
	err := CheckStatus([]string{"some-pod"}, nil, 1)
	if err == nil {
		t.Fatal("expected error when no kubeconfig is available, got nil")
	}
	if !strings.Contains(err.Error(), "kubeconfig") {
		t.Fatalf("expected kubeconfig-related error, got: %s", err.Error())
	}
}

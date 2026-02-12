package fluxhandler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileExists(t *testing.T) {
	td := t.TempDir()
	f := filepath.Join(td, "a.yaml")
	if fileExists(f) {
		t.Fatalf("expected missing file to return false")
	}
	if err := os.WriteFile(f, []byte("x"), 0644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
	if !fileExists(f) {
		t.Fatalf("expected existing file to return true")
	}
	if fileExists(td) {
		t.Fatalf("expected directory to return false")
	}
}

func TestCreateKsYaml(t *testing.T) {
	td := t.TempDir()
	if err := createKsYaml(td, "parent"); err != nil {
		t.Fatalf("createKsYaml failed: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(td, "ks.yaml"))
	if err != nil {
		t.Fatalf("read ks.yaml failed: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, "name: parent") {
		t.Fatalf("expected ks yaml to contain parent name: %s", s)
	}
	if !strings.Contains(s, "path: "+filepath.ToSlash(td)+"/app") {
		t.Fatalf("expected ks yaml to contain normalized path: %s", s)
	}
}

func TestCreateOrUpdateKustomizationYaml(t *testing.T) {
	td := t.TempDir()
	if err := os.WriteFile(filepath.Join(td, "namespace.yaml"), []byte("apiVersion: v1\nkind: Namespace\n"), 0644); err != nil {
		t.Fatalf("write namespace failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(td, "deployment.yaml"), []byte("kind: Deployment\n"), 0644); err != nil {
		t.Fatalf("write deployment failed: %v", err)
	}
	sub := filepath.Join(td, "sub")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatalf("mkdir sub failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sub, "ks.yaml"), []byte("kind: Kustomization\n"), 0644); err != nil {
		t.Fatalf("write sub ks failed: %v", err)
	}

	if err := createOrUpdateKustomizationYaml(td); err != nil {
		t.Fatalf("createOrUpdateKustomizationYaml failed: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(td, "kustomization.yaml"))
	if err != nil {
		t.Fatalf("read kustomization.yaml failed: %v", err)
	}
	content := string(b)
	if !strings.Contains(content, "- namespace.yaml") || !strings.Contains(content, "- deployment.yaml") {
		t.Fatalf("expected resources to be included, got: %s", content)
	}
	if !strings.Contains(content, "- sub/ks.yaml") {
		t.Fatalf("expected directory with ks.yaml to be added as sub/ks.yaml, got: %s", content)
	}

	if err := createOrUpdateKustomizationYaml(td); err != nil {
		t.Fatalf("second createOrUpdateKustomizationYaml failed: %v", err)
	}
	b2, _ := os.ReadFile(filepath.Join(td, "kustomization.yaml"))
	if strings.Count(string(b2), "namespace.yaml") != 1 {
		t.Fatalf("expected no duplicates after rerun, got: %s", string(b2))
	}
}

func TestProcessDirectory(t *testing.T) {
	td := t.TempDir()
	appParent := filepath.Join(td, "mychart")
	if err := os.MkdirAll(filepath.Join(appParent, "app"), 0755); err != nil {
		t.Fatalf("mkdir app path failed: %v", err)
	}
	other := filepath.Join(td, "other")
	if err := os.MkdirAll(other, 0755); err != nil {
		t.Fatalf("mkdir other failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(other, "service.yaml"), []byte("kind: Service\n"), 0644); err != nil {
		t.Fatalf("write service failed: %v", err)
	}

	if err := ProcessDirectory(td); err != nil {
		t.Fatalf("ProcessDirectory failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(appParent, "ks.yaml")); err != nil {
		t.Fatalf("expected ks.yaml in app parent: %v", err)
	}
	if _, err := os.Stat(filepath.Join(other, "kustomization.yaml")); err != nil {
		t.Fatalf("expected kustomization.yaml in other folder: %v", err)
	}
}

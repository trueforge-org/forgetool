package fluxhandler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadHelmRepo_InvalidYAML(t *testing.T) {
	td := t.TempDir()
	p := filepath.Join(td, "invalid.yaml")
	os.WriteFile(p, []byte("invalid: [yaml: broken"), 0644)

	_, err := LoadHelmRepo(p)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestLoadHelmRepo_NonExistentFile(t *testing.T) {
	_, err := LoadHelmRepo("/nonexistent/file.yaml")
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
}

func TestLoadHelmRepo_ValidYAML(t *testing.T) {
	td := t.TempDir()
	p := filepath.Join(td, "repo.yaml")
	content := `metadata:
  name: test-repo
  namespace: flux-system
spec:
  interval: "1h"
  url: "https://charts.example.com"
`
	os.WriteFile(p, []byte(content), 0644)

	repo, err := LoadHelmRepo(p)
	if err != nil {
		t.Fatalf("LoadHelmRepo failed: %v", err)
	}
	if repo.Metadata.Name != "test-repo" {
		t.Fatalf("expected name 'test-repo', got %q", repo.Metadata.Name)
	}
	if repo.Spec.URL != "https://charts.example.com" {
		t.Fatalf("expected URL 'https://charts.example.com', got %q", repo.Spec.URL)
	}
}

func TestLoadAllHelmRepos_SkipsKustomize(t *testing.T) {
	td := t.TempDir()
	// Create a kustomize.yaml that should be skipped
	os.WriteFile(filepath.Join(td, "kustomize.yaml"), []byte("apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization"), 0644)
	// Create a valid repo
	os.WriteFile(filepath.Join(td, "myrepo.yaml"), []byte("metadata:\n  name: myrepo\nspec:\n  url: https://example.com"), 0644)

	repos, err := LoadAllHelmRepos(td)
	if err != nil {
		t.Fatalf("LoadAllHelmRepos failed: %v", err)
	}
	if _, ok := repos["myrepo"]; !ok {
		t.Fatal("expected 'myrepo' to be loaded")
	}
	// kustomize.yaml should have been skipped
	if len(repos) != 1 {
		t.Fatalf("expected 1 repo (skipping kustomize), got %d", len(repos))
	}
}

func TestLoadAllHelmRepos_SkipsDirectories(t *testing.T) {
	td := t.TempDir()
	os.MkdirAll(filepath.Join(td, "subdir"), 0755)
	os.WriteFile(filepath.Join(td, "repo.yaml"), []byte("metadata:\n  name: repo1\nspec:\n  url: https://example.com"), 0644)

	repos, err := LoadAllHelmRepos(td)
	if err != nil {
		t.Fatalf("LoadAllHelmRepos failed: %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(repos))
	}
}

func TestLoadAllHelmRepos_SkipsNonYaml(t *testing.T) {
	td := t.TempDir()
	os.WriteFile(filepath.Join(td, "readme.txt"), []byte("not yaml"), 0644)
	os.WriteFile(filepath.Join(td, "repo.yaml"), []byte("metadata:\n  name: repo1\nspec:\n  url: https://example.com"), 0644)

	repos, err := LoadAllHelmRepos(td)
	if err != nil {
		t.Fatalf("LoadAllHelmRepos failed: %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(repos))
	}
}

func TestLoadAllHelmRepos_InvalidDir(t *testing.T) {
	_, err := LoadAllHelmRepos("/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for non-existent directory, got nil")
	}
}

func TestLoadAllHelmRepos_InvalidRepoSkipped(t *testing.T) {
	td := t.TempDir()
	os.WriteFile(filepath.Join(td, "bad.yaml"), []byte("invalid: [yaml"), 0644)
	os.WriteFile(filepath.Join(td, "good.yaml"), []byte("metadata:\n  name: goodrepo\nspec:\n  url: https://example.com"), 0644)

	repos, err := LoadAllHelmRepos(td)
	if err != nil {
		t.Fatalf("LoadAllHelmRepos failed: %v", err)
	}
	// Bad yaml should be logged and skipped, good repo should load
	if _, ok := repos["goodrepo"]; !ok {
		t.Fatal("expected 'goodrepo' to be loaded despite bad.yaml")
	}
}

func TestLoadHelmRelease_ValidFile(t *testing.T) {
	td := t.TempDir()
	p := filepath.Join(td, "helm-release.yaml")
	content := `apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: my-release
  namespace: default
spec:
  interval: "5m"
  chart:
    spec:
      chart: my-chart
      version: "1.0.0"
      sourceRef:
        kind: HelmRepository
        name: my-repo
        namespace: flux-system
  releaseName: custom-name
  values:
    key: value
`
	os.WriteFile(p, []byte(content), 0644)

	hr, err := LoadHelmRelease(p)
	if err != nil {
		t.Fatalf("LoadHelmRelease failed: %v", err)
	}
	if hr.Metadata.Name != "my-release" {
		t.Fatalf("expected name 'my-release', got %q", hr.Metadata.Name)
	}
	if hr.Spec.ReleaseName != "custom-name" {
		t.Fatalf("expected releaseName 'custom-name', got %q", hr.Spec.ReleaseName)
	}
	if hr.Spec.Chart.Spec.Chart != "my-chart" {
		t.Fatalf("expected chart 'my-chart', got %q", hr.Spec.Chart.Spec.Chart)
	}
	if hr.Spec.Values["key"] != "value" {
		t.Fatal("expected values to contain key=value")
	}
}

func TestLoadHelmRelease_NilValues(t *testing.T) {
	td := t.TempDir()
	p := filepath.Join(td, "helm-release.yaml")
	content := `apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: no-values
spec:
  chart:
    spec:
      chart: my-chart
`
	os.WriteFile(p, []byte(content), 0644)

	hr, err := LoadHelmRelease(p)
	if err != nil {
		t.Fatalf("LoadHelmRelease failed: %v", err)
	}
	if hr.Spec.Values == nil {
		t.Fatal("expected Values to be initialized (not nil)")
	}
}

func TestReplacePlaceholder_ReplaceAll(t *testing.T) {
	td := t.TempDir()
	p := filepath.Join(td, "test.yaml")
	os.WriteFile(p, []byte("hello PLACEHOLDER world PLACEHOLDER"), 0644)

	if err := ReplacePlaceholder(p, "PLACEHOLDER", "REPLACED"); err != nil {
		t.Fatalf("ReplacePlaceholder failed: %v", err)
	}

	data, _ := os.ReadFile(p)
	if !strings.Contains(string(data), "REPLACED") || strings.Contains(string(data), "PLACEHOLDER") {
		t.Fatalf("expected all PLACEHOLDERs replaced, got: %s", string(data))
	}
}

func TestReplacePlaceholder_FileNotFound(t *testing.T) {
	err := ReplacePlaceholder("/nonexistent/file.yaml", "x", "y")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestGetKnownHostsEntry_Github(t *testing.T) {
	entry := getKnownHostsEntry("github.com")
	if !strings.Contains(entry, "github.com") {
		t.Fatalf("expected github.com in known hosts entry, got %q", entry)
	}
}

func TestGetKnownHostsEntry_Custom(t *testing.T) {
	entry := getKnownHostsEntry("gitlab.com")
	if !strings.Contains(entry, "gitlab.com") {
		t.Fatalf("expected gitlab.com in known hosts entry, got %q", entry)
	}
}

func TestGenerateKnownHostsEntry(t *testing.T) {
	entry := generateKnownHostsEntry("custom.example.com")
	if !strings.HasPrefix(entry, "custom.example.com") {
		t.Fatalf("expected entry to start with custom.example.com, got %q", entry)
	}
}

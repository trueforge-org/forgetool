package fluxhandler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadHelmRepo(t *testing.T) {
	td := t.TempDir()
	f := filepath.Join(td, "repo.yaml")
	content := `metadata:
  name: stable
  namespace: flux-system
spec:
  interval: 10m
  url: https://example.com/charts
`
	if err := os.WriteFile(f, []byte(content), 0644); err != nil {
		t.Fatalf("write repo file failed: %v", err)
	}
	r, err := LoadHelmRepo(f)
	if err != nil {
		t.Fatalf("LoadHelmRepo failed: %v", err)
	}
	if r.Metadata.Name != "stable" || r.Spec.URL == "" {
		t.Fatalf("unexpected repo content: %+v", r)
	}
}

func TestLoadAllHelmRepos(t *testing.T) {
	td := t.TempDir()
	good := `metadata:
  name: stable
spec:
  url: https://example.com/charts
`
	if err := os.WriteFile(filepath.Join(td, "repo1.yaml"), []byte(good), 0644); err != nil {
		t.Fatalf("write good yaml failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(td, "bad.yaml"), []byte("metadata: ["), 0644); err != nil {
		t.Fatalf("write bad yaml failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(td, "kustomize.yaml"), []byte(good), 0644); err != nil {
		t.Fatalf("write kustomize yaml failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(td, "notes.txt"), []byte("ignore"), 0644); err != nil {
		t.Fatalf("write txt failed: %v", err)
	}

	repos, err := LoadAllHelmRepos(td)
	if err != nil {
		t.Fatalf("LoadAllHelmRepos failed: %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("expected 1 valid repo, got %d", len(repos))
	}
	if repos["stable"] == nil {
		t.Fatalf("expected key 'stable' to be present")
	}
}

package fluxhandler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadHelmRelease(t *testing.T) {
	td := t.TempDir()
	path := filepath.Join(td, "helm-release.yaml")
	yaml := `apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: myrel
  namespace: ns
spec:
  chart:
    spec:
      chart: app
      version: 1.0.0
      sourceRef:
        name: myrepo
`
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	hr, err := LoadHelmRelease(path)
	if err != nil {
		t.Fatalf("LoadHelmRelease failed: %v", err)
	}
	if hr.Metadata.Name != "myrel" || hr.Metadata.Namespace != "ns" {
		t.Fatalf("unexpected metadata: %+v", hr.Metadata)
	}
	if hr.Spec.Values == nil {
		t.Fatalf("expected values map initialized")
	}
}

func TestLoadHelmReleaseErrors(t *testing.T) {
	if _, err := LoadHelmRelease(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Fatalf("expected read error for missing file")
	}

	td := t.TempDir()
	bad := filepath.Join(td, "bad.yaml")
	if err := os.WriteFile(bad, []byte("spec: ["), 0644); err != nil {
		t.Fatalf("write bad file failed: %v", err)
	}
	if _, err := LoadHelmRelease(bad); err == nil {
		t.Fatalf("expected yaml unmarshal error")
	}
}

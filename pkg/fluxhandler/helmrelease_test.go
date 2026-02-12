package fluxhandler

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoadHelmReleaseNilValues(t *testing.T) {
	td := t.TempDir()
	path := filepath.Join(td, "hr.yaml")
	content := `apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: test
spec:
  chart:
    spec:
      chart: mychart
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	hr, err := LoadHelmRelease(path)
	if err != nil {
		t.Fatalf("LoadHelmRelease: %v", err)
	}
	if hr.Spec.Values == nil {
		t.Fatal("expected Values map to be initialized, got nil")
	}
	if len(hr.Spec.Values) != 0 {
		t.Fatalf("expected empty Values map, got %d entries", len(hr.Spec.Values))
	}
}

func TestLoadHelmReleaseWithReleaseName(t *testing.T) {
	td := t.TempDir()
	path := filepath.Join(td, "hr.yaml")
	content := `apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: original
  namespace: default
spec:
  releaseName: custom-release
  chart:
    spec:
      chart: mychart
      version: 2.0.0
      sourceRef:
        kind: HelmRepository
        name: myrepo
        namespace: flux-system
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	hr, err := LoadHelmRelease(path)
	if err != nil {
		t.Fatalf("LoadHelmRelease: %v", err)
	}
	if hr.Spec.ReleaseName != "custom-release" {
		t.Fatalf("expected releaseName 'custom-release', got %q", hr.Spec.ReleaseName)
	}
	if hr.Metadata.Name != "original" {
		t.Fatalf("expected metadata name 'original', got %q", hr.Metadata.Name)
	}
	if hr.Spec.Chart.Spec.SourceRef.Kind != "HelmRepository" {
		t.Fatalf("expected sourceRef kind 'HelmRepository', got %q", hr.Spec.Chart.Spec.SourceRef.Kind)
	}
	if hr.Spec.Chart.Spec.SourceRef.Namespace != "flux-system" {
		t.Fatalf("expected sourceRef namespace 'flux-system', got %q", hr.Spec.Chart.Spec.SourceRef.Namespace)
	}
	if hr.Spec.Chart.Spec.Version != "2.0.0" {
		t.Fatalf("expected chart version '2.0.0', got %q", hr.Spec.Chart.Spec.Version)
	}
}

func TestLoadHelmReleaseFileNotFound(t *testing.T) {
	_, err := LoadHelmRelease("/nonexistent/path/to/file.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
	if got := err.Error(); len(got) == 0 {
		t.Fatal("expected non-empty error message")
	}
}

func TestLoadHelmReleaseInvalidYAML(t *testing.T) {
	td := t.TempDir()
	path := filepath.Join(td, "bad.yaml")
	// Tabs are invalid in YAML for indentation; use content that cannot be unmarshalled into HelmRelease
	content := []byte("{{invalid yaml content}}")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	_, err := LoadHelmRelease(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestHelmReleaseSerializationRoundTrip(t *testing.T) {
	original := HelmRelease{
		APIVersion: "helm.toolkit.fluxcd.io/v2",
		Kind:       "HelmRelease",
		Metadata: Metadata{
			Name:      "roundtrip",
			Namespace: "test-ns",
		},
		Spec: Spec{
			Interval:    "5m",
			ReleaseName: "my-release",
			Chart: Chart{
				Spec: ChartSpec{
					Chart:   "nginx",
					Version: "1.2.3",
					SourceRef: SourceRef{
						Kind:      "HelmRepository",
						Name:      "bitnami",
						Namespace: "flux-system",
					},
				},
			},
			Values: map[string]interface{}{
				"replicaCount": 3,
				"image":        "nginx:latest",
			},
		},
	}

	data, err := yaml.Marshal(&original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var restored HelmRelease
	if err := yaml.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if restored.APIVersion != original.APIVersion {
		t.Fatalf("apiVersion mismatch: got %q, want %q", restored.APIVersion, original.APIVersion)
	}
	if restored.Kind != original.Kind {
		t.Fatalf("kind mismatch: got %q, want %q", restored.Kind, original.Kind)
	}
	if restored.Metadata.Name != original.Metadata.Name {
		t.Fatalf("metadata.name mismatch: got %q, want %q", restored.Metadata.Name, original.Metadata.Name)
	}
	if restored.Metadata.Namespace != original.Metadata.Namespace {
		t.Fatalf("metadata.namespace mismatch: got %q, want %q", restored.Metadata.Namespace, original.Metadata.Namespace)
	}
	if restored.Spec.Interval != original.Spec.Interval {
		t.Fatalf("spec.interval mismatch: got %q, want %q", restored.Spec.Interval, original.Spec.Interval)
	}
	if restored.Spec.ReleaseName != original.Spec.ReleaseName {
		t.Fatalf("spec.releaseName mismatch: got %q, want %q", restored.Spec.ReleaseName, original.Spec.ReleaseName)
	}
	if restored.Spec.Chart.Spec.Chart != original.Spec.Chart.Spec.Chart {
		t.Fatalf("chart name mismatch: got %q, want %q", restored.Spec.Chart.Spec.Chart, original.Spec.Chart.Spec.Chart)
	}
	if restored.Spec.Chart.Spec.Version != original.Spec.Chart.Spec.Version {
		t.Fatalf("chart version mismatch: got %q, want %q", restored.Spec.Chart.Spec.Version, original.Spec.Chart.Spec.Version)
	}
	if restored.Spec.Chart.Spec.SourceRef.Name != original.Spec.Chart.Spec.SourceRef.Name {
		t.Fatalf("sourceRef name mismatch: got %q, want %q", restored.Spec.Chart.Spec.SourceRef.Name, original.Spec.Chart.Spec.SourceRef.Name)
	}
	if restored.Spec.Values == nil {
		t.Fatal("expected Values to be non-nil after round-trip")
	}
	if len(restored.Spec.Values) != len(original.Spec.Values) {
		t.Fatalf("values length mismatch: got %d, want %d", len(restored.Spec.Values), len(original.Spec.Values))
	}
}

func TestHelmChartStructFields(t *testing.T) {
	hc := HelmChart{
		ChartPath: "/charts/my-app",
		Retry:     true,
		Wait:      false,
	}
	if hc.ChartPath != "/charts/my-app" {
		t.Fatalf("ChartPath: got %q, want %q", hc.ChartPath, "/charts/my-app")
	}
	if !hc.Retry {
		t.Fatal("Retry: expected true")
	}
	if hc.Wait {
		t.Fatal("Wait: expected false")
	}

	hc2 := HelmChart{}
	if hc2.ChartPath != "" {
		t.Fatalf("zero-value ChartPath: got %q, want empty", hc2.ChartPath)
	}
	if hc2.Retry {
		t.Fatal("zero-value Retry: expected false")
	}
	if hc2.Wait {
		t.Fatal("zero-value Wait: expected false")
	}
}

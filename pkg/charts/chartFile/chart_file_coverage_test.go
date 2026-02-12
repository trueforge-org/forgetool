package chartFile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveToFile_ValidChart(t *testing.T) {
	h := NewHelmChart()

	// Set up valid metadata
	h.Metadata = ChartMetadata{
		APIVersion:  "v2",
		Name:        "test-chart",
		Version:     "1.0.0",
		Description: "A test chart",
		AppVersion:  "1.0",
		Icon:        "https://example.com/icon.png",
		Home:        "https://example.com",
		KubeVersion: ">=1.27.0-0",
		Maintainers: []Maintainer{
			{Name: "Test", URL: "https://example.com"},
		},
		Annotations: map[string]string{},
	}

	td := t.TempDir()
	outFile := filepath.Join(td, "Chart.yaml")
	if err := h.SaveToFile(outFile); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	info, err := os.Stat(outFile)
	if err != nil {
		t.Fatalf("saved file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("saved file is empty")
	}

	// Re-load and verify
	h2 := NewHelmChart()
	if err := h2.LoadFromFile(outFile); err != nil {
		t.Fatalf("re-loading saved file failed: %v", err)
	}
	if h2.Metadata.Name != "test-chart" {
		t.Fatalf("expected name 'test-chart', got %q", h2.Metadata.Name)
	}
}

func TestSaveToFile_InvalidMetadata(t *testing.T) {
	h := NewHelmChart()
	// Empty metadata should fail validation
	td := t.TempDir()
	outFile := filepath.Join(td, "Chart.yaml")
	err := h.SaveToFile(outFile)
	if err == nil {
		t.Fatal("expected validation error for empty metadata")
	}
}

func TestSaveToFile_InvalidPath(t *testing.T) {
	h := NewHelmChart()
	h.Metadata = ChartMetadata{
		APIVersion:  "v2",
		Name:        "test",
		Version:     "1.0.0",
		Description: "test",
		AppVersion:  "1.0",
		Icon:        "https://example.com/icon.png",
		Home:        "https://example.com",
		KubeVersion: ">=1.27.0-0",
		Maintainers: []Maintainer{{Name: "Test", URL: "https://test.com"}},
	}
	err := h.SaveToFile("/nonexistent/path/Chart.yaml")
	if err == nil {
		t.Fatal("expected error for non-existent directory")
	}
}

func TestSetAnnotation_NewAnnotation(t *testing.T) {
	h := NewHelmChart()
	h.Metadata.Annotations = map[string]string{}

	h.setAnnotation("custom/key", "custom-value", false)
	if h.Metadata.Annotations["custom/key"] != "custom-value" {
		t.Fatalf("expected annotation 'custom-value', got %q", h.Metadata.Annotations["custom/key"])
	}
}

func TestSetAnnotation_NoForceOverwrite(t *testing.T) {
	h := NewHelmChart()
	h.Metadata.Annotations = map[string]string{
		"existing/key": "original",
	}

	h.setAnnotation("existing/key", "new-value", false)
	if h.Metadata.Annotations["existing/key"] != "original" {
		t.Fatalf("expected annotation to remain 'original', got %q", h.Metadata.Annotations["existing/key"])
	}
}

func TestSetAnnotation_ForceOverwrite(t *testing.T) {
	h := NewHelmChart()
	h.Metadata.Annotations = map[string]string{
		"existing/key": "original",
	}

	h.setAnnotation("existing/key", "new-value", true)
	if h.Metadata.Annotations["existing/key"] != "new-value" {
		t.Fatalf("expected annotation to be overwritten to 'new-value', got %q", h.Metadata.Annotations["existing/key"])
	}
}

func TestSetDeprecation_DefaultFalse(t *testing.T) {
	h := NewHelmChart()
	h.setDeprecation()
	if h.Metadata.Deprecated {
		t.Fatal("expected deprecated to be false")
	}
}

func TestSetIcon_Empty(t *testing.T) {
	h := NewHelmChart()
	h.setIcon("https://example.com/icon.png")
	if h.Metadata.Icon != "https://example.com/icon.png" {
		t.Fatalf("expected icon set, got %q", h.Metadata.Icon)
	}
}

func TestSetIcon_AlreadySet(t *testing.T) {
	h := NewHelmChart()
	h.Metadata.Icon = "https://existing.com/icon.png"
	h.setIcon("https://new.com/icon.png")
	if h.Metadata.Icon != "https://existing.com/icon.png" {
		t.Fatal("expected existing icon to be preserved")
	}
}

func TestSetHome_Empty(t *testing.T) {
	h := NewHelmChart()
	h.setHome("https://example.com")
	if h.Metadata.Home != "https://example.com" {
		t.Fatalf("expected home set, got %q", h.Metadata.Home)
	}
}

func TestSetHome_AlreadySet(t *testing.T) {
	h := NewHelmChart()
	h.Metadata.Home = "https://existing.com"
	h.setHome("https://new.com")
	if h.Metadata.Home != "https://existing.com" {
		t.Fatal("expected existing home to be preserved")
	}
}

func TestSetDescription_Empty(t *testing.T) {
	h := NewHelmChart()
	h.setDescription("My chart description")
	if h.Metadata.Description != "My chart description" {
		t.Fatalf("expected description set, got %q", h.Metadata.Description)
	}
}

func TestSetDescription_AlreadySet(t *testing.T) {
	h := NewHelmChart()
	h.Metadata.Description = "Existing description"
	h.setDescription("New description")
	if h.Metadata.Description != "Existing description" {
		t.Fatal("expected existing description to be preserved")
	}
}

func TestSetAppVersion_Empty(t *testing.T) {
	h := NewHelmChart()
	h.setAppVersion("2.0.0")
	if h.Metadata.AppVersion != "2.0.0" {
		t.Fatalf("expected appVersion set, got %q", h.Metadata.AppVersion)
	}
}

func TestSetAppVersion_AlreadySet(t *testing.T) {
	h := NewHelmChart()
	h.Metadata.AppVersion = "1.0.0"
	h.setAppVersion("2.0.0")
	if h.Metadata.AppVersion != "1.0.0" {
		t.Fatal("expected existing appVersion to be preserved")
	}
}

func TestSetType_Empty(t *testing.T) {
	h := NewHelmChart()
	h.setType("application")
	if h.Metadata.Type != "application" {
		t.Fatalf("expected type set, got %q", h.Metadata.Type)
	}
}

func TestSetType_AlreadySet(t *testing.T) {
	h := NewHelmChart()
	h.Metadata.Type = "library"
	h.setType("application")
	if h.Metadata.Type != "library" {
		t.Fatal("expected existing type to be preserved")
	}
}

func TestSetApiVersion_AlwaysOverwrites(t *testing.T) {
	h := NewHelmChart()
	h.Metadata.APIVersion = "v1"
	h.setApiVersion("v2")
	// setApiVersion always overwrites
	if h.Metadata.APIVersion != "v2" {
		t.Fatalf("expected apiVersion to be 'v2', got %q", h.Metadata.APIVersion)
	}
}

func TestSetKubeVersion_AlwaysOverwrites(t *testing.T) {
	h := NewHelmChart()
	h.Metadata.KubeVersion = ">=1.20.0-0"
	h.setKubeVersion(">=1.27.0-0")
	// setKubeVersion always overwrites
	if h.Metadata.KubeVersion != ">=1.27.0-0" {
		t.Fatalf("expected kubeVersion to be '>=1.27.0-0', got %q", h.Metadata.KubeVersion)
	}
}

func TestSetMaintainers_AlwaysReplaces(t *testing.T) {
	h := NewHelmChart()
	h.Metadata.Maintainers = []Maintainer{{Name: "OldMaintainer", URL: "https://old.com"}}

	h.setMaintainers(Maintainer{
		Name:  "NewMaintainer",
		Email: "new@example.com",
		URL:   "https://new.com",
	})
	// setMaintainers always replaces
	if len(h.Metadata.Maintainers) != 1 {
		t.Fatalf("expected 1 maintainer, got %d", len(h.Metadata.Maintainers))
	}
	if h.Metadata.Maintainers[0].Name != "NewMaintainer" {
		t.Fatalf("expected maintainer 'NewMaintainer', got %q", h.Metadata.Maintainers[0].Name)
	}
}

func TestSetDefaultValues_SetsAllFields(t *testing.T) {
	h := NewHelmChart()
	h.Metadata.Name = "test-chart"
	h.Metadata.Version = "1.0.0"

	h.setDefaultValues()

	if h.Metadata.APIVersion != apiVersion {
		t.Fatalf("expected apiVersion %q, got %q", apiVersion, h.Metadata.APIVersion)
	}
	if h.Metadata.KubeVersion != kubeVersion {
		t.Fatalf("expected kubeVersion %q, got %q", kubeVersion, h.Metadata.KubeVersion)
	}
	if h.Metadata.Type != chartType {
		t.Fatalf("expected type %q, got %q", chartType, h.Metadata.Type)
	}
	if h.Metadata.AppVersion != defaultAppVersion {
		t.Fatalf("expected appVersion %q, got %q", defaultAppVersion, h.Metadata.AppVersion)
	}
	if h.Metadata.Description != defaultDescription {
		t.Fatalf("expected description %q, got %q", defaultDescription, h.Metadata.Description)
	}
	if h.Metadata.Annotations == nil {
		t.Fatal("expected annotations to be initialized")
	}
	if h.Metadata.Annotations["trueforge.org/category"] != defaultCategory {
		t.Fatalf("expected category annotation %q", defaultCategory)
	}
}

func TestLoadAndSaveRoundTrip(t *testing.T) {
	testDataPath := filepath.Join("..", "..", "..", "testdata", "chart_yaml")
	srcFile := filepath.Join(testDataPath, "validChart.yaml")

	h := NewHelmChart()
	if err := h.LoadFromFile(srcFile); err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	td := t.TempDir()
	outFile := filepath.Join(td, "Chart.yaml")
	if err := h.SaveToFile(outFile); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	h2 := NewHelmChart()
	if err := h2.LoadFromFile(outFile); err != nil {
		t.Fatalf("re-LoadFromFile failed: %v", err)
	}

	if h.Metadata.Name != h2.Metadata.Name {
		t.Fatalf("name mismatch: %q vs %q", h.Metadata.Name, h2.Metadata.Name)
	}
	if h.Metadata.Version != h2.Metadata.Version {
		t.Fatalf("version mismatch: %q vs %q", h.Metadata.Version, h2.Metadata.Version)
	}
}

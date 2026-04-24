package chartFile

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/charts/image"
)

func TestSetAppVersionFromImage(t *testing.T) {
	type TestData struct {
		chart  *HelmChart
		image  *image.Images
		key    string
		result string
	}

	tests := []TestData{
		{
			chart: &HelmChart{
				Metadata: ChartMetadata{
					AppVersion: "1.0.0",
				},
			},
			image: &image.Images{
				ImagesMap: map[string]image.ImageDetails{
					"image": {
						Repository: "nginx",
						Tag:        "1.15.8",
						Link:       "https://hub.docker.com/_/nginx",
						Version:    "1.15.8",
					},
				},
			},
			key:    "image",
			result: "1.15.8",
		},
		{
			chart: &HelmChart{
				Metadata: ChartMetadata{
					AppVersion: "1.0.0",
				},
			},
			image: &image.Images{
				ImagesMap: map[string]image.ImageDetails{
					"image": {
						Repository: "nginx",
						Tag:        "1.15.8",
						Link:       "https://hub.docker.com/_/nginx",
						Version:    "1.15.8",
					},
				},
			},
			key:    "nonexistent",
			result: "1.0.0",
		},
	}

	for _, tt := range tests {
		setAppVersionFromImage(tt.chart, tt.image, tt.key)
		if tt.chart.Metadata.AppVersion != tt.result {
			t.Errorf("Expected %s, got %s", tt.result, tt.chart.Metadata.AppVersion)
		}
	}
}

func TestResolveChartPath_ErrorsAndDirectory(t *testing.T) {
	if _, err := resolveChartPath(filepath.Join(t.TempDir(), "does-not-exist")); err == nil {
		t.Fatalf("expected stat error for missing path")
	}

	d := t.TempDir()
	resolved, err := resolveChartPath(d)
	if err != nil {
		t.Fatalf("unexpected resolve error: %v", err)
	}
	if resolved != filepath.Join(d, "Chart.yaml") {
		t.Fatalf("unexpected resolved chart path: %s", resolved)
	}
}

func TestLoadChartImages_Error(t *testing.T) {
	if _, _, err := loadChartImages(filepath.Join(t.TempDir(), "Chart.yaml")); err == nil {
		t.Fatalf("expected error when values.yaml is missing")
	}
}

func TestBumpChartVersion_InvalidAndInvalidSemver(t *testing.T) {
	chart := NewHelmChart()
	chart.Metadata.Name = "test"
	chart.Metadata.Version = "1.2.3"
	bumpChartVersion(chart, "noop")
	if chart.Metadata.Version != "1.2.3" {
		t.Fatalf("expected version unchanged for invalid bump kind")
	}

	chart.Metadata.Version = "not-semver"
	bumpChartVersion(chart, "patch")
	if chart.Metadata.Version != "" {
		t.Fatalf("expected version to be empty when semver increment fails")
	}
}

func TestGenerateChartArtifacts_ReadmeAndHelmignoreErrors(t *testing.T) {
	td := t.TempDir()
	chartPath := filepath.Join(td, "stable", "app", "Chart.yaml")
	if err := os.MkdirAll(filepath.Dir(chartPath), 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(chartPath, []byte("name: app\nversion: 1.0.0\n"), 0644); err != nil {
		t.Fatalf("write chart failed: %v", err)
	}

	if err := generateChartArtifacts(chartPath, "app", "stable"); err == nil {
		t.Fatalf("expected readme generation error when template root is missing")
	}

	tplRoot := filepath.Join(td, "templates-root")
	if err := os.MkdirAll(filepath.Join(tplRoot, "templates"), 0755); err != nil {
		t.Fatalf("mkdir templates failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tplRoot, "templates", "README.md.tpl"), []byte("# CHARTPLACEHOLDER\n"), 0644); err != nil {
		t.Fatalf("write readme template failed: %v", err)
	}

	deepChartPath := filepath.Join(tplRoot, "charts", "stable", "app", "Chart.yaml")
	if err := os.MkdirAll(filepath.Dir(deepChartPath), 0755); err != nil {
		t.Fatalf("mkdir deep chart failed: %v", err)
	}
	if err := os.WriteFile(deepChartPath, []byte("name: app\nversion: 1.0.0\n"), 0644); err != nil {
		t.Fatalf("write deep chart failed: %v", err)
	}

	if err := generateChartArtifacts(deepChartPath, "app", "stable"); err == nil {
		t.Fatalf("expected helmignore generation error when helmignore template is missing")
	}
}

func TestUpdateChartFile_ErrorBranches(t *testing.T) {
	if err := UpdateChartFile(filepath.Join(t.TempDir(), "missing"), "patch"); err == nil {
		t.Fatalf("expected resolveChartPath error")
	}

	badChartDir := t.TempDir()
	badChartPath := filepath.Join(badChartDir, "Chart.yaml")
	if err := os.WriteFile(badChartPath, []byte("name: [broken\n"), 0644); err != nil {
		t.Fatalf("write bad chart failed: %v", err)
	}
	if err := UpdateChartFile(badChartPath, "patch"); err == nil {
		t.Fatalf("expected chart load error")
	}

	chartDir := filepath.Join(t.TempDir(), "stable", "app")
	chartPath := filepath.Join(chartDir, "Chart.yaml")
	if err := os.MkdirAll(chartDir, 0755); err != nil {
		t.Fatalf("mkdir chart dir failed: %v", err)
	}
	if err := os.WriteFile(chartPath, []byte("apiVersion: v2\nname: app\nversion: 1.0.0\ndescription: d\nappVersion: 1.0\nicon: https://x\nhome: https://x\nkubeVersion: '>=1.27.0-0'\nmaintainers:\n  - name: t\n    url: https://x\n"), 0644); err != nil {
		t.Fatalf("write chart failed: %v", err)
	}
	if err := UpdateChartFile(chartPath, "patch"); err == nil {
		t.Fatalf("expected loadChartImages error when values.yaml is missing")
	}

	if err := os.WriteFile(filepath.Join(chartDir, "values.yaml"), []byte("image:\n  repository: nginx\n  tag: 1.2.3\n"), 0644); err != nil {
		t.Fatalf("write values failed: %v", err)
	}
	orig := updateSourcesFunc
	updateSourcesFunc = func(_ *HelmChart, _ string, _ []string) error { return os.ErrInvalid }
	t.Cleanup(func() { updateSourcesFunc = orig })
	if err := UpdateChartFile(chartPath, "patch"); err == nil {
		t.Fatalf("expected updateSources error")
	}
}

func TestUpdateChartFile_SaveAndArtifactErrors(t *testing.T) {
	orig := updateSourcesFunc
	updateSourcesFunc = updateSources
	t.Cleanup(func() { updateSourcesFunc = orig })
	origSave := saveChartFunc
	t.Cleanup(func() { saveChartFunc = origSave })

	chartDir := filepath.Join(t.TempDir(), "stable", "broken-save")
	chartPath := filepath.Join(chartDir, "Chart.yaml")
	if err := os.MkdirAll(chartDir, 0755); err != nil {
		t.Fatalf("mkdir chart dir failed: %v", err)
	}
	if err := os.WriteFile(chartPath, []byte("apiVersion: v2\nname: app\nversion: 1.0.0\ndescription: d\nappVersion: 1.0\nicon: https://x\nhome: https://x\nkubeVersion: '>=1.27.0-0'\nmaintainers:\n  - name: t\n    url: https://x\n"), 0644); err != nil {
		t.Fatalf("write chart failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(chartDir, "values.yaml"), []byte("image:\n  repository: nginx\n  tag: 1.2.3\n"), 0644); err != nil {
		t.Fatalf("write values failed: %v", err)
	}
	saveChartFunc = func(_ *HelmChart, _ string) error { return os.ErrInvalid }
	if err := UpdateChartFile(chartPath, "patch"); err == nil {
		t.Fatalf("expected save error")
	}
	saveChartFunc = origSave

	chartDir2 := filepath.Join(t.TempDir(), "stable", "artifact-fail")
	chartPath2 := filepath.Join(chartDir2, "Chart.yaml")
	if err := os.MkdirAll(chartDir2, 0755); err != nil {
		t.Fatalf("mkdir chart dir 2 failed: %v", err)
	}
	if err := os.WriteFile(chartPath2, []byte("apiVersion: v2\nname: app\nversion: 1.0.0\ndescription: d\nappVersion: 1.0\nicon: https://x\nhome: https://x\nkubeVersion: '>=1.27.0-0'\nmaintainers:\n  - name: t\n    url: https://x\n"), 0644); err != nil {
		t.Fatalf("write chart 2 failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(chartDir2, "values.yaml"), []byte("image:\n  repository: nginx\n  tag: 1.2.3\n"), 0644); err != nil {
		t.Fatalf("write values 2 failed: %v", err)
	}
	if err := UpdateChartFile(chartPath2, "patch"); err == nil {
		t.Fatalf("expected artifact generation error because templates are missing")
	}
}

func TestInitializeAnnotations_NilMap(t *testing.T) {
	chart := &HelmChart{}
	initializeAnnotations(chart)
	if chart.Metadata.Annotations == nil {
		t.Fatalf("expected annotations map to be initialized")
	}
}
func TestGetTrain(t *testing.T) {
	type TestData struct {
		name      string
		chart     *HelmChart
		chartPath string
		result    string
	}

	tests := []TestData{
		{
			name: "Test get train from path",
			chart: &HelmChart{
				Metadata: ChartMetadata{
					Name: "test-chart",
					Annotations: map[string]string{
						"trueforge.org/train": "express",
					},
				},
			},
			chartPath: "../../testdata/updater/stable/my-app/Chart.yaml",
			result:    "stable",
		},
		{
			name: "Test get train from annotations as fallback",
			chart: &HelmChart{
				Metadata: ChartMetadata{
					Name: "test-chart",
					Annotations: map[string]string{
						"trueforge.org/train": "dev",
					},
				},
			},
			// Too short path, cant detect train from path
			// so we should fallback to annotations
			chartPath: "my-app/Chart.yaml",
			result:    "dev",
		},
		{
			name: "Test failing to get train from path or annotations",
			chart: &HelmChart{
				Metadata: ChartMetadata{
					Name:        "test-chart",
					Annotations: map[string]string{},
				},
			},
			// Too short path, cant detect train from path
			// so we should fallback to annotations
			chartPath: "my-app/Chart.yaml",
			result:    "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			train := GetTrain(tt.chartPath, tt.chart)

			if train != tt.result {
				t.Errorf("Expected train to be %s, but got %s", tt.result, train)
			}
		})
	}
}

func TestSetMetadata(t *testing.T) {
	type TestData struct {
		chart    *HelmChart
		train    string
		expected *HelmChart
	}

	tests := []TestData{
		{
			chart: &HelmChart{
				Metadata: ChartMetadata{
					Name:        "test-chart",
					Annotations: map[string]string{},
				},
			},
			train: "stable",
			expected: &HelmChart{
				Metadata: ChartMetadata{
					Name: "test-chart",
					Annotations: map[string]string{
						"truecharts.org/train": "stable",
					},
					Icon: "https://truecharts.org/img/hotlink-ok/chart-icons/test-chart.webp",
					Home: "https://truecharts.org/charts/stable/test-chart",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.chart.Metadata.Name, func(t *testing.T) {
			setMetadata(tt.chart, tt.train)

			if !reflect.DeepEqual(tt.chart, tt.expected) {
				t.Errorf("Expected chart to be %v, but got %v", tt.expected, tt.chart)
			}
		})
	}
}

func TestUpdateSources(t *testing.T) {
	type TestData struct {
		name       string
		chart      *HelmChart
		train      string
		imageLinks []string
		expected   []string
	}

	tests := []TestData{
		{
			name: "Test update sources",
			chart: &HelmChart{
				Metadata: ChartMetadata{
					Name: "test-chart",
					Sources: []string{
						"",
						"https://ghcr/truecharts/some-chart",
						"https://docker.io/truecharts/some-chart",
						"https://hub.docker/truecharts/some-chart",
						"https://fleet.linuxserver/truecharts/some-chart",
						"https://mcr.microsoft/truecharts/some-chart",
						"https://github.com/truecharts/some-chart",
						"https://gallery.ecr.aws/truecharts/some-chart",
						"https://gcr/truecharts/some-chart",
						"https://quay/truecharts/some-chart",
						"http://truecharts/some-chart",
						"https://truecharts.azurecr.io/some-chart",
						"https://truecharts.ocir.io/some-chart",
						"https://unrelated.com/some-chart",
						"https://unrelated.com/some-chart",
						"https://cr.hotio.dev/truecharts/some-chart",
					},
				},
			},
			train: "stable",
			imageLinks: []string{
				"",
				"https://hub.docker.com/_/nginx",
				"https://quay.io/truecharts/test-chart",
			},
			expected: []string{
				"https://github.com/trueforge-org/truecharts/tree/master/charts/stable/test-chart",
				"https://hub.docker.com/_/nginx",
				"https://quay.io/truecharts/test-chart",
				"https://unrelated.com/some-chart",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.chart.Metadata.Name, func(t *testing.T) {
			if err := updateSources(tt.chart, tt.train, tt.imageLinks); err != nil {
				t.Errorf("Expected no error, but got %v", err)
			}

			if !reflect.DeepEqual(tt.chart.Metadata.Sources, tt.expected) {
				t.Errorf("Expected chart to be %v, but got %v", tt.expected, tt.chart.Metadata.Sources)
			}
		})
	}
}

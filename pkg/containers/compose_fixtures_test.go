package containers

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

var updateGolden = flag.Bool("update", false, "update golden expected.yaml fixtures under testdata/compose")

// caseFile is the per-test-case configuration loaded from case.yaml.
//
// Layout of one fixture under testdata/compose/<case>/:
//
//	case.yaml         — app name, version and (optional) extra dependency-app
//	                    versions for resolved dependencies.
//	settings.yaml     — the app's settings.yaml, parsed by ParseSettings.
//	deps/<name>.yaml  — settings.yaml for each resolvable dependency app.
//	                    Optional; missing dependencies are reported as
//	                    not-found by the resolver.
//	expected.yaml     — golden output of BuildComposeYAML. Re-generate by
//	                    running `go test ./pkg/containers -run TestBuildComposeYAML_Fixtures -update`.
type caseFile struct {
	App         string            `yaml:"app"`
	Version     string            `yaml:"version"`
	DepVersions map[string]string `yaml:"dep_versions,omitempty"`
}

func TestBuildComposeYAML_Fixtures(t *testing.T) {
	root := "testdata/compose"
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read fixtures dir: %v", err)
	}

	var caseNames []string
	for _, e := range entries {
		if e.IsDir() {
			caseNames = append(caseNames, e.Name())
		}
	}
	sort.Strings(caseNames)

	if len(caseNames) == 0 {
		t.Fatalf("no fixture directories under %s", root)
	}

	for _, name := range caseNames {
		name := name
		t.Run(name, func(t *testing.T) {
			caseDir := filepath.Join(root, name)

			caseBytes, err := os.ReadFile(filepath.Join(caseDir, "case.yaml"))
			if err != nil {
				t.Fatalf("read case.yaml: %v", err)
			}
			var c caseFile
			if err := yaml.Unmarshal(caseBytes, &c); err != nil {
				t.Fatalf("parse case.yaml: %v", err)
			}
			if c.App == "" {
				t.Fatalf("case.yaml: app is required")
			}

			settings, ok, err := ParseSettings(filepath.Join(caseDir, "settings.yaml"))
			if err != nil {
				t.Fatalf("parse settings.yaml: %v", err)
			}
			if !ok {
				t.Fatalf("settings.yaml not found in %s", caseDir)
			}

			resolve := newDirResolver(t, filepath.Join(caseDir, "deps"), c.DepVersions)

			// Make the random-secret substitution deterministic so
			// fixtures that exercise MY<NAME>PASS placeholders can
			// pin an exact expected.yaml. Each unique placeholder
			// gets a stable, sequence-numbered fake secret.
			prevSecret := randomSecretFn
			defer func() { randomSecretFn = prevSecret }()
			var secretSeq int
			randomSecretFn = func() (string, error) {
				secretSeq++
				return fmt.Sprintf("test-secret-%d", secretSeq), nil
			}

			got, err := BuildComposeYAML(c.App, c.Version, settings, resolve)
			if err != nil {
				t.Fatalf("BuildComposeYAML: %v", err)
			}

			expectedPath := filepath.Join(caseDir, "expected.yaml")
			if *updateGolden {
				if err := os.WriteFile(expectedPath, []byte(got), 0o644); err != nil {
					t.Fatalf("update golden: %v", err)
				}
				t.Logf("updated %s", expectedPath)
				return
			}

			wantBytes, err := os.ReadFile(expectedPath)
			if err != nil {
				t.Fatalf("read expected.yaml (run with -update to create): %v", err)
			}
			want := string(wantBytes)
			if got != want {
				t.Errorf("rendered compose differs from %s\n--- got ---\n%s\n--- want ---\n%s",
					expectedPath, got, want)
			}
		})
	}
}

// newDirResolver returns a DependencyResolver that loads each dependency's
// settings.yaml from depsDir/<name>/settings.yaml and uses the version from
// versions[name] (defaulting to empty, which renders the "rolling" tag).
func newDirResolver(t *testing.T, depsDir string, versions map[string]string) DependencyResolver {
	t.Helper()
	return func(name string) (AppSettings, string, bool, error) {
		path := filepath.Join(depsDir, name, "settings.yaml")
		settings, ok, err := ParseSettings(path)
		if err != nil {
			return AppSettings{}, "", false, err
		}
		if !ok {
			return AppSettings{}, "", false, nil
		}
		return settings, versions[name], true, nil
	}
}

// TestBuildComposeYAML_FixturesNotEmpty is a sanity check that ensures every
// fixture's expected.yaml ends with a newline and renders the app's own
// service. It catches accidentally truncated golden files.
func TestBuildComposeYAML_FixturesNotEmpty(t *testing.T) {
	root := "testdata/compose"
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read fixtures dir: %v", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		expected := filepath.Join(root, e.Name(), "expected.yaml")
		data, err := os.ReadFile(expected)
		if err != nil {
			t.Errorf("%s: %v", expected, err)
			continue
		}
		if len(data) == 0 || !strings.HasSuffix(string(data), "\n") {
			t.Errorf("%s: expected file must be non-empty and end with a newline", expected)
		}
	}
}

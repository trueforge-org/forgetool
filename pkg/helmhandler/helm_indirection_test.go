package helmhandler

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

// resetHelmFns saves and restores all overridable function variables in helm.go.
func resetHelmFns(t *testing.T) {
	t.Helper()
	oInit := helmInitHelmActionConfigFn
	oEnsure := helmEnsureNamespaceFn
	oResolve := helmResolveChartPathFn
	oLoad := helmLoaderLoadFn
	oBuild := helmBuildReleaseValueFilesFn
	oMerge := helmMergeValueFilesFn
	oRetry := helmRunInstallWithTimeoutRetryFn
	oWait := helmWaitForReleaseFn
	oNewInst := helmNewInstallFn
	oNewUpg := helmNewUpgradeFn
	oUpgRun := helmUpgradeRunFn
	oInstRun := helmInstallRunFn
	oStatusRun := helmNewStatusRunFn
	oSleep := helmSleepFn
	oKs := helmGetKubernetesClientSetFn
	oNsGet := helmNamespaceGetFn
	oNsCreate := helmNamespaceCreateFn
	oActionInit := helmActionConfigInitFn
	oMergeVals := helmValuesMergeFn
	oPull := helmHelmPullFn
	oCreateV := helmCreateValuesYAMLFn
	oStat := helmOSStatFn
	oMarshal := helmYAMLMarshalFn
	oWrite := helmWriteFileFn

	t.Cleanup(func() {
		helmInitHelmActionConfigFn = oInit
		helmEnsureNamespaceFn = oEnsure
		helmResolveChartPathFn = oResolve
		helmLoaderLoadFn = oLoad
		helmBuildReleaseValueFilesFn = oBuild
		helmMergeValueFilesFn = oMerge
		helmRunInstallWithTimeoutRetryFn = oRetry
		helmWaitForReleaseFn = oWait
		helmNewInstallFn = oNewInst
		helmNewUpgradeFn = oNewUpg
		helmUpgradeRunFn = oUpgRun
		helmInstallRunFn = oInstRun
		helmNewStatusRunFn = oStatusRun
		helmSleepFn = oSleep
		helmGetKubernetesClientSetFn = oKs
		helmNamespaceGetFn = oNsGet
		helmNamespaceCreateFn = oNsCreate
		helmActionConfigInitFn = oActionInit
		helmValuesMergeFn = oMergeVals
		helmHelmPullFn = oPull
		helmCreateValuesYAMLFn = oCreateV
		helmOSStatFn = oStat
		helmYAMLMarshalFn = oMarshal
		helmWriteFileFn = oWrite
	})
}

func newStubChart() *chart.Chart {
	return &chart.Chart{
		Metadata: &chart.Metadata{Name: "stub", Version: "0.1.0"},
		Values:   map[string]interface{}{"key": "value"},
	}
}

func successInit(_ string, _ bool) (*cli.EnvSettings, *action.Configuration, error) {
	return cli.New(), &action.Configuration{}, nil
}

func TestHelmInstall_AllStubsHappyPath(t *testing.T) {
	resetHelmFns(t)

	helmInitHelmActionConfigFn = successInit
	helmEnsureNamespaceFn = func(*action.Configuration, string) error { return nil }
	helmResolveChartPathFn = func(string, string, string) (string, error) { return "/tmp/stub", nil }
	helmLoaderLoadFn = func(string) (*chart.Chart, error) { return newStubChart(), nil }
	helmBuildReleaseValueFilesFn = func(string, string, map[string]interface{}, bool) ([]string, error) { return nil, nil }
	helmMergeValueFilesFn = func(*cli.EnvSettings, []string) (map[string]interface{}, error) {
		return map[string]interface{}{}, nil
	}
	helmRunInstallWithTimeoutRetryFn = func(*action.Install, *chart.Chart, map[string]interface{}) (*release.Release, error) {
		return &release.Release{Name: "rel", Namespace: "ns"}, nil
	}
	helmWaitForReleaseFn = func(*action.Configuration, string, string) {}
	helmNewInstallFn = func(*action.Configuration) *action.Install { return &action.Install{} }

	if err := HelmInstall("repo", "chart", "rel", "ns", "vals", "1.0.0", false, true, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHelmInstall_DryRun(t *testing.T) {
	if err := HelmInstall("", "", "", "", "", "", true, false, true); err != nil {
		t.Fatalf("dry run should return nil, got: %v", err)
	}
}

func TestHelmInstall_ErrorPaths(t *testing.T) {
	cases := []struct {
		name string
		set  func()
		want string
	}{
		{
			name: "init error",
			set: func() {
				helmInitHelmActionConfigFn = func(string, bool) (*cli.EnvSettings, *action.Configuration, error) {
					return nil, nil, errors.New("init boom")
				}
			},
			want: "init boom",
		},
		{
			name: "ensure namespace error",
			set: func() {
				helmInitHelmActionConfigFn = successInit
				helmEnsureNamespaceFn = func(*action.Configuration, string) error { return errors.New("ns fail") }
			},
			want: "failed to ensure namespace",
		},
		{
			name: "resolve error",
			set: func() {
				helmInitHelmActionConfigFn = successInit
				helmEnsureNamespaceFn = func(*action.Configuration, string) error { return nil }
				helmResolveChartPathFn = func(string, string, string) (string, error) { return "", errors.New("resolve") }
			},
			want: "resolve",
		},
		{
			name: "loader error",
			set: func() {
				helmInitHelmActionConfigFn = successInit
				helmEnsureNamespaceFn = func(*action.Configuration, string) error { return nil }
				helmResolveChartPathFn = func(string, string, string) (string, error) { return "p", nil }
				helmLoaderLoadFn = func(string) (*chart.Chart, error) { return nil, errors.New("load") }
			},
			want: "failed to load chart",
		},
		{
			name: "build values error",
			set: func() {
				helmInitHelmActionConfigFn = successInit
				helmEnsureNamespaceFn = func(*action.Configuration, string) error { return nil }
				helmResolveChartPathFn = func(string, string, string) (string, error) { return "p", nil }
				helmLoaderLoadFn = func(string) (*chart.Chart, error) { return newStubChart(), nil }
				helmNewInstallFn = func(*action.Configuration) *action.Install { return &action.Install{} }
				helmBuildReleaseValueFilesFn = func(string, string, map[string]interface{}, bool) ([]string, error) {
					return nil, errors.New("buildvals")
				}
			},
			want: "buildvals",
		},
		{
			name: "merge values error",
			set: func() {
				helmInitHelmActionConfigFn = successInit
				helmEnsureNamespaceFn = func(*action.Configuration, string) error { return nil }
				helmResolveChartPathFn = func(string, string, string) (string, error) { return "p", nil }
				helmLoaderLoadFn = func(string) (*chart.Chart, error) { return newStubChart(), nil }
				helmNewInstallFn = func(*action.Configuration) *action.Install { return &action.Install{} }
				helmBuildReleaseValueFilesFn = func(string, string, map[string]interface{}, bool) ([]string, error) { return nil, nil }
				helmMergeValueFilesFn = func(*cli.EnvSettings, []string) (map[string]interface{}, error) {
					return nil, errors.New("merge")
				}
			},
			want: "failed to merge values",
		},
		{
			name: "install retry error",
			set: func() {
				helmInitHelmActionConfigFn = successInit
				helmEnsureNamespaceFn = func(*action.Configuration, string) error { return nil }
				helmResolveChartPathFn = func(string, string, string) (string, error) { return "p", nil }
				helmLoaderLoadFn = func(string) (*chart.Chart, error) { return newStubChart(), nil }
				helmNewInstallFn = func(*action.Configuration) *action.Install { return &action.Install{} }
				helmBuildReleaseValueFilesFn = func(string, string, map[string]interface{}, bool) ([]string, error) { return nil, nil }
				helmMergeValueFilesFn = func(*cli.EnvSettings, []string) (map[string]interface{}, error) { return nil, nil }
				helmRunInstallWithTimeoutRetryFn = func(*action.Install, *chart.Chart, map[string]interface{}) (*release.Release, error) {
					return nil, errors.New("retry")
				}
			},
			want: "retry",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resetHelmFns(t)
			c.set()
			err := HelmInstall("repo", "chart", "rel", "ns", "vals", "1.0.0", false, false, true)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("expected error containing %q, got %v", c.want, err)
			}
		})
	}
}

func TestHelmUpgrade(t *testing.T) {
	t.Run("happy path with wait", func(t *testing.T) {
		resetHelmFns(t)
		helmInitHelmActionConfigFn = successInit
		helmEnsureNamespaceFn = func(*action.Configuration, string) error { return nil }
		helmResolveChartPathFn = func(string, string, string) (string, error) { return "p", nil }
		helmLoaderLoadFn = func(string) (*chart.Chart, error) { return newStubChart(), nil }
		helmNewUpgradeFn = func(*action.Configuration) *action.Upgrade { return &action.Upgrade{} }
		helmBuildReleaseValueFilesFn = func(string, string, map[string]interface{}, bool) ([]string, error) { return nil, nil }
		helmMergeValueFilesFn = func(*cli.EnvSettings, []string) (map[string]interface{}, error) { return nil, nil }
		helmUpgradeRunFn = func(*action.Upgrade, string, *chart.Chart, map[string]interface{}) (*release.Release, error) {
			return &release.Release{Name: "rel", Namespace: "ns"}, nil
		}
		helmWaitForReleaseFn = func(*action.Configuration, string, string) {}
		if err := HelmUpgrade("r", "c", "rel", "ns", "v", "1.0.0", true, true); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	cases := []struct {
		name string
		set  func()
		want string
	}{
		{"init error", func() {
			helmInitHelmActionConfigFn = func(string, bool) (*cli.EnvSettings, *action.Configuration, error) {
				return nil, nil, errors.New("init")
			}
		}, "init"},
		{"ensure error", func() {
			helmInitHelmActionConfigFn = successInit
			helmEnsureNamespaceFn = func(*action.Configuration, string) error { return errors.New("e") }
		}, "failed to ensure namespace"},
		{"resolve error", func() {
			helmInitHelmActionConfigFn = successInit
			helmEnsureNamespaceFn = func(*action.Configuration, string) error { return nil }
			helmResolveChartPathFn = func(string, string, string) (string, error) { return "", errors.New("rs") }
		}, "rs"},
		{"load error", func() {
			helmInitHelmActionConfigFn = successInit
			helmEnsureNamespaceFn = func(*action.Configuration, string) error { return nil }
			helmResolveChartPathFn = func(string, string, string) (string, error) { return "p", nil }
			helmLoaderLoadFn = func(string) (*chart.Chart, error) { return nil, errors.New("ld") }
		}, "failed to load chart"},
		{"build error", func() {
			helmInitHelmActionConfigFn = successInit
			helmEnsureNamespaceFn = func(*action.Configuration, string) error { return nil }
			helmResolveChartPathFn = func(string, string, string) (string, error) { return "p", nil }
			helmLoaderLoadFn = func(string) (*chart.Chart, error) { return newStubChart(), nil }
			helmNewUpgradeFn = func(*action.Configuration) *action.Upgrade { return &action.Upgrade{} }
			helmBuildReleaseValueFilesFn = func(string, string, map[string]interface{}, bool) ([]string, error) {
				return nil, errors.New("bv")
			}
		}, "bv"},
		{"merge error", func() {
			helmInitHelmActionConfigFn = successInit
			helmEnsureNamespaceFn = func(*action.Configuration, string) error { return nil }
			helmResolveChartPathFn = func(string, string, string) (string, error) { return "p", nil }
			helmLoaderLoadFn = func(string) (*chart.Chart, error) { return newStubChart(), nil }
			helmNewUpgradeFn = func(*action.Configuration) *action.Upgrade { return &action.Upgrade{} }
			helmBuildReleaseValueFilesFn = func(string, string, map[string]interface{}, bool) ([]string, error) { return nil, nil }
			helmMergeValueFilesFn = func(*cli.EnvSettings, []string) (map[string]interface{}, error) { return nil, errors.New("m") }
		}, "failed to merge values"},
		{"upgrade error", func() {
			helmInitHelmActionConfigFn = successInit
			helmEnsureNamespaceFn = func(*action.Configuration, string) error { return nil }
			helmResolveChartPathFn = func(string, string, string) (string, error) { return "p", nil }
			helmLoaderLoadFn = func(string) (*chart.Chart, error) { return newStubChart(), nil }
			helmNewUpgradeFn = func(*action.Configuration) *action.Upgrade { return &action.Upgrade{} }
			helmBuildReleaseValueFilesFn = func(string, string, map[string]interface{}, bool) ([]string, error) { return nil, nil }
			helmMergeValueFilesFn = func(*cli.EnvSettings, []string) (map[string]interface{}, error) { return nil, nil }
			helmUpgradeRunFn = func(*action.Upgrade, string, *chart.Chart, map[string]interface{}) (*release.Release, error) {
				return nil, errors.New("up")
			}
		}, "failed to upgrade chart"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resetHelmFns(t)
			c.set()
			err := HelmUpgrade("r", "c", "rel", "ns", "v", "1.0.0", false, true)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("expected error containing %q, got %v", c.want, err)
			}
		})
	}
}

func TestRunInstallWithTimeoutRetry(t *testing.T) {
	t.Run("first call success", func(t *testing.T) {
		resetHelmFns(t)
		helmInstallRunFn = func(*action.Install, *chart.Chart, map[string]interface{}) (*release.Release, error) {
			return &release.Release{Name: "ok"}, nil
		}
		rel, err := runInstallWithTimeoutRetry(&action.Install{}, newStubChart(), nil)
		if err != nil || rel.Name != "ok" {
			t.Fatalf("unexpected: rel=%v err=%v", rel, err)
		}
	})

	t.Run("non-timeout error", func(t *testing.T) {
		resetHelmFns(t)
		helmInstallRunFn = func(*action.Install, *chart.Chart, map[string]interface{}) (*release.Release, error) {
			return nil, errors.New("permission denied")
		}
		_, err := runInstallWithTimeoutRetry(&action.Install{}, newStubChart(), nil)
		if err == nil || !strings.Contains(err.Error(), "failed to install chart") {
			t.Fatalf("expected install error, got %v", err)
		}
	})

	t.Run("timeout retry success", func(t *testing.T) {
		resetHelmFns(t)
		helmSleepFn = func(time.Duration) {}
		calls := 0
		helmInstallRunFn = func(*action.Install, *chart.Chart, map[string]interface{}) (*release.Release, error) {
			calls++
			if calls == 1 {
				return nil, errors.New("context deadline exceeded: timed out")
			}
			return &release.Release{Name: "rel"}, nil
		}
		rel, err := runInstallWithTimeoutRetry(&action.Install{}, newStubChart(), nil)
		if err != nil || rel.Name != "rel" {
			t.Fatalf("expected retry success, got %v %v", rel, err)
		}
	})

	t.Run("timeout retry timeout again", func(t *testing.T) {
		resetHelmFns(t)
		helmSleepFn = func(time.Duration) {}
		helmInstallRunFn = func(*action.Install, *chart.Chart, map[string]interface{}) (*release.Release, error) {
			return nil, errors.New("timed out")
		}
		_, err := runInstallWithTimeoutRetry(&action.Install{}, newStubChart(), nil)
		if err == nil || !strings.Contains(err.Error(), "after retry, with another timeout") {
			t.Fatalf("expected double timeout error, got %v", err)
		}
	})

	t.Run("timeout retry then non-timeout error", func(t *testing.T) {
		resetHelmFns(t)
		helmSleepFn = func(time.Duration) {}
		calls := 0
		helmInstallRunFn = func(*action.Install, *chart.Chart, map[string]interface{}) (*release.Release, error) {
			calls++
			if calls == 1 {
				return nil, errors.New("timed out")
			}
			return nil, errors.New("permission denied")
		}
		_, err := runInstallWithTimeoutRetry(&action.Install{}, newStubChart(), nil)
		if err == nil || !strings.Contains(err.Error(), "failed to install chart after retry") {
			t.Fatalf("expected after retry error, got %v", err)
		}
	})
}

func TestEnsureNamespace(t *testing.T) {
	t.Run("exists already", func(t *testing.T) {
		resetHelmFns(t)
		helmGetKubernetesClientSetFn = func(*action.Configuration) (kubernetes.Interface, error) { return fake.NewSimpleClientset(), nil }
		helmNamespaceGetFn = func(kubernetes.Interface, string) error { return nil }
		if err := ensureNamespace(&action.Configuration{}, "ns"); err != nil {
			t.Fatalf("unexpected: %v", err)
		}
	})

	t.Run("created when missing", func(t *testing.T) {
		resetHelmFns(t)
		helmGetKubernetesClientSetFn = func(*action.Configuration) (kubernetes.Interface, error) { return fake.NewSimpleClientset(), nil }
		helmNamespaceGetFn = func(kubernetes.Interface, string) error { return errors.New("not found") }
		helmNamespaceCreateFn = func(kubernetes.Interface, string) error { return nil }
		if err := ensureNamespace(&action.Configuration{}, "ns"); err != nil {
			t.Fatalf("unexpected: %v", err)
		}
	})

	t.Run("namespaceExists clientset error", func(t *testing.T) {
		resetHelmFns(t)
		helmGetKubernetesClientSetFn = func(*action.Configuration) (kubernetes.Interface, error) { return nil, errors.New("ks") }
		if err := ensureNamespace(&action.Configuration{}, "ns"); err == nil || !strings.Contains(err.Error(), "failed to check if namespace exists") {
			t.Fatalf("expected check error, got %v", err)
		}
	})

	t.Run("create namespace clientset error", func(t *testing.T) {
		resetHelmFns(t)
		first := true
		helmGetKubernetesClientSetFn = func(*action.Configuration) (kubernetes.Interface, error) {
			if first {
				first = false
				return fake.NewSimpleClientset(), nil
			}
			return nil, errors.New("ks2")
		}
		helmNamespaceGetFn = func(kubernetes.Interface, string) error { return errors.New("nope") }
		if err := ensureNamespace(&action.Configuration{}, "ns"); err == nil || !strings.Contains(err.Error(), "failed to create namespace") {
			t.Fatalf("expected create error, got %v", err)
		}
	})

	t.Run("create namespace api error", func(t *testing.T) {
		resetHelmFns(t)
		helmGetKubernetesClientSetFn = func(*action.Configuration) (kubernetes.Interface, error) { return fake.NewSimpleClientset(), nil }
		helmNamespaceGetFn = func(kubernetes.Interface, string) error { return errors.New("nope") }
		helmNamespaceCreateFn = func(kubernetes.Interface, string) error { return errors.New("api") }
		if err := ensureNamespace(&action.Configuration{}, "ns"); err == nil || !strings.Contains(err.Error(), "failed to create namespace") {
			t.Fatalf("expected create error, got %v", err)
		}
	})

	t.Run("create namespace already exists swallowed", func(t *testing.T) {
		resetHelmFns(t)
		helmGetKubernetesClientSetFn = func(*action.Configuration) (kubernetes.Interface, error) { return fake.NewSimpleClientset(), nil }
		helmNamespaceGetFn = func(kubernetes.Interface, string) error { return errors.New("nope") }
		helmNamespaceCreateFn = func(kubernetes.Interface, string) error { return errors.New("namespace already exists") }
		if err := ensureNamespace(&action.Configuration{}, "ns"); err != nil {
			t.Fatalf("expected nil for already exists, got %v", err)
		}
	})
}

func TestResolveChartPath(t *testing.T) {
	t.Run("local path passthrough", func(t *testing.T) {
		resetHelmFns(t)
		got, err := resolveChartPath("/local/path", "chart", "1.0.0")
		if err != nil || got != "/local/path" {
			t.Fatalf("unexpected: got=%q err=%v", got, err)
		}
	})

	t.Run("https with success", func(t *testing.T) {
		resetHelmFns(t)
		helmHelmPullFn = func(string, string, string, string, bool) error { return nil }
		got, err := resolveChartPath("https://example.com", "mychart", "1.2.3")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(got, "mychart-1.2.3.tgz") {
			t.Fatalf("expected tgz path, got %q", got)
		}
	})

	t.Run("oci with pull error", func(t *testing.T) {
		resetHelmFns(t)
		helmHelmPullFn = func(string, string, string, string, bool) error { return errors.New("pull") }
		_, err := resolveChartPath("oci://example.com", "c", "1")
		if err == nil || !strings.Contains(err.Error(), "failed to pull chart") {
			t.Fatalf("expected pull error, got %v", err)
		}
	})
}

func TestBuildReleaseValueFiles(t *testing.T) {
	t.Run("create error", func(t *testing.T) {
		resetHelmFns(t)
		helmCreateValuesYAMLFn = func(map[string]interface{}, string) error { return errors.New("c") }
		_, err := buildReleaseValueFiles("rel", "vals", nil, false)
		if err == nil || !strings.Contains(err.Error(), "tempvalues") {
			t.Fatalf("expected create error, got %v", err)
		}
	})

	t.Run("with values file present", func(t *testing.T) {
		resetHelmFns(t)
		helmCreateValuesYAMLFn = func(map[string]interface{}, string) error { return nil }
		helmOSStatFn = func(string) (os.FileInfo, error) { return nil, nil }
		got, err := buildReleaseValueFiles("rel", "vals", nil, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 files, got %d", len(got))
		}
	})

	t.Run("values file missing", func(t *testing.T) {
		resetHelmFns(t)
		helmCreateValuesYAMLFn = func(map[string]interface{}, string) error { return nil }
		helmOSStatFn = func(string) (os.FileInfo, error) { return nil, errors.New("nope") }
		got, err := buildReleaseValueFiles("rel", "vals", nil, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 file, got %d", len(got))
		}
	})
}

func TestMergeValueFiles(t *testing.T) {
	resetHelmFns(t)
	called := false
	helmValuesMergeFn = func(*values.Options, *cli.EnvSettings) (map[string]interface{}, error) {
		called = true
		return map[string]interface{}{"a": 1}, nil
	}
	got, err := mergeValueFiles(cli.New(), []string{"f1"})
	if err != nil || !called || got["a"] != 1 {
		t.Fatalf("unexpected: %v %v %v", err, called, got)
	}
}

func TestInitHelmActionConfig(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		resetHelmFns(t)
		helmActionConfigInitFn = func(*action.Configuration, *cli.EnvSettings, string, func(string, ...interface{})) error {
			return errors.New("init")
		}
		_, _, err := initHelmActionConfig("ns", true)
		if err == nil || !strings.Contains(err.Error(), "failed to initialize") {
			t.Fatalf("expected init error, got %v", err)
		}
	})

	t.Run("success silent", func(t *testing.T) {
		resetHelmFns(t)
		helmActionConfigInitFn = func(*action.Configuration, *cli.EnvSettings, string, func(string, ...interface{})) error { return nil }
		settings, cfg, err := initHelmActionConfig("ns", true)
		if err != nil || settings == nil || cfg == nil {
			t.Fatalf("unexpected: %v %v %v", settings, cfg, err)
		}
	})

	t.Run("success non-silent", func(t *testing.T) {
		resetHelmFns(t)
		helmActionConfigInitFn = func(*action.Configuration, *cli.EnvSettings, string, func(string, ...interface{})) error { return nil }
		_, _, err := initHelmActionConfig("ns", false)
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
	})
}

func TestWaitForRelease(t *testing.T) {
	resetHelmFns(t)
	helmSleepFn = func(time.Duration) {}
	calls := 0
	helmNewStatusRunFn = func(*action.Configuration, string) (*release.Release, error) {
		calls++
		if calls < 2 {
			return &release.Release{Info: &release.Info{Status: release.StatusPendingInstall}}, nil
		}
		return &release.Release{Info: &release.Info{Status: release.StatusDeployed}}, nil
	}
	waitForRelease(&action.Configuration{}, "rel", "ns")
	if calls < 2 {
		t.Fatalf("expected at least 2 status calls, got %d", calls)
	}
}

func TestNamespaceExistsClientsetError(t *testing.T) {
	resetHelmFns(t)
	helmGetKubernetesClientSetFn = func(*action.Configuration) (kubernetes.Interface, error) { return nil, errors.New("ks") }
	ok, err := namespaceExists(&action.Configuration{}, "ns")
	if err == nil || ok {
		t.Fatalf("expected error and false, got %v %v", ok, err)
	}
}

func TestCreateNamespaceClientsetError(t *testing.T) {
	resetHelmFns(t)
	helmGetKubernetesClientSetFn = func(*action.Configuration) (kubernetes.Interface, error) { return nil, errors.New("ks") }
	if err := createNamespace(&action.Configuration{}, "ns"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestCreateValuesYAML_MarshalError(t *testing.T) {
	resetHelmFns(t)
	td := t.TempDir()
	helmYAMLMarshalFn = func(interface{}) ([]byte, error) { return nil, errors.New("marshal") }
	if err := createValuesYAML(map[string]interface{}{"k": "v"}, filepath.Join(td, "v.yaml")); err == nil {
		t.Fatalf("expected marshal error")
	}
}

func TestCreateValuesYAML_WriteError(t *testing.T) {
	resetHelmFns(t)
	td := t.TempDir()
	helmWriteFileFn = func(string, []byte, os.FileMode) error { return errors.New("write") }
	if err := createValuesYAML(map[string]interface{}{"a": "b"}, filepath.Join(td, "v.yaml")); err == nil {
		t.Fatalf("expected write error")
	}
}

// --- helm_pull.go coverage ---

func resetHelmPullFns(t *testing.T) {
	t.Helper()
	o1 := helmPullActionConfigInitFn
	o2 := helmPullNewDefaultRegistryClientFn
	o3 := helmPullNewPullWithOptsFn
	o4 := helmPullMkdirAllFn
	o5 := helmPullConfigureVerificationFn
	o6 := helmPullResolveLinkFn
	o7 := helmPullUpdateHelmRepoFn
	o8 := helmPullClientRunFn
	o9 := helmPullRemoveFn
	o10 := helmPullPathJoinFn
	o11 := helmPullRepoNewChartRepositoryFn
	o12 := helmPullDownloadIndexFileFn
	o13 := helmPullRepoNewFileFn
	o14 := helmPullRepoLoadFileFn
	o15 := helmPullRepoWriteFileFn
	o16 := helmPullRegistryNewClientFn
	o17 := helmPullStatFn
	t.Cleanup(func() {
		helmPullActionConfigInitFn = o1
		helmPullNewDefaultRegistryClientFn = o2
		helmPullNewPullWithOptsFn = o3
		helmPullMkdirAllFn = o4
		helmPullConfigureVerificationFn = o5
		helmPullResolveLinkFn = o6
		helmPullUpdateHelmRepoFn = o7
		helmPullClientRunFn = o8
		helmPullRemoveFn = o9
		helmPullPathJoinFn = o10
		helmPullRepoNewChartRepositoryFn = o11
		helmPullDownloadIndexFileFn = o12
		helmPullRepoNewFileFn = o13
		helmPullRepoLoadFileFn = o14
		helmPullRepoWriteFileFn = o15
		helmPullRegistryNewClientFn = o16
		helmPullStatFn = o17
	})
}

func TestConfigureHelmPullVerification(t *testing.T) {
	cases := []struct {
		repo string
		want bool
	}{
		{"https://charts.trueforge.org", true},
		{"https://library-charts.trueforge.org", true},
		{"https://deps.trueforge.org", true},
		{"https://charts.jetstack.io", true},
		{"https://other", false},
	}
	for _, c := range cases {
		client := &action.Pull{}
		configureHelmPullVerification(client, c.repo)
		if client.Verify != c.want {
			t.Fatalf("repo %s: expected Verify=%v, got %v", c.repo, c.want, client.Verify)
		}
	}
}

func TestResolveHelmPullLink(t *testing.T) {
	link, repo := resolveHelmPullLink("https://example.com", "mychart")
	if link != "mychart" || repo != "https://example.com" {
		t.Fatalf("https case: %q %q", link, repo)
	}
	link, repo = resolveHelmPullLink("oci://example.com/repo", "mychart")
	if link != "oci://example.com/repo/mychart" || repo != "" {
		t.Fatalf("oci case: %q %q", link, repo)
	}
}

func TestHelmPull_HappyPath(t *testing.T) {
	resetHelmPullFns(t)
	helmPullActionConfigInitFn = func(*action.Configuration, *cli.EnvSettings, func(string, ...interface{})) error { return nil }
	helmPullNewDefaultRegistryClientFn = func(bool, *cli.EnvSettings) (*registry.Client, error) { return nil, nil }
	helmPullMkdirAllFn = func(string, os.FileMode) error { return nil }
	helmPullConfigureVerificationFn = func(*action.Pull, string) {}
	helmPullResolveLinkFn = func(string, string) (string, string) { return "link", "repo" }
	helmPullUpdateHelmRepoFn = func(string, string, bool) error { return nil }
	helmPullClientRunFn = func(*action.Pull, string) (string, error) { return "ok", nil }
	if err := HelmPull("https://example.com", "chart", "1.0", "/tmp", true); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestHelmPull_NonHTTPSkipsRepoUpdate(t *testing.T) {
	resetHelmPullFns(t)
	helmPullActionConfigInitFn = func(*action.Configuration, *cli.EnvSettings, func(string, ...interface{})) error { return nil }
	helmPullNewDefaultRegistryClientFn = func(bool, *cli.EnvSettings) (*registry.Client, error) { return nil, nil }
	helmPullMkdirAllFn = func(string, os.FileMode) error { return nil }
	helmPullConfigureVerificationFn = func(*action.Pull, string) {}
	helmPullResolveLinkFn = func(string, string) (string, string) { return "link", "" }
	called := false
	helmPullUpdateHelmRepoFn = func(string, string, bool) error { called = true; return nil }
	helmPullClientRunFn = func(*action.Pull, string) (string, error) { return "", nil }
	if err := HelmPull("oci://e.com", "chart", "1.0", "/tmp", false); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if called {
		t.Fatalf("expected updateHelmRepo not to be called for non-http repo")
	}
}

func TestHelmPull_InitError(t *testing.T) {
	resetHelmPullFns(t)
	helmPullActionConfigInitFn = func(*action.Configuration, *cli.EnvSettings, func(string, ...interface{})) error {
		return errors.New("init")
	}
	if err := HelmPull("repo", "c", "1", "", true); err == nil || !strings.Contains(err.Error(), "failed to initialize") {
		t.Fatalf("expected init error, got %v", err)
	}
}

func TestHelmPull_RegistryError(t *testing.T) {
	resetHelmPullFns(t)
	helmPullActionConfigInitFn = func(*action.Configuration, *cli.EnvSettings, func(string, ...interface{})) error { return nil }
	helmPullNewDefaultRegistryClientFn = func(bool, *cli.EnvSettings) (*registry.Client, error) { return nil, errors.New("reg") }
	if err := HelmPull("repo", "c", "1", "", true); err == nil || !strings.Contains(err.Error(), "reg") {
		t.Fatalf("expected registry error, got %v", err)
	}
}

func TestHelmPull_MkdirError(t *testing.T) {
	resetHelmPullFns(t)
	helmPullActionConfigInitFn = func(*action.Configuration, *cli.EnvSettings, func(string, ...interface{})) error { return nil }
	helmPullNewDefaultRegistryClientFn = func(bool, *cli.EnvSettings) (*registry.Client, error) { return nil, nil }
	helmPullMkdirAllFn = func(string, os.FileMode) error { return errors.New("mkdir") }
	if err := HelmPull("repo", "c", "1", "/tmp", false); err == nil || !strings.Contains(err.Error(), "Failed to create cache") {
		t.Fatalf("expected mkdir error, got %v", err)
	}
}

func TestHelmPull_RunError(t *testing.T) {
	resetHelmPullFns(t)
	helmPullActionConfigInitFn = func(*action.Configuration, *cli.EnvSettings, func(string, ...interface{})) error { return nil }
	helmPullNewDefaultRegistryClientFn = func(bool, *cli.EnvSettings) (*registry.Client, error) { return nil, nil }
	helmPullMkdirAllFn = func(string, os.FileMode) error { return nil }
	helmPullConfigureVerificationFn = func(*action.Pull, string) {}
	helmPullResolveLinkFn = func(string, string) (string, string) { return "link", "" }
	helmPullClientRunFn = func(*action.Pull, string) (string, error) { return "", errors.New("run") }
	helmPullRemoveFn = func(string) error { return nil }
	if err := HelmPull("oci://e.com", "c", "1", "/tmp", true); err == nil || !strings.Contains(err.Error(), "run") {
		t.Fatalf("expected run error, got %v", err)
	}
}

func TestHelmPull_KeyringSetMessage(t *testing.T) {
	resetHelmPullFns(t)
	helmPullActionConfigInitFn = func(*action.Configuration, *cli.EnvSettings, func(string, ...interface{})) error { return nil }
	helmPullNewDefaultRegistryClientFn = func(bool, *cli.EnvSettings) (*registry.Client, error) { return nil, nil }
	helmPullMkdirAllFn = func(string, os.FileMode) error { return nil }
	helmPullConfigureVerificationFn = func(c *action.Pull, _ string) { c.Keyring = "/tmp/key.gpg" }
	helmPullResolveLinkFn = func(string, string) (string, string) { return "link", "" }
	helmPullClientRunFn = func(*action.Pull, string) (string, error) { return "some output", nil }
	if err := HelmPull("oci://e.com", "c", "1", "/tmp", false); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestUpdateHelmRepo_Errors(t *testing.T) {
	t.Run("new chart repo error", func(t *testing.T) {
		resetHelmPullFns(t)
		helmPullRepoNewChartRepositoryFn = func(*repo.Entry, *cli.EnvSettings) (*repo.ChartRepository, error) {
			return nil, errors.New("ncr")
		}
		if err := updateHelmRepo("n", "u", true); err == nil || !strings.Contains(err.Error(), "failed to create chart repository") {
			t.Fatalf("expected chart repo error, got %v", err)
		}
	})

	t.Run("mkdir error", func(t *testing.T) {
		resetHelmPullFns(t)
		helmPullRepoNewChartRepositoryFn = func(*repo.Entry, *cli.EnvSettings) (*repo.ChartRepository, error) {
			return &repo.ChartRepository{}, nil
		}
		helmPullMkdirAllFn = func(string, os.FileMode) error { return errors.New("mk") }
		if err := updateHelmRepo("n", "u", true); err == nil || !strings.Contains(err.Error(), "failed to create cache") {
			t.Fatalf("expected mkdir error, got %v", err)
		}
	})

	t.Run("download index error", func(t *testing.T) {
		resetHelmPullFns(t)
		helmPullRepoNewChartRepositoryFn = func(*repo.Entry, *cli.EnvSettings) (*repo.ChartRepository, error) {
			return &repo.ChartRepository{}, nil
		}
		helmPullMkdirAllFn = func(string, os.FileMode) error { return nil }
		helmPullDownloadIndexFileFn = func(*repo.ChartRepository) error { return errors.New("dl") }
		if err := updateHelmRepo("n", "u", true); err == nil || !strings.Contains(err.Error(), "failed to download index") {
			t.Fatalf("expected download error, got %v", err)
		}
	})

	t.Run("load file error", func(t *testing.T) {
		resetHelmPullFns(t)
		helmPullRepoNewChartRepositoryFn = func(*repo.Entry, *cli.EnvSettings) (*repo.ChartRepository, error) {
			return &repo.ChartRepository{}, nil
		}
		helmPullMkdirAllFn = func(string, os.FileMode) error { return nil }
		helmPullDownloadIndexFileFn = func(*repo.ChartRepository) error { return nil }
		helmPullStatFn = func(string) (os.FileInfo, error) { return nil, nil }
		helmPullRepoLoadFileFn = func(string) (*repo.File, error) { return nil, errors.New("lf") }
		if err := updateHelmRepo("n", "u", true); err == nil || !strings.Contains(err.Error(), "failed to load repositories") {
			t.Fatalf("expected load error, got %v", err)
		}
	})

	t.Run("write file error", func(t *testing.T) {
		resetHelmPullFns(t)
		helmPullRepoNewChartRepositoryFn = func(*repo.Entry, *cli.EnvSettings) (*repo.ChartRepository, error) {
			return &repo.ChartRepository{}, nil
		}
		helmPullMkdirAllFn = func(string, os.FileMode) error { return nil }
		helmPullDownloadIndexFileFn = func(*repo.ChartRepository) error { return nil }
		helmPullStatFn = func(string) (os.FileInfo, error) { return nil, errors.New("nf") }
		helmPullRepoWriteFileFn = func(*repo.File, string) error { return errors.New("wf") }
		if err := updateHelmRepo("n", "u", true); err == nil || !strings.Contains(err.Error(), "failed to write repositories") {
			t.Fatalf("expected write error, got %v", err)
		}
	})

	t.Run("happy non-silent", func(t *testing.T) {
		resetHelmPullFns(t)
		helmPullRepoNewChartRepositoryFn = func(*repo.Entry, *cli.EnvSettings) (*repo.ChartRepository, error) {
			return &repo.ChartRepository{}, nil
		}
		helmPullMkdirAllFn = func(string, os.FileMode) error { return nil }
		helmPullDownloadIndexFileFn = func(*repo.ChartRepository) error { return nil }
		helmPullStatFn = func(string) (os.FileInfo, error) { return nil, errors.New("nf") }
		helmPullRepoWriteFileFn = func(*repo.File, string) error { return nil }
		if err := updateHelmRepo("n", "u", false); err != nil {
			t.Fatalf("unexpected: %v", err)
		}
	})
}

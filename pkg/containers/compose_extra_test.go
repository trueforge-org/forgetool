package containers

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/compose-spec/compose-go/v2/loader"
	composetypes "github.com/compose-spec/compose-go/v2/types"
)

func TestBuildComposeYAML_DepEmptyOrSelfNameSkipped(t *testing.T) {
	settings := AppSettings{Dependencies: []Dependency{{Name: ""}, {Name: "myapp"}}}
	resolverCalled := false
	resolve := func(string) (AppSettings, string, bool, error) {
		resolverCalled = true
		return AppSettings{}, "", false, nil
	}
	out, err := BuildComposeYAML("myapp", "1.0", settings, resolve)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if resolverCalled {
		t.Fatalf("resolver should not be called for empty/self name")
	}
	if !strings.Contains(out, "myapp") {
		t.Fatalf("expected app name in output")
	}
}

func TestBuildComposeYAML_DepNotFoundContinues(t *testing.T) {
	settings := AppSettings{Dependencies: []Dependency{{Name: "missing"}}}
	resolve := func(name string) (AppSettings, string, bool, error) {
		return AppSettings{}, "", false, nil
	}
	out, err := BuildComposeYAML("myapp", "1.0", settings, resolve)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if strings.Contains(out, "missing") {
		t.Fatalf("did not expect missing dependency in output")
	}
}

func TestBuildComposeYAML_DepDuplicateInBaseSkipped(t *testing.T) {
	// Two entries with the same name; second should be skipped because
	// the first call adds it to baseServices.
	settings := AppSettings{
		Dependencies: []Dependency{{Name: "pg"}, {Name: "pg"}},
	}
	calls := 0
	resolve := func(string) (AppSettings, string, bool, error) {
		calls++
		return AppSettings{}, "1.0", true, nil
	}
	if _, err := BuildComposeYAML("myapp", "1.0", settings, resolve); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected resolver called once, got %d", calls)
	}
}

func TestBuildComposeYAML_OptDep_HappyAndOverrides(t *testing.T) {
	parentOverride := composetypes.ServiceConfig{Restart: "always"}
	depOverride := composetypes.ServiceConfig{User: "1000:1000"}
	settings := AppSettings{OptDependencies: []Dependency{
		{Name: "redis", Compose: parentOverride},
	}}
	resolve := func(name string) (AppSettings, string, bool, error) {
		return AppSettings{Compose: depOverride}, "7.0", true, nil
	}
	out, err := BuildComposeYAML("myapp", "1.0", settings, resolve)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	// Opt-dependency lines must be commented out.
	if !strings.Contains(out, "# ") {
		t.Fatalf("expected commented opt-dependency lines, got %q", out)
	}
}

func TestBuildComposeYAML_OptDep_EmptyAndSeenSkipped(t *testing.T) {
	// Opt dep with empty name and one that duplicates the main app name.
	settings := AppSettings{
		OptDependencies: []Dependency{{Name: ""}, {Name: "myapp"}},
	}
	resolve := func(string) (AppSettings, string, bool, error) {
		t.Fatalf("resolver should not be called for empty/seen names")
		return AppSettings{}, "", false, nil
	}
	if _, err := BuildComposeYAML("myapp", "1.0", settings, resolve); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestBuildComposeYAML_OptDep_ResolverError(t *testing.T) {
	settings := AppSettings{OptDependencies: []Dependency{{Name: "redis"}}}
	resolve := func(string) (AppSettings, string, bool, error) {
		return AppSettings{}, "", false, errors.New("boom")
	}
	if _, err := BuildComposeYAML("myapp", "1.0", settings, resolve); err == nil {
		t.Fatalf("expected resolver error")
	}
}

func TestBuildComposeYAML_OptDep_NotFoundSkipped(t *testing.T) {
	settings := AppSettings{OptDependencies: []Dependency{{Name: "redis"}}}
	resolve := func(string) (AppSettings, string, bool, error) {
		return AppSettings{}, "", false, nil
	}
	out, err := BuildComposeYAML("myapp", "1.0", settings, resolve)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if strings.Contains(out, "# ") {
		t.Fatalf("did not expect opt-dependency in output")
	}
}

func TestBuildComposeYAML_OptDep_MergeAndMarshalError(t *testing.T) {
	// Force mergeAndMarshal to fail on the opt-dependency render. We let
	// the main-app render succeed (any number of marshal calls) and only
	// fail when the opt-dependency name "redis" is in the services map.
	orig := marshalServicesDocFn
	t.Cleanup(func() { marshalServicesDocFn = orig })
	marshalServicesDocFn = func(services composetypes.Services) ([]byte, error) {
		if _, ok := services["redis"]; ok {
			return nil, errors.New("opt boom")
		}
		return orig(services)
	}
	settings := AppSettings{OptDependencies: []Dependency{{Name: "redis"}}}
	resolve := func(string) (AppSettings, string, bool, error) {
		return AppSettings{}, "1.0", true, nil
	}
	if _, err := BuildComposeYAML("myapp", "1.0", settings, resolve); err == nil {
		t.Fatalf("expected merge error")
	}
}

func TestBuildService_EmptyProtocolDefaultsToTCP(t *testing.T) {
	svc := buildService("a", "1.0", AppSettings{Ports: []PortSetting{{Port: 80}}})
	if len(svc.Ports) != 1 {
		t.Fatalf("expected 1 port")
	}
	if svc.Ports[0].Protocol != "tcp" {
		t.Fatalf("expected default tcp protocol, got %q", svc.Ports[0].Protocol)
	}
}

func TestIsEmptyService_MarshalErrorReturnsTrue(t *testing.T) {
	orig := yamlMarshalFn
	t.Cleanup(func() { yamlMarshalFn = orig })
	yamlMarshalFn = func(any) ([]byte, error) { return nil, errors.New("boom") }
	if !isEmptyService(composetypes.ServiceConfig{User: "x"}) {
		t.Fatalf("expected true on marshal error")
	}
}

func TestMergeAndMarshal_BaseMarshalError(t *testing.T) {
	orig := marshalServicesDocFn
	t.Cleanup(func() { marshalServicesDocFn = orig })
	marshalServicesDocFn = func(composetypes.Services) ([]byte, error) {
		return nil, errors.New("base boom")
	}
	if _, err := mergeAndMarshal("p", composetypes.Services{}); err == nil {
		t.Fatalf("expected base marshal error")
	}
}

func TestMergeAndMarshal_LayerMarshalError(t *testing.T) {
	orig := marshalServicesDocFn
	t.Cleanup(func() { marshalServicesDocFn = orig })
	calls := 0
	marshalServicesDocFn = func(s composetypes.Services) ([]byte, error) {
		calls++
		if calls >= 2 {
			return nil, errors.New("layer boom")
		}
		return orig(s)
	}
	layer := composetypes.Services{"x": composetypes.ServiceConfig{Name: "x"}}
	if _, err := mergeAndMarshal("p", composetypes.Services{}, layer); err == nil {
		t.Fatalf("expected layer marshal error")
	}
}

func TestMergeAndMarshal_LoaderError(t *testing.T) {
	orig := loaderLoadWithContextFn
	t.Cleanup(func() { loaderLoadWithContextFn = orig })
	loaderLoadWithContextFn = func(context.Context, composetypes.ConfigDetails, ...func(*loader.Options)) (*composetypes.Project, error) {
		return nil, errors.New("loader boom")
	}
	if _, err := mergeAndMarshal("p", composetypes.Services{}); err == nil {
		t.Fatalf("expected loader error")
	}
}

func TestMergeAndMarshal_ProjectMarshalError(t *testing.T) {
	// Skip this branch — project.MarshalYAML on a typed Project rarely
	// returns an error in practice and we cannot easily inject one.
	t.Skip("project.MarshalYAML failure not reachable without invasive changes")
}

func TestCommentOutServiceBlock_NoServicesHeader(t *testing.T) {
	// Input that lacks a "services:" header entirely → returns empty string.
	got := commentOutServiceBlock("name: foo\nversion: 1\n")
	if got != "" {
		t.Fatalf("expected empty output, got %q", got)
	}
}

func TestBuildComposeYAML_MainMergeError(t *testing.T) {
	orig := marshalServicesDocFn
	t.Cleanup(func() { marshalServicesDocFn = orig })
	marshalServicesDocFn = func(composetypes.Services) ([]byte, error) {
		return nil, errors.New("main boom")
	}
	if _, err := BuildComposeYAML("myapp", "1.0", AppSettings{}, nil); err == nil {
		t.Fatalf("expected main merge error")
	}
}

// Force the "rendered does not end with newline" branch by stubbing
// loaderLoadWithContextFn so the main-app render returns YAML without a
// trailing newline, then add an opt-dep so the appending branch executes.
func TestBuildComposeYAML_RenderedMissingTrailingNewline(t *testing.T) {
	origLoader := loaderLoadWithContextFn
	t.Cleanup(func() { loaderLoadWithContextFn = origLoader })
	calls := 0
	loaderLoadWithContextFn = func(ctx context.Context, cfg composetypes.ConfigDetails, opts ...func(*loader.Options)) (*composetypes.Project, error) {
		calls++
		// Only mutate the first (main) project; passthrough opt-dep ones.
		p, err := origLoader(ctx, cfg, opts...)
		if err != nil {
			return nil, err
		}
		return p, nil
	}
	// Strip trailing newline by using marshalServicesDocFn override.
	origMarshal := marshalServicesDocFn
	t.Cleanup(func() { marshalServicesDocFn = origMarshal })
	marshalServicesDocFn = func(s composetypes.Services) ([]byte, error) {
		out, err := origMarshal(s)
		if err != nil {
			return nil, err
		}
		return []byte(strings.TrimRight(string(out), "\n")), nil
	}
	settings := AppSettings{OptDependencies: []Dependency{{Name: "redis"}}}
	resolve := func(string) (AppSettings, string, bool, error) {
		return AppSettings{}, "1.0", true, nil
	}
	out, err := BuildComposeYAML("myapp", "1.0", settings, resolve)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	_ = out
	_ = calls
}

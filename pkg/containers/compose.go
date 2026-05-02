package containers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/v2/loader"
	composetypes "github.com/compose-spec/compose-go/v2/types"
	"gopkg.in/yaml.v3"
)

// passPlaceholderRE matches placeholders of the form MY<NAME>PASS that the
// generator substitutes with a freshly generated random secret. NAME is one
// or more uppercase alphanumeric/underscore characters, typically the
// uppercased name of a dependency (e.g. MYPOSTGRESPASS), but the
// substitution is purely textual so any token matching the pattern is
// handled.
var passPlaceholderRE = regexp.MustCompile(`MY[A-Z0-9_]+PASS`)

// randomSecretFn produces the random secret used to replace each unique
// MY<NAME>PASS placeholder. Exposed as a package variable so tests can
// substitute a deterministic generator.
var randomSecretFn = func() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// substitutePassPlaceholders replaces every occurrence of MY<NAME>PASS in
// the rendered compose document with a random secret. All occurrences of
// the same placeholder share the same secret within a single document, but
// different placeholders (e.g. MYPOSTGRESPASS vs MYREDISPASS) get distinct
// secrets, and each invocation of BuildComposeYAML generates fresh ones.
func substitutePassPlaceholders(rendered string) (string, error) {
	matches := passPlaceholderRE.FindAllString(rendered, -1)
	if len(matches) == 0 {
		return rendered, nil
	}
	secrets := make(map[string]string)
	for _, m := range matches {
		if _, ok := secrets[m]; ok {
			continue
		}
		s, err := randomSecretFn()
		if err != nil {
			return "", fmt.Errorf("generate secret for %s: %w", m, err)
		}
		secrets[m] = s
	}
	return passPlaceholderRE.ReplaceAllStringFunc(rendered, func(m string) string {
		return secrets[m]
	}), nil
}

// DependencyResolver resolves a dependency name (which is the directory name
// of another app in the same repository) to that app's settings and version
// tag. It returns found=false to signal that the dependency cannot be
// resolved; in that case the dependency is skipped silently.
type DependencyResolver func(name string) (settings AppSettings, version string, found bool, err error)

// BuildComposeYAML assembles a docker-compose document from the app settings
// using the canonical compose-spec types and renders it as YAML.
//
// Required dependencies declared in settings.dependencies are resolved via
// the provided DependencyResolver and added as additional services to the
// same compose file. Each Dependency entry has a "name" field (used to look
// up the dependency app and as the service name) and an optional "compose"
// field — a docker-compose service-spec fragment whose fields are merged on
// top of the service generated from the dependency's own settings.yaml.
//
// Optional dependencies declared in settings.opt_dependencies follow the same
// merge rules but are appended at the end as commented-out service blocks so
// that a user can uncomment them to enable the related service.
//
// The image tag falls back to "rolling" when no usable version is provided
// (empty or "latest"). Passing a nil resolver disables dependency expansion.
func BuildComposeYAML(app, version string, settings AppSettings, resolve DependencyResolver) (string, error) {
	baseServices := composetypes.Services{
		app: buildService(app, version, settings),
	}
	// Override layers, applied in order on top of baseServices using the
	// canonical compose-spec merge semantics.
	depOwnOverrides := composetypes.Services{}
	parentOverrides := composetypes.Services{}

	// Top-level "compose" override for the app's own service.
	if !isEmptyService(settings.Compose) {
		ovr := settings.Compose
		ovr.Name = app
		depOwnOverrides[app] = ovr
	}

	for _, dep := range settings.Dependencies {
		name := dep.Name
		if name == "" || name == app || resolve == nil {
			continue
		}
		if _, exists := baseServices[name]; exists {
			continue
		}
		depSettings, depVersion, found, err := resolve(name)
		if err != nil {
			return "", fmt.Errorf("resolve dependency %q: %w", name, err)
		}
		if !found {
			continue
		}
		baseServices[name] = buildService(name, depVersion, depSettings)
		// Layer 2: the dependency's own top-level "compose" override.
		if !isEmptyService(depSettings.Compose) {
			ovr := depSettings.Compose
			ovr.Name = name
			depOwnOverrides[name] = ovr
		}
		// Layer 3: the override declared by the parent app on this
		// dependency entry.
		if !isEmptyService(dep.Compose) {
			ovr := dep.Compose
			ovr.Name = name
			parentOverrides[name] = ovr
		}
	}

	rendered, err := mergeAndMarshal(app, baseServices, depOwnOverrides, parentOverrides)
	if err != nil {
		return "", err
	}

	if resolve != nil && len(settings.OptDependencies) > 0 {
		var commented strings.Builder
		seen := map[string]bool{}
		for n := range baseServices {
			seen[n] = true
		}
		for _, dep := range settings.OptDependencies {
			name := dep.Name
			if name == "" || seen[name] {
				continue
			}
			depSettings, depVersion, found, err := resolve(name)
			if err != nil {
				return "", fmt.Errorf("resolve opt_dependency %q: %w", name, err)
			}
			if !found {
				continue
			}
			seen[name] = true
			base := composetypes.Services{name: buildService(name, depVersion, depSettings)}
			depOwn := composetypes.Services{}
			parent := composetypes.Services{}
			if !isEmptyService(depSettings.Compose) {
				ovr := depSettings.Compose
				ovr.Name = name
				depOwn[name] = ovr
			}
			if !isEmptyService(dep.Compose) {
				ovr := dep.Compose
				ovr.Name = name
				parent[name] = ovr
			}
			depYAML, err := mergeAndMarshal(name, base, depOwn, parent)
			if err != nil {
				return "", fmt.Errorf("render opt_dependency %q: %w", name, err)
			}
			commented.WriteString(commentOutServiceBlock(depYAML))
		}
		if commented.Len() > 0 {
			rendered += commented.String()
		}
	}

	return substitutePassPlaceholders(rendered)
}

// buildService converts an app's settings into a single ServiceConfig entry.
func buildService(app, version string, settings AppSettings) composetypes.ServiceConfig {
	tag := version
	if tag == "" || strings.EqualFold(tag, "latest") {
		tag = "rolling"
	}

	svc := composetypes.ServiceConfig{
		Name:          app,
		Image:         "ghcr.io/trueforge-org/" + app + ":" + tag,
		ContainerName: app,
		Restart:       "unless-stopped",
	}

	for _, p := range settings.Ports {
		proto := strings.ToLower(p.Protocol)
		if proto == "" {
			proto = "tcp"
		}
		// Bind to loopback by default so generated stacks are not
		// accidentally exposed on LAN/internet interfaces. Operators
		// who want a different bind address (e.g. 0.0.0.0 or a
		// specific NIC) can override this via the service's
		// `compose:` block in settings.yaml.
		svc.Ports = append(svc.Ports, composetypes.ServicePortConfig{
			HostIP:    "127.0.0.1",
			Target:    uint32(p.Port),
			Published: strconv.Itoa(p.Port),
			Protocol:  proto,
		})
	}

	if len(settings.Env) > 0 {
		env := make(composetypes.MappingWithEquals, len(settings.Env))
		for _, e := range settings.Env {
			val := e.Default
			env[e.Name] = &val
		}
		svc.Environment = env
	}

	// Bind-mount host paths are namespaced under the service name beneath
	// a fixed absolute base directory so that dependencies whose volume
	// targets share a basename with the app or each other (e.g. multiple
	// "/data" mounts) do not collide on the host filesystem. Using an
	// absolute path also keeps the host layout independent of the
	// directory the compose file happens to live in. Operators who want
	// a different on-disk layout can override the source via the
	// service's `compose:` block in settings.yaml.
	for _, v := range settings.Volumes {
		hostName := filepath.Base(v.Path)
		svc.Volumes = append(svc.Volumes, composetypes.ServiceVolumeConfig{
			Type:   composetypes.VolumeTypeBind,
			Source: "/mnt/tank/apps/" + app + "/" + hostName,
			Target: v.Path,
		})
	}

	return svc
}

// serviceToMap was previously needed to convert ServiceConfig values to
// generic maps for the loader. Since dependency overrides are now typed
// compose ServiceConfig values, we can build typed Projects directly and
// hand them to the loader as raw YAML content.

// marshalServicesDoc renders a typed compose Services map as a YAML document
// of the shape `{services: {...}}`, ready to be ingested by the compose-go
// loader. Going through the typed ServiceConfig.MarshalYAML guarantees that
// the canonical compose-spec representation is used for every field.
var marshalServicesDocFn = marshalServicesDoc

func marshalServicesDoc(services composetypes.Services) ([]byte, error) {
	project := &composetypes.Project{Services: services}
	return project.MarshalYAML()
}

// isEmptyService reports whether a ServiceConfig has any user-supplied fields
// that should be applied as a compose override. The Name field is set by the
// caller and ignored here.
var yamlMarshalFn = yaml.Marshal

var loaderLoadWithContextFn = loader.LoadWithContext

func isEmptyService(svc composetypes.ServiceConfig) bool {
	svc.Name = ""
	out, err := yamlMarshalFn(svc)
	if err != nil {
		return true
	}
	trimmed := strings.TrimSpace(string(out))
	return trimmed == "" || trimmed == "{}"
}

// projectMarshalYAMLFn is an indirection seam to enable testing the
// post-merge marshal error branch in mergeAndMarshal.
var projectMarshalYAMLFn = func(p *composetypes.Project) ([]byte, error) {
	return p.MarshalYAML()
}

// mergeAndMarshal merges the given base services with zero or more override
// layers using the compose-go loader (which implements the canonical
// compose-spec merge semantics) and returns the resulting YAML document.
// Override layers are applied in order, so later layers win over earlier
// ones. Empty layers are skipped.
func mergeAndMarshal(projectName string, baseServices composetypes.Services, overrideLayers ...composetypes.Services) (string, error) {
	baseYAML, err := marshalServicesDocFn(baseServices)
	if err != nil {
		return "", fmt.Errorf("marshal base compose: %w", err)
	}

	configFiles := []composetypes.ConfigFile{
		{Filename: "base.yaml", Content: baseYAML},
	}
	for i, layer := range overrideLayers {
		if len(layer) == 0 {
			continue
		}
		layerYAML, err := marshalServicesDocFn(layer)
		if err != nil {
			return "", fmt.Errorf("marshal override layer %d: %w", i, err)
		}
		configFiles = append(configFiles, composetypes.ConfigFile{
			Filename: fmt.Sprintf("override-%d.yaml", i),
			Content:  layerYAML,
		})
	}

	cfg := composetypes.ConfigDetails{
		ConfigFiles: configFiles,
		Environment: composetypes.Mapping{},
	}

	project, err := loaderLoadWithContextFn(context.Background(), cfg, func(o *loader.Options) {
		o.SetProjectName(projectName, true)
		o.SkipValidation = true
		o.SkipNormalization = true
		o.SkipConsistencyCheck = true
		o.SkipInterpolation = true
		o.SkipResolveEnvironment = true
	})
	if err != nil {
		return "", fmt.Errorf("merge compose configs: %w", err)
	}

	out, err := projectMarshalYAMLFn(project)
	if err != nil {
		return "", fmt.Errorf("marshal merged compose: %w", err)
	}
	return string(out), nil
}

// commentOutServiceBlock takes the marshalled YAML of a single-service
// compose project and returns the body of its "services:" map with every
// non-empty line prefixed by "# ", so the result can be appended after an
// existing services block to surface the service as a commented-out, opt-in
// entry.
func commentOutServiceBlock(in string) string {
	lines := strings.Split(in, "\n")
	var out []string
	inServices := false
	for _, l := range lines {
		if !inServices {
			if strings.TrimSpace(l) == "services:" {
				inServices = true
			}
			continue
		}
		if strings.TrimSpace(l) == "" {
			continue
		}
		out = append(out, "# "+l)
	}
	if len(out) == 0 {
		return ""
	}
	return strings.Join(out, "\n") + "\n"
}

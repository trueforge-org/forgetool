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

// hostIPLineRE matches the rendered `host_ip:` line inside a port entry.
// We comment these lines out so the generated stack publishes on all
// interfaces by default while keeping the loopback hint one keystroke
// away for operators who want to restrict the bind. Indentation is
// preserved so the commented line stays aligned with its siblings.
var hostIPLineRE = regexp.MustCompile(`(?m)^(\s*)(host_ip:.*)$`)

// commentOutHostIP rewrites every `host_ip:` line into `# host_ip:` while
// keeping the original indentation, surfacing the field as an opt-in hint
// rather than a hard binding.
func commentOutHostIP(in string) string {
	return hostIPLineRE.ReplaceAllString(in, "$1# $2")
}

// volumeTargetLineRE matches the `target: /<path>` line inside a rendered
// volume entry. Port entries also render a `target:` field but with a
// numeric value, so anchoring the value on a leading slash distinguishes
// the two without needing a stateful YAML walker.
var volumeTargetLineRE = regexp.MustCompile(`(?m)^(\s*)(target:\s*/[^\n]*)$`)

// addReadOnlyDefault appends `read_only: false` to every rendered volume
// block so operators can flip the mount to read-only by editing a single
// boolean instead of having to remember the field name. The line is
// indented to match the sibling `target:` so it stays inside the same
// volume entry.
func addReadOnlyDefault(in string) string {
	return volumeTargetLineRE.ReplaceAllString(in, "${1}${2}\n${1}read_only: false")
}

// injectPrivilegedDefault makes the `privileged:` flag visible on every
// generated service. compose-go's typed marshaller drops the field when it
// is false (it has the `omitempty` tag), so without this step operators
// would have to remember the field name to opt a service in. We walk the
// rendered document service-by-service and insert `privileged: false`
// right after the service's `container_name:` line, but only when no
// `privileged:` line already exists in that service block (e.g. because
// the user set it via the typed field or a `compose:` override).
func injectPrivilegedDefault(in string) string {
	lines := strings.Split(in, "\n")
	// Service blocks are rendered with two-space indentation under the
	// top-level "services:" key, so each "  <name>:" line at exactly two
	// leading spaces marks a new service. The body uses four leading
	// spaces.
	type segment struct {
		start, end int // [start, end) line range of the service body
		nameIdx    int // index of the "  <name>:" header line
	}
	var segs []segment
	inServices := false
	cur := -1
	curName := -1
	for i, l := range lines {
		if !inServices {
			if strings.TrimSpace(l) == "services:" {
				inServices = true
			}
			continue
		}
		// A line that starts a top-level (zero-indent) key ends the
		// services block.
		if len(l) > 0 && l[0] != ' ' && l[0] != '#' && strings.Contains(l, ":") {
			if cur != -1 {
				segs = append(segs, segment{cur, i, curName})
				cur = -1
			}
			inServices = false
			continue
		}
		// Service header: exactly two leading spaces, then "<name>:".
		if strings.HasPrefix(l, "  ") && !strings.HasPrefix(l, "   ") && strings.HasSuffix(strings.TrimSpace(l), ":") {
			if cur != -1 {
				segs = append(segs, segment{cur, i, curName})
			}
			cur = i + 1
			curName = i
		}
	}
	if cur != -1 {
		segs = append(segs, segment{cur, len(lines), curName})
	}

	type insertion struct {
		afterLine int
		text      string
	}
	var inserts []insertion
	for _, s := range segs {
		hasPrivileged := false
		containerNameLine := -1
		for i := s.start; i < s.end; i++ {
			t := strings.TrimSpace(lines[i])
			if strings.HasPrefix(t, "privileged:") {
				hasPrivileged = true
				break
			}
			if containerNameLine == -1 && strings.HasPrefix(t, "container_name:") {
				containerNameLine = i
			}
		}
		if hasPrivileged {
			continue
		}
		anchor := containerNameLine
		if anchor == -1 {
			anchor = s.nameIdx
		}
		inserts = append(inserts, insertion{anchor, "    privileged: false"})
	}
	if len(inserts) == 0 {
		return in
	}
	// Apply insertions from the bottom up so earlier indices stay valid.
	for i := len(inserts) - 1; i >= 0; i-- {
		ins := inserts[i]
		lines = append(lines[:ins.afterLine+1], append([]string{ins.text}, lines[ins.afterLine+1:]...)...)
	}
	return strings.Join(lines, "\n")
}

// UnitBytes.MarshalYAML, which always renders as a quoted decimal byte
// count. We rewrite those into the short compose-spec suffix form
// (e.g. "1G", "512M") for any value that is an exact multiple of a
// power-of-two unit, leaving non-round values as the raw byte count.
var memoryBytesRE = regexp.MustCompile(`(?m)^(\s*(?:memory|shm_size):\s*)"(\d+)"[ \t]*$`)

// humanizeMemoryValues rewrites quoted byte counts on `memory:` lines to
// human-readable compose-spec sizes (G/M/K) when the value divides cleanly,
// keeping the canonical byte form otherwise.
func humanizeMemoryValues(in string) string {
	return memoryBytesRE.ReplaceAllStringFunc(in, func(line string) string {
		m := memoryBytesRE.FindStringSubmatch(line)
		if len(m) != 3 {
			return line
		}
		n, err := strconv.ParseInt(m[2], 10, 64)
		if err != nil || n <= 0 {
			return line
		}
		const (
			kib = 1024
			mib = 1024 * kib
			gib = 1024 * mib
		)
		switch {
		case n%gib == 0:
			return fmt.Sprintf("%s%dG", m[1], n/gib)
		case n%mib == 0:
			return fmt.Sprintf("%s%dM", m[1], n/mib)
		case n%kib == 0:
			return fmt.Sprintf("%s%dK", m[1], n/kib)
		}
		return line
	})
}

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

	rendered = humanizeMemoryValues(rendered)
	rendered = commentOutHostIP(rendered)
	rendered = addReadOnlyDefault(rendered)
	rendered = injectPrivilegedDefault(rendered)
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
		// Add the shared "apps" GID (568) so processes inside the
		// container can read/write the host bind mounts without
		// having to match the container's primary UID exactly.
		GroupAdd: []string{"568"},
		// Drop all Linux capabilities by default. Apps that genuinely
		// need a capability can re-add it via the `compose:` block in
		// settings.yaml (e.g. cap_add: [NET_BIND_SERVICE]).
		CapDrop: []string{"ALL"},
		// Apps may declare a per-app capability allow-list directly
		// in settings.yaml via top-level `cap_add:`. The values are
		// passed through verbatim and merged with the default
		// `cap_drop: [ALL]` above; an empty list means no caps are
		// re-added.
		CapAdd: append([]string(nil), settings.CapAdd...),
		// Privileged is only set when the user explicitly opts in;
		// the typed field is `bool` with `omitempty`, so an unset
		// (nil) value renders nothing here and the
		// injectPrivilegedDefault post-processor adds the explicit
		// `privileged: false` line for visibility.
		Privileged: settings.Privileged != nil && *settings.Privileged,
		// /dev/shm size: honour any per-app override, otherwise fall
		// back to the defaultShmSize constant. compose-go's
		// MarshalYAML renders this as a quoted byte count which the
		// humanizeMemoryValues post-processor rewrites to the short
		// suffix form (e.g. "256M").
		ShmSize: shmSizeOrDefault(settings.ShmSize),
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

	svc.Deploy = applyResourceDefaults(settings.Resources)

	return svc
}

// defaultCPULimit and defaultMemoryLimit are the conservative resource caps
// applied to every generated service. They are intentionally generous
// enough for typical self-hosted apps but small enough to prevent a single
// runaway container from starving the host. Per-app overrides go through
// the `resources:` block in settings.yaml or the service's `compose:`
// override.
const (
	defaultCPULimit    composetypes.NanoCPUs  = 4.0
	defaultMemoryLimit composetypes.UnitBytes = 4 << 30   // 4 GiB
	defaultShmSize     composetypes.UnitBytes = 256 << 20 // 256 MiB
)

// shmSizeOrDefault returns the user's per-app /dev/shm override or the
// generator's defaultShmSize when none is set.
func shmSizeOrDefault(user *composetypes.UnitBytes) composetypes.UnitBytes {
	if user != nil && *user > 0 {
		return *user
	}
	return defaultShmSize
}

// applyResourceDefaults returns a *DeployConfig populated with the user's
// resource settings, falling back to defaultCPULimit / defaultMemoryLimit
// for any limit field the user left unset. Reservations are passed through
// untouched (no default) so apps that don't request guaranteed capacity
// stay schedulable.
func applyResourceDefaults(user composetypes.Resources) *composetypes.DeployConfig {
	limits := composetypes.Resource{
		NanoCPUs:    defaultCPULimit,
		MemoryBytes: defaultMemoryLimit,
	}
	if user.Limits != nil {
		if user.Limits.NanoCPUs != 0 {
			limits.NanoCPUs = user.Limits.NanoCPUs
		}
		if user.Limits.MemoryBytes != 0 {
			limits.MemoryBytes = user.Limits.MemoryBytes
		}
		limits.Pids = user.Limits.Pids
		limits.Devices = user.Limits.Devices
		limits.GenericResources = user.Limits.GenericResources
	}
	return &composetypes.DeployConfig{
		Resources: composetypes.Resources{
			Limits:       &limits,
			Reservations: user.Reservations,
		},
	}
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

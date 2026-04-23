package website

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// AppSettings represents the contents of an app's settings.yaml file.
type AppSettings struct {
	SchemaVersion  int             `yaml:"schema_version"`
	UpstreamEnvURL string          `yaml:"upstream_env_url"`
	Ports          []PortSetting   `yaml:"ports"`
	Env            []EnvSetting    `yaml:"env"`
	Volumes        []VolumeSetting `yaml:"volumes"`
}

// PortSetting is a single port entry in settings.yaml.
type PortSetting struct {
	Port     int    `yaml:"port"`
	Protocol string `yaml:"protocol"`
	Required bool   `yaml:"required"`
}

// EnvSetting is a single environment variable entry in settings.yaml.
type EnvSetting struct {
	Name     string `yaml:"name"`
	Default  string `yaml:"default"`
	Required bool   `yaml:"required"`
}

// VolumeSetting is a single volume entry in settings.yaml.
type VolumeSetting struct {
	Path     string `yaml:"path"`
	Required bool   `yaml:"required"`
}

// ContainerOptions configures a single container app docs build.
type ContainerOptions struct {
	// App is the application name (the directory under AppsDir).
	App string
	// AppsDir is the directory containing per-app sources. Defaults to "apps".
	AppsDir string
	// WebsiteDir is the root of the website checkout. Defaults to "website".
	WebsiteDir string
	// TemplatePath is the README/index template. Defaults to
	// "templates/README.md.tmpl".
	TemplatePath string
	// ComposeTemplatePath is the docker-compose snippet template. Defaults to
	// "templates/docker-compose.yaml.tmpl". When the template file is missing
	// no docker-compose.md page is generated.
	ComposeTemplatePath string
	// IconFallbackBaseURL is queried for icons when the app does not provide a
	// local icon.webp / icon-small.webp. Defaults to the truecharts charts/stable
	// raw URL used by the legacy script.
	IconFallbackBaseURL string
}

const defaultIconFallbackBaseURL = "https://raw.githubusercontent.com/trueforge-org/truecharts/refs/heads/master/charts/stable"

func (o *ContainerOptions) applyDefaults() {
	if o.AppsDir == "" {
		o.AppsDir = "apps"
	}
	if o.WebsiteDir == "" {
		o.WebsiteDir = "website"
	}
	if o.TemplatePath == "" {
		o.TemplatePath = "templates/README.md.tmpl"
	}
	if o.ComposeTemplatePath == "" {
		o.ComposeTemplatePath = "templates/docker-compose.yaml.tmpl"
	}
	if o.IconFallbackBaseURL == "" {
		o.IconFallbackBaseURL = defaultIconFallbackBaseURL
	}
}

// containerPaths bundles the derived file system locations for one app.
type containerPaths struct {
	bake        string
	appDir      string
	docsSrc     string
	readme      string
	screenshots string
	settings    string

	docsBase    string
	tmpDocsBase string
	docsDir     string
	indexFile   string

	iconsDir       string
	iconsSmallDir  string
	screenshotsDir string
}

func (o *ContainerOptions) paths() containerPaths {
	docsBase := filepath.Join(o.WebsiteDir, "containerforge", "src", "content", "docs", "containers")
	tmpDocsBase := filepath.Join("tmpwebsite", "src", "content", "docs", "containers")
	appDir := filepath.Join(o.AppsDir, o.App)
	return containerPaths{
		bake:           filepath.Join(appDir, "docker-bake.hcl"),
		appDir:         appDir,
		docsSrc:        filepath.Join(appDir, "docs"),
		readme:         filepath.Join(appDir, "README.md"),
		screenshots:    filepath.Join(appDir, "screenshots"),
		settings:       filepath.Join(appDir, "settings.yaml"),
		docsBase:       docsBase,
		tmpDocsBase:    tmpDocsBase,
		docsDir:        filepath.Join(docsBase, o.App),
		indexFile:      filepath.Join(docsBase, o.App, "index.md"),
		iconsDir:       filepath.Join(o.WebsiteDir, "containerforge", "public", "img", "hotlink-ok", "container-icons"),
		iconsSmallDir:  filepath.Join(o.WebsiteDir, "containerforge", "public", "img", "hotlink-ok", "container-icons-small"),
		screenshotsDir: filepath.Join(o.WebsiteDir, "containerforge", "public", "img", "hotlink-ok", "container-screenshots", o.App),
	}
}

// ProcessApp regenerates the website documentation for a single container app.
// It is the Go equivalent of running .github/scripts/container-docs.sh APP.
func ProcessApp(opts ContainerOptions) error {
	opts.applyDefaults()
	if opts.App == "" {
		return errors.New("website: App must be set")
	}
	p := opts.paths()

	if _, err := os.Stat(p.bake); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Info().Msgf("docker-bake.hcl does not exist for %s, skipping", opts.App)
			return nil
		}
		return err
	}

	log.Info().Msgf("copying docs to website for %s", opts.App)

	if err := keepDocsSafe(p.docsBase, p.tmpDocsBase, opts.App); err != nil {
		return err
	}
	if err := resetDir(p.docsDir); err != nil {
		return err
	}
	if err := copyContainerAssets(opts, p); err != nil {
		return err
	}
	if err := processContainerIndex(opts, p); err != nil {
		return err
	}
	if err := restoreSafeDocs(p.docsBase, p.tmpDocsBase, opts.App); err != nil {
		return err
	}
	log.Info().Msgf("finished processing %s", opts.App)
	return nil
}

func copyContainerAssets(opts ContainerOptions, p containerPaths) error {
	if err := copyTreeContents(p.docsSrc, p.docsDir); err != nil {
		return fmt.Errorf("copy docs: %w", err)
	}

	// Icons: prefer local, fall back to a remote URL (best effort, not fatal).
	iconLocal := filepath.Join(p.appDir, "icon.webp")
	iconDst := filepath.Join(p.iconsDir, opts.App+".webp")
	if _, err := os.Stat(iconLocal); err == nil {
		if cpErr := copyFileIfExists(iconLocal, iconDst); cpErr != nil {
			return cpErr
		}
	} else {
		url := strings.TrimRight(opts.IconFallbackBaseURL, "/") + "/" + opts.App + "/icon.webp"
		if dlErr := downloadFile(url, iconDst); dlErr != nil {
			log.Info().Msgf("no chart icon found for %s: %v", opts.App, dlErr)
		}
	}

	iconSmallLocal := filepath.Join(p.appDir, "icon-small.webp")
	iconSmallDst := filepath.Join(p.iconsSmallDir, opts.App+".webp")
	if _, err := os.Stat(iconSmallLocal); err == nil {
		if cpErr := copyFileIfExists(iconSmallLocal, iconSmallDst); cpErr != nil {
			return cpErr
		}
	} else {
		url := strings.TrimRight(opts.IconFallbackBaseURL, "/") + "/" + opts.App + "/icon-small.webp"
		if dlErr := downloadFile(url, iconSmallDst); dlErr != nil {
			log.Info().Msgf("no chart icon-small found for %s: %v", opts.App, dlErr)
		}
	}

	if err := copyTreeContents(p.screenshots, p.screenshotsDir); err != nil {
		return fmt.Errorf("copy screenshots: %w", err)
	}
	return nil
}

// HCL variable parsing — these regexes mirror the awk/sed snippets used by the
// legacy bash script.
var (
	bakeVariableRe = regexp.MustCompile(`^\s*variable\s+"(\w+)"\s*\{`)
	bakeDefaultRe  = regexp.MustCompile(`^\s*default\s*=\s*"(.+)"`)
)

func parseBakeVars(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	vars := map[string]string{}
	var current string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if m := bakeVariableRe.FindStringSubmatch(line); m != nil {
			current = m[1]
			continue
		}
		if current == "" {
			continue
		}
		if m := bakeDefaultRe.FindStringSubmatch(line); m != nil {
			vars[current] = m[1]
			current = ""
		}
	}
	return vars, scanner.Err()
}

func processContainerIndex(opts ContainerOptions, p containerPaths) error {
	vars, err := parseBakeVars(p.bake)
	if err != nil {
		return fmt.Errorf("parse %s: %w", p.bake, err)
	}

	// Generate the standalone docker-compose.md page first so that it shows
	// up in the sidebar links collected below.
	if err := writeComposePage(opts, p, vars); err != nil {
		return err
	}

	links, err := collectDocsLinks(p.docsDir)
	if err != nil {
		return err
	}
	var docsLinksBuilder strings.Builder
	for _, l := range links {
		fmt.Fprintf(&docsLinksBuilder, "- [**%s**](./%s)\n", l.Title, l.Slug)
	}

	readme, err := readReadmeBody(p.readme, 3)
	if err != nil {
		return err
	}
	var readmeContent string
	if strings.TrimSpace(readme) != "" {
		readmeContent = "## Readme\n\n" + readme
	}

	tmplBytes, err := os.ReadFile(opts.TemplatePath)
	if err != nil {
		return fmt.Errorf("read template: %w", err)
	}
	rendered := string(tmplBytes)
	licenseBadge := strings.ReplaceAll(vars["LICENSE"], "-", "--")
	rendered = strings.ReplaceAll(rendered, "{{ APP }}", opts.App)
	rendered = strings.ReplaceAll(rendered, "{{ VERSION }}", vars["VERSION"])
	rendered = strings.ReplaceAll(rendered, "{{ LICENSE }}", licenseBadge)
	rendered = strings.ReplaceAll(rendered, "{{ SOURCE }}", vars["SOURCE"])
	rendered = strings.ReplaceAll(rendered, "{{ DOCS_LINKS }}", strings.TrimRight(docsLinksBuilder.String(), "\n"))
	rendered = strings.ReplaceAll(rendered, "{{ README_CONTENT }}", readmeContent)
	// Backwards compatibility: legacy templates may still contain the
	// {{ COMPOSE_FILE }} placeholder; the snippet now lives on its own page,
	// so simply strip the placeholder.
	rendered = strings.ReplaceAll(rendered, "{{ COMPOSE_FILE }}", "")

	if err := os.MkdirAll(filepath.Dir(p.indexFile), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p.indexFile, []byte(rendered), 0o644)
}

// writeComposePage renders the docker-compose snippet for the app and writes
// it to a standalone docker-compose.md page inside the per-app docs directory.
// When settings.yaml or the compose template are absent the page is not
// created (and any pre-existing one is removed for cleanliness).
func writeComposePage(opts ContainerOptions, p containerPaths, vars map[string]string) error {
	pagePath := filepath.Join(p.docsDir, "docker-compose.md")

	settings, ok, err := parseSettings(p.settings)
	if err != nil {
		return fmt.Errorf("parse settings.yaml: %w", err)
	}
	if !ok {
		_ = os.Remove(pagePath)
		return nil
	}

	snippet, err := renderComposeSnippet(opts.App, vars["VERSION"], settings, opts.ComposeTemplatePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Info().Msgf("compose template not found for %s, skipping", opts.App)
			_ = os.Remove(pagePath)
			return nil
		}
		return err
	}

	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("title: Docker Compose\n")
	b.WriteString("---\n\n")
	fmt.Fprintf(&b, "Example `docker-compose.yaml` for **%s**:\n\n", opts.App)
	b.WriteString("```yaml\n")
	b.WriteString(snippet)
	if !strings.HasSuffix(snippet, "\n") {
		b.WriteString("\n")
	}
	b.WriteString("```\n")

	if err := os.MkdirAll(p.docsDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(pagePath, []byte(b.String()), 0o644)
}

// parseSettings reads an app's settings.yaml file. It returns (settings, true,
// nil) on success, (zero, false, nil) if the file does not exist, or an error
// on any other failure.
func parseSettings(path string) (AppSettings, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return AppSettings{}, false, nil
		}
		return AppSettings{}, false, err
	}
	var s AppSettings
	if err := yaml.Unmarshal(data, &s); err != nil {
		return AppSettings{}, false, fmt.Errorf("unmarshal %s: %w", path, err)
	}
	return s, true, nil
}

// renderComposeSnippet generates a docker-compose YAML string from the
// docker-compose.yaml.tmpl template and the app's settings. The template uses
// ${TOKEN} scalar substitutions and # BEGIN_X / # END_X comment blocks to
// delimit repeated sections.
func renderComposeSnippet(app, version string, settings AppSettings, tmplPath string) (string, error) {
	tmplBytes, err := os.ReadFile(tmplPath)
	if err != nil {
		return "", err
	}

	// Strip leading header comment lines (before the first non-comment line)
	// and all BEGIN_*/END_* comment blocks.
	lines := strings.Split(string(tmplBytes), "\n")
	var kept []string
	inBlock := false
	headerDone := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# BEGIN_") {
			inBlock = true
			continue
		}
		if strings.HasPrefix(trimmed, "# END_") {
			inBlock = false
			continue
		}
		if inBlock {
			continue
		}
		// Skip leading top-level comment lines before services:
		if !headerDone {
			if strings.HasPrefix(trimmed, "#") || trimmed == "" {
				continue
			}
			headerDone = true
		}
		kept = append(kept, line)
	}
	result := strings.Join(kept, "\n")

	// Scalar token substitutions.
	// Use the version from the bake file when available; fall back to "rolling"
	// so the image reference is always usable. Never emit a bare image name
	// (no tag) or the mutable "latest" tag.
	tag := version
	if tag == "" || strings.EqualFold(tag, "latest") {
		tag = "rolling"
	}
	image := "ghcr.io/trueforge-org/" + app + ":" + tag
	result = strings.ReplaceAll(result, "${SERVICE_NAME}", app)
	result = strings.ReplaceAll(result, "${IMAGE}", image)
	result = strings.ReplaceAll(result, "${CONTAINER_NAME}", app)
	result = strings.ReplaceAll(result, "${RESTART_POLICY:-unless-stopped}", "unless-stopped")

	// Replace ports placeholder.
	result = replacePortsSection(result, settings.Ports)

	// Replace environment placeholder.
	result = replaceEnvSection(result, settings.Env)

	// Replace volumes placeholder.
	result = replaceVolumesSection(result, settings.Volumes)

	return strings.TrimSpace(result) + "\n", nil
}

// replacePortsSection rewrites the static "ports:\n      []" placeholder with
// actual port mappings derived from settings, or leaves it as "ports: []" when
// there are no ports.
func replacePortsSection(result string, ports []PortSetting) string {
	if len(ports) == 0 {
		return strings.ReplaceAll(result, "    ports:\n      []", "    ports: []")
	}
	var b strings.Builder
	b.WriteString("    ports:\n")
	for _, p := range ports {
		if strings.EqualFold(p.Protocol, "tcp") {
			fmt.Fprintf(&b, "      - \"%d:%d\"\n", p.Port, p.Port)
		} else {
			fmt.Fprintf(&b, "      - \"%d:%d/%s\"\n", p.Port, p.Port, strings.ToLower(p.Protocol))
		}
	}
	return strings.ReplaceAll(result, "    ports:\n      []", strings.TrimRight(b.String(), "\n"))
}

// replaceEnvSection rewrites the static "environment: {}" placeholder with
// actual environment variables derived from settings.
func replaceEnvSection(result string, env []EnvSetting) string {
	if len(env) == 0 {
		return result
	}
	var b strings.Builder
	b.WriteString("    environment:\n")
	for _, e := range env {
		fmt.Fprintf(&b, "      %s: %q\n", e.Name, e.Default)
	}
	return strings.ReplaceAll(result, "    environment: {}", strings.TrimRight(b.String(), "\n"))
}

// replaceVolumesSection rewrites the default "./config:/config:rw" placeholder
// with actual volume mount lines derived from settings.
func replaceVolumesSection(result string, volumes []VolumeSetting) string {
	if len(volumes) == 0 {
		return result
	}
	var b strings.Builder
	b.WriteString("    volumes:\n")
	for _, v := range volumes {
		hostName := filepath.Base(v.Path)
		fmt.Fprintf(&b, "      - ./%s:%s\n", hostName, v.Path)
	}
	return strings.ReplaceAll(result, "    volumes:\n      - ./config:/config:rw", strings.TrimRight(b.String(), "\n"))
}

// DiscoverApps lists the apps under appsDir that contain a docker-bake.hcl
// file. It is a convenience used by the cobra command when the user wants to
// process every app at once.
func DiscoverApps(appsDir string) ([]string, error) {
	entries, err := os.ReadDir(appsDir)
	if err != nil {
		return nil, err
	}
	var apps []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		bake := filepath.Join(appsDir, e.Name(), "docker-bake.hcl")
		if _, err := os.Stat(bake); err == nil {
			apps = append(apps, e.Name())
		}
	}
	return apps, nil
}

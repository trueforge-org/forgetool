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

	"github.com/trueforge-org/forgetool/pkg/containers"
)

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
	// ComposeTemplatePath is the markdown page template used to render the
	// per-app docker-compose.md page. Defaults to
	// "templates/docker-compose.md.tmpl". The template supports the
	// placeholders "{{ APP }}" and "{{COMPOSE}}"; the latter is replaced with
	// a fenced YAML code block containing the generated compose file. When the
	// template file is missing no docker-compose.md page is generated.
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
		o.ComposeTemplatePath = "templates/docker-compose.md.tmpl"
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
// The compose YAML is generated from the AppSettings using Go structs and
// embedded as a fenced code block under the "{{COMPOSE}}" placeholder of the
// page template (templates/docker-compose.md.tmpl by default). When
// settings.yaml or the page template are absent the page is not created (and
// any pre-existing one is removed for cleanliness).
func writeComposePage(opts ContainerOptions, p containerPaths, vars map[string]string) error {
	pagePath := filepath.Join(p.docsDir, "docker-compose.md")

	settings, ok, err := containers.ParseSettings(p.settings)
	if err != nil {
		return fmt.Errorf("parse settings.yaml: %w", err)
	}
	if !ok {
		_ = os.Remove(pagePath)
		return nil
	}

	tmplBytes, err := os.ReadFile(opts.ComposeTemplatePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Info().Msgf("compose page template not found for %s, skipping", opts.App)
			_ = os.Remove(pagePath)
			return nil
		}
		return err
	}

	composeYAML, err := containers.BuildComposeYAML(opts.App, vars["VERSION"], settings, dependencyResolver(opts.AppsDir))
	if err != nil {
		return err
	}

	var fenced strings.Builder
	fenced.WriteString("```yaml\n")
	fenced.WriteString(composeYAML)
	fenced.WriteString("```")

	rendered := string(tmplBytes)
	rendered = strings.ReplaceAll(rendered, "{{ APP }}", opts.App)
	rendered = strings.ReplaceAll(rendered, "{{APP}}", opts.App)
	rendered = strings.ReplaceAll(rendered, "{{COMPOSE}}", fenced.String())
	rendered = strings.ReplaceAll(rendered, "{{ COMPOSE }}", fenced.String())

	if err := os.MkdirAll(p.docsDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(pagePath, []byte(rendered), 0o644)
}

// dependencyResolver returns a containers.DependencyResolver that looks up
// dependency apps under appsDir. For each requested image it loads the app's
// settings.yaml and parses its docker-bake.hcl for the VERSION variable. A
// dependency without both files is reported as not found and silently
// skipped by BuildComposeYAML.
func dependencyResolver(appsDir string) containers.DependencyResolver {
	return func(image string) (containers.AppSettings, string, bool, error) {
		appDir := filepath.Join(appsDir, image)
		settingsPath := filepath.Join(appDir, "settings.yaml")
		bakePath := filepath.Join(appDir, "docker-bake.hcl")

		settings, ok, err := containers.ParseSettings(settingsPath)
		if err != nil {
			return containers.AppSettings{}, "", false, err
		}
		if !ok {
			return containers.AppSettings{}, "", false, nil
		}
		if _, err := os.Stat(bakePath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return settings, "", true, nil
			}
			return containers.AppSettings{}, "", false, err
		}
		vars, err := parseBakeVars(bakePath)
		if err != nil {
			return containers.AppSettings{}, "", false, err
		}
		return settings, vars["VERSION"], true, nil
	}
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

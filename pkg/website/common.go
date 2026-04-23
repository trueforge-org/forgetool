// Package website builds website documentation pages for charts and
// container apps. It is a Go replacement for the legacy
// .github/scripts/chart-docs.sh (truecharts) and
// .github/scripts/container-docs.sh (containerforge) shell scripts and is
// designed to produce equivalent on-disk output so it can drop into the
// existing release workflows.
package website

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

// safeDocs is the list of files that must be preserved across a docs
// regeneration cycle (they are kept aside, the docs directory is wiped, then
// they are restored).
var safeDocs = []string{"CHANGELOG.md"}

// httpClient is package-level so tests can swap it. Defaults to a sane client.
var httpClient = &http.Client{Timeout: 30 * time.Second}

// keepDocsSafe moves any safeDocs files for itemPath out of docsDir into
// tmpDir/itemPath so they can be restored after the docs directory is wiped.
func keepDocsSafe(docsDir, tmpDir, itemPath string) error {
	if err := os.MkdirAll(filepath.Join(tmpDir, itemPath), 0o755); err != nil {
		return fmt.Errorf("create safe-docs tmp dir: %w", err)
	}
	for _, doc := range safeDocs {
		src := filepath.Join(docsDir, itemPath, doc)
		if _, err := os.Stat(src); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return err
		}
		dst := filepath.Join(tmpDir, itemPath, doc)
		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("save %s: %w", doc, err)
		}
	}
	return nil
}

// restoreSafeDocs is the inverse of keepDocsSafe.
func restoreSafeDocs(docsDir, tmpDir, itemPath string) error {
	for _, doc := range safeDocs {
		src := filepath.Join(tmpDir, itemPath, doc)
		if _, err := os.Stat(src); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return err
		}
		dst := filepath.Join(docsDir, itemPath, doc)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("restore %s: %w", doc, err)
		}
	}
	return nil
}

// resetDir removes dir (if present) and recreates an empty directory at the
// same path.
func resetDir(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove %s: %w", dir, err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("recreate %s: %w", dir, err)
	}
	return nil
}

// copyTreeContents copies every entry inside src into dst. Missing src is not
// an error (parity with `cp -rf src/* dst/ 2>/dev/null || :`).
func copyTreeContents(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	for _, e := range entries {
		s := filepath.Join(src, e.Name())
		d := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if err := helper.CopyDir(s, d, true); err != nil {
				return err
			}
			continue
		}
		if err := helper.CopyFile(s, d, true); err != nil {
			return err
		}
	}
	return nil
}

// copyFileIfExists copies src to dst when src exists; missing src is not an
// error.
func copyFileIfExists(src, dst string) error {
	if _, err := os.Stat(src); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return helper.CopyFile(src, dst, true)
}

// downloadFile fetches url and writes the response body to dst. Non-200
// responses are reported as errors so the caller can decide whether to ignore
// them (mirroring the `|| echo` fallback in the shell scripts).
func downloadFile(url, dst string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http %d for %s", resp.StatusCode, url)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

// ensureFrontMatter ensures file has a YAML front-matter title. If the file
// already begins with a `---` front-matter block it is left alone. If it
// instead starts with an old-style markdown `# Title` line, that line is
// promoted into a `title: ...` front matter block. Returns false (without
// modification) when neither is present.
func ensureFrontMatter(file string) (bool, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return false, err
	}
	content := string(data)
	if strings.HasPrefix(strings.TrimLeft(content, " \t"), "---") {
		return true, nil
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var (
		title    string
		foundIdx = -1
		lines    []string
	)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}
	for i, l := range lines {
		if strings.HasPrefix(l, "# ") {
			title = strings.TrimSpace(strings.TrimPrefix(l, "# "))
			foundIdx = i
			break
		}
	}
	if foundIdx == -1 || title == "" {
		return false, nil
	}
	// Drop the original `# Title` line.
	lines = append(lines[:foundIdx], lines[foundIdx+1:]...)
	rebuilt := fmt.Sprintf("---\ntitle: %s\n---\n%s", title, strings.Join(lines, "\n"))
	if !strings.HasSuffix(rebuilt, "\n") {
		rebuilt += "\n"
	}
	if err := os.WriteFile(file, []byte(rebuilt), 0o644); err != nil {
		return false, err
	}
	return true, nil
}

// extractFrontMatterTitle returns the value of the `title:` key inside the
// leading YAML front matter block of file, if any.
func extractFrontMatterTitle(file string) (string, error) {
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	inBlock := false
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if !inBlock {
			if trimmed == "---" {
				inBlock = true
				continue
			}
			// No front matter found before the first non-empty line.
			if trimmed != "" {
				return "", nil
			}
			continue
		}
		if trimmed == "---" {
			break
		}
		if strings.HasPrefix(trimmed, "title:") {
			val := strings.TrimSpace(strings.TrimPrefix(trimmed, "title:"))
			val = strings.Trim(val, `"'`)
			return val, nil
		}
	}
	return "", scanner.Err()
}

// docsLink represents a single sibling-doc entry on the index page.
type docsLink struct {
	Title string
	Slug  string
}

// collectDocsLinks scans dir for *.md / *.mdx files (excluding index.md*),
// ensures each has a usable title, and returns them sorted by filename to
// produce stable output across runs.
func collectDocsLinks(dir string) ([]docsLink, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	type candidate struct {
		path string
		name string
	}
	var candidates []candidate
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".md" && ext != ".mdx" {
			continue
		}
		if name == "index.md" || name == "index.mdx" {
			continue
		}
		candidates = append(candidates, candidate{path: filepath.Join(dir, name), name: name})
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].name < candidates[j].name })

	var links []docsLink
	for _, c := range candidates {
		ok, err := ensureFrontMatter(c.path)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		title, err := extractFrontMatterTitle(c.path)
		if err != nil {
			return nil, err
		}
		if title == "" {
			continue
		}
		base := strings.TrimSuffix(c.name, filepath.Ext(c.name))
		links = append(links, docsLink{Title: title, Slug: strings.ToLower(base)})
	}
	return links, nil
}

// readReadmeBody returns the README body to embed in an index page. The first
// `skipLines` lines are dropped (matching `tail -n +N`) and `## ` headings are
// demoted to `### ` (matching `sed s/##/###/`). Missing files yield an empty
// string with no error.
func readReadmeBody(path string, skipLines int) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	lines := strings.Split(string(data), "\n")
	if skipLines > len(lines) {
		skipLines = len(lines)
	}
	trimmed := lines[skipLines:]
	for i, l := range trimmed {
		if strings.Contains(l, "##") {
			trimmed[i] = strings.Replace(l, "##", "###", 1)
		}
	}
	return strings.Join(trimmed, "\n"), nil
}

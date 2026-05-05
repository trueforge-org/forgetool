package website

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/trueforge-org/forgetool/v4/pkg/helper"
)

type fakeDirEntry struct{ name string }

func (f fakeDirEntry) Name() string               { return f.name }
func (f fakeDirEntry) IsDir() bool                { return false }
func (f fakeDirEntry) Type() os.FileMode          { return 0 }
func (f fakeDirEntry) Info() (os.FileInfo, error) { return nil, nil }

type fakeDirEntryDir struct{ name string }

func (f fakeDirEntryDir) Name() string               { return f.name }
func (f fakeDirEntryDir) IsDir() bool                { return true }
func (f fakeDirEntryDir) Type() os.FileMode          { return os.ModeDir }
func (f fakeDirEntryDir) Info() (os.FileInfo, error) { return nil, nil }

func writeBakeHCL(t *testing.T, dir, app, version, license, source string) string {
	t.Helper()
	appDir := filepath.Join(dir, app)
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	bakePath := filepath.Join(appDir, "docker-bake.hcl")
	content := "variable \"APP\" {\n  default = \"" + app + "\"\n}\n" +
		"variable \"VERSION\" {\n  default = \"" + version + "\"\n}\n" +
		"variable \"LICENSE\" {\n  default = \"" + license + "\"\n}\n" +
		"variable \"SOURCE\" {\n  default = \"" + source + "\"\n}\n"
	if err := os.WriteFile(bakePath, []byte(content), 0644); err != nil {
		t.Fatalf("write bake: %v", err)
	}
	return bakePath
}

func readBakeEntry(t *testing.T, dir string) os.DirEntry {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		if e.Name() == "docker-bake.hcl" {
			return e
		}
	}
	t.Fatal("docker-bake.hcl not found")
	return nil
}

func TestGetContainerData_AddsContainer(t *testing.T) {
	td := t.TempDir()
	bakePath := writeBakeHCL(t, td, "myapp", "2.0.0", "Apache-2.0", "https://github.com/example/myapp")
	entry := readBakeEntry(t, filepath.Dir(bakePath))

	opts := &ContainerListOptions{}
	if err := opts.GetContainerData(bakePath, entry, nil); err != nil && err != filepath.SkipDir {
		t.Fatalf("GetContainerData failed: %v", err)
	}

	if opts.list == nil {
		t.Fatal("list nil")
	}
	if opts.list.TotalCount != 1 {
		t.Fatalf("expected totalcount 1 got %d", opts.list.TotalCount)
	}
	if len(opts.list.Containers) != 1 {
		t.Fatalf("expected 1 container got %d", len(opts.list.Containers))
	}
	c := opts.list.Containers[0]
	if c.Name != "myapp" {
		t.Fatalf("expected name myapp got %s", c.Name)
	}
	if c.Version != "2.0.0" {
		t.Fatalf("expected version 2.0.0 got %s", c.Version)
	}
	if c.License != "Apache-2.0" {
		t.Fatalf("expected license Apache-2.0 got %s", c.License)
	}
	if c.Source != "https://github.com/example/myapp" {
		t.Fatalf("expected source got %s", c.Source)
	}
	if c.Icon != defaultIcon {
		t.Fatalf("expected default icon, got %s", c.Icon)
	}
}

func TestGetContainerData_ReturnsInputError(t *testing.T) {
	opts := &ContainerListOptions{}
	wantErr := errors.New("walker error")
	if err := opts.GetContainerData("whatever", fakeDirEntry{name: "docker-bake.hcl"}, wantErr); !errors.Is(err, wantErr) {
		t.Fatalf("expected input error to be returned, got: %v", err)
	}
}

func TestGetContainerData_IgnoresNonBakeFile(t *testing.T) {
	td := t.TempDir()
	otherPath := filepath.Join(td, "Dockerfile")
	if err := os.WriteFile(otherPath, []byte("FROM ubuntu\n"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	entries, err := os.ReadDir(td)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}

	opts := &ContainerListOptions{}
	if err := opts.GetContainerData(otherPath, entries[0], nil); err != nil {
		t.Fatalf("GetContainerData should ignore non-bake files: %v", err)
	}
	if opts.list == nil {
		t.Fatal("expected list to be initialized")
	}
	if opts.list.TotalCount != 0 {
		t.Fatalf("expected total count unchanged for non-bake file")
	}
}

func TestGetContainerData_SkipsExcludedDirs(t *testing.T) {
	for _, dir := range helper.ExcludedDirs {
		opts := &ContainerListOptions{}
		err := opts.GetContainerData("/some/path", fakeDirEntryDir{name: dir}, nil)
		if err != filepath.SkipDir {
			t.Fatalf("expected SkipDir for excluded directory %q, got: %v", dir, err)
		}
	}
}

func TestGetContainerData_AppendsMultiple(t *testing.T) {
	td := t.TempDir()
	pathA := writeBakeHCL(t, td, "appa", "1.0.0", "MIT", "https://example.com/appa")
	pathB := writeBakeHCL(t, td, "appb", "1.0.0", "MIT", "https://example.com/appb")
	entryA := readBakeEntry(t, filepath.Dir(pathA))
	entryB := readBakeEntry(t, filepath.Dir(pathB))

	opts := &ContainerListOptions{}
	for _, pair := range []struct {
		path  string
		entry os.DirEntry
	}{{pathA, entryA}, {pathB, entryB}} {
		if err := opts.GetContainerData(pair.path, pair.entry, nil); err != nil && err != filepath.SkipDir {
			t.Fatalf("GetContainerData failed: %v", err)
		}
	}

	if opts.list.TotalCount != 2 {
		t.Fatalf("expected total count 2, got %d", opts.list.TotalCount)
	}
	if len(opts.list.Containers) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(opts.list.Containers))
	}
}

func TestParseBakeVariables(t *testing.T) {
	td := t.TempDir()
	bakePath := filepath.Join(td, "docker-bake.hcl")
	hcl := "target \"docker-metadata-action\" {}\n\n" +
		"variable \"APP\" {\n  default = \"testapp\"\n}\n\n" +
		"variable \"VERSION\" {\n  // renovate comment\n  default = \"3.2.1\"\n}\n\n" +
		"variable \"LICENSE\" {\n  default = \"GPL-3.0\"\n}\n\n" +
		"variable \"SOURCE\" {\n  default = \"https://github.com/test/app\"\n}\n"
	if err := os.WriteFile(bakePath, []byte(hcl), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	vars, err := parseBakeVariables(bakePath)
	if err != nil {
		t.Fatalf("parseBakeVariables failed: %v", err)
	}

	want := map[string]string{
		"APP":     "testapp",
		"VERSION": "3.2.1",
		"LICENSE": "GPL-3.0",
		"SOURCE":  "https://github.com/test/app",
	}
	for k, v := range want {
		if vars[k] != v {
			t.Errorf("expected %s=%q, got %q", k, v, vars[k])
		}
	}
}

func TestParseBakeVariables_FileNotFound(t *testing.T) {
	_, err := parseBakeVariables("/nonexistent/docker-bake.hcl")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestBuildContainer_DefaultIcon(t *testing.T) {
	c := buildContainer(map[string]string{
		"APP": "foo", "VERSION": "1.0", "SOURCE": "https://example.com", "LICENSE": "MIT",
	})
	if c.Icon != defaultIcon {
		t.Fatalf("expected default icon, got %s", c.Icon)
	}
}

func TestBuildContainer_CustomIcon(t *testing.T) {
	c := buildContainer(map[string]string{
		"APP": "foo", "VERSION": "1.0", "SOURCE": "https://example.com",
		"LICENSE": "MIT", "ICON": "https://custom.icon/img.png",
	})
	if c.Icon != "https://custom.icon/img.png" {
		t.Fatalf("expected custom icon, got %s", c.Icon)
	}
}

func TestValidateEntry_ErrorPassthrough(t *testing.T) {
	wantErr := errors.New("some error")
	if err := validateEntry(fakeDirEntry{name: "x"}, wantErr); !errors.Is(err, wantErr) {
		t.Fatalf("expected error passthrough, got: %v", err)
	}
}

func TestValidateEntry_NormalFile(t *testing.T) {
	if err := validateEntry(fakeDirEntry{name: "docker-bake.hcl"}, nil); err != nil {
		t.Fatalf("expected no error for normal file, got: %v", err)
	}
}

func TestWriteContainerList_NilList(t *testing.T) {
	opts := &ContainerListOptions{OutputPath: filepath.Join(t.TempDir(), "out.json")}
	if err := opts.WriteContainerList(); err == nil {
		t.Fatal("expected error when list is nil")
	}
}

func TestWriteContainerList_WritesFile(t *testing.T) {
	td := t.TempDir()
	out := filepath.Join(td, "containers.json")
	opts := &ContainerListOptions{OutputPath: out}
	opts.list = &ContainerList{
		TotalCount: 1,
		Containers: []Container{{Name: "c", Version: "1.0", Source: "https://x", License: "MIT", Icon: defaultIcon}},
	}

	if err := opts.WriteContainerList(); err != nil {
		t.Fatalf("WriteContainerList failed: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("expected output file: %v", err)
	}
	var cl ContainerList
	if err := json.Unmarshal(data, &cl); err != nil {
		t.Fatalf("invalid json written: %v", err)
	}
	if cl.TotalCount != 1 || len(cl.Containers) != 1 || cl.Containers[0].Name != "c" {
		t.Fatalf("unexpected content: %+v", cl)
	}
}

func TestWriteContainerList_WriteError(t *testing.T) {
	td := t.TempDir()
	opts := &ContainerListOptions{OutputPath: td}
	opts.list = &ContainerList{TotalCount: 1}
	if err := opts.WriteContainerList(); err == nil {
		t.Fatal("expected write error when output path is a directory")
	}
}

func TestWriteContainerList_MarshalError(t *testing.T) {
	orig := marshalContainerList
	marshalContainerList = func(_ interface{}) ([]byte, error) {
		return nil, errors.New("marshal fail")
	}
	t.Cleanup(func() { marshalContainerList = orig })

	opts := &ContainerListOptions{OutputPath: filepath.Join(t.TempDir(), "out.json")}
	opts.list = &ContainerList{TotalCount: 1}
	if err := opts.WriteContainerList(); err == nil {
		t.Fatal("expected marshal error")
	}
}

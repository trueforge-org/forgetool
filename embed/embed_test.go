package embed

import (
	"embed"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

var defaultRemoveAll = removeAll
var defaultMkdirAll = mkdirAll
var defaultWriteFile = writeFile
var defaultWalkDir = walkDir
var defaultFromEmbeddedFS = fromEmbeddedFS
var defaultFatalErr = fatalErr

func resetEmbedHooks() {
	removeAll = defaultRemoveAll
	mkdirAll = defaultMkdirAll
	writeFile = defaultWriteFile
	walkDir = defaultWalkDir
	fromEmbeddedFS = defaultFromEmbeddedFS
	fatalErr = defaultFatalErr
	resolveClusterTemplateVersionHook = func() (string, error) { return "", errors.New("test stub") }
	downloadClusterTemplateReleaseHook = func(string) error { return errors.New("test stub") }
}

type readFileStubFS struct {
	fs.FS
	readFile func(name string) ([]byte, error)
}

func (r readFileStubFS) ReadFile(name string) ([]byte, error) {
	return r.readFile(name)
}

type dirEntryStub struct {
	name string
	dir  bool
}

func (d dirEntryStub) Name() string               { return d.name }
func (d dirEntryStub) IsDir() bool                { return d.dir }
func (d dirEntryStub) Type() fs.FileMode          { return fs.ModeDir }
func (d dirEntryStub) Info() (fs.FileInfo, error) { return nil, nil }

func TestFilesToCache_WritesGenericFiles(t *testing.T) {
	resetEmbedHooks()
	t.Cleanup(resetEmbedHooks)

	oldCache := helper.CacheDir
	helper.CacheDir = t.TempDir()
	t.Cleanup(func() {
		helper.CacheDir = oldCache
	})

	filesToCache(GenericFiles, "generic")

	foundFile := false
	err := filepath.WalkDir(helper.CacheDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			foundFile = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking cache failed: %v", err)
	}
	if !foundFile {
		t.Fatalf("expected generic embedded files to be written to cache")
	}

	entries, err := os.ReadDir(helper.CacheDir)
	if err != nil {
		t.Fatalf("reading cache dir failed: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("expected cache directory to contain entries")
	}
}

func TestFilesToCache_MkdirAllErrorReturnsEarly(t *testing.T) {
	resetEmbedHooks()
	t.Cleanup(resetEmbedHooks)

	oldCache := helper.CacheDir
	helper.CacheDir = t.TempDir()
	t.Cleanup(func() { helper.CacheDir = oldCache })

	calledWalk := false
	mkdirAll = func(path string, perm os.FileMode) error {
		if path == helper.CacheDir {
			return errors.New("mkdir failed")
		}
		return os.MkdirAll(path, perm)
	}
	walkDir = func(fsys fs.FS, root string, fn fs.WalkDirFunc) error {
		calledWalk = true
		return nil
	}

	filesToCache(GenericFiles, "generic")

	if calledWalk {
		t.Fatal("expected filesToCache to return before walk when cache mkdir fails")
	}
}

func TestFilesToCache_WalkDirErrorPath(t *testing.T) {
	resetEmbedHooks()
	t.Cleanup(resetEmbedHooks)

	oldCache := helper.CacheDir
	helper.CacheDir = t.TempDir()
	t.Cleanup(func() { helper.CacheDir = oldCache })

	walkDir = func(fsys fs.FS, root string, fn fs.WalkDirFunc) error {
		return fn("x", nil, errors.New("walk failed"))
	}

	filesToCache(GenericFiles, "generic")
}

func TestFilesToCache_CreateDirectoryErrorPath(t *testing.T) {
	resetEmbedHooks()
	t.Cleanup(resetEmbedHooks)

	oldCache := helper.CacheDir
	helper.CacheDir = t.TempDir()
	t.Cleanup(func() { helper.CacheDir = oldCache })

	dirPath := filepath.Join(helper.CacheDir, "existing-file")
	if err := os.WriteFile(dirPath, []byte("x"), 0600); err != nil {
		t.Fatalf("failed to seed existing file for mkdir failure: %v", err)
	}

	walkDir = func(fsys fs.FS, root string, fn fs.WalkDirFunc) error {
		return fn("existing-file", dirEntryStub{name: "existing-file", dir: true}, nil)
	}

	filesToCache(GenericFiles, "generic")
}

func TestFilesToCache_ReadFileErrorPath(t *testing.T) {
	resetEmbedHooks()
	t.Cleanup(resetEmbedHooks)

	oldCache := helper.CacheDir
	helper.CacheDir = t.TempDir()
	t.Cleanup(func() { helper.CacheDir = oldCache })

	fromEmbeddedFS = func(embeddedFS embed.FS, sub string) (readDirFS, error) {
		base := fstest.MapFS{"f.txt": &fstest.MapFile{Data: []byte("x")}}
		return readFileStubFS{
			FS: base,
			readFile: func(name string) ([]byte, error) {
				return nil, errors.New("read failed")
			},
		}, nil
	}

	filesToCache(GenericFiles, "generic")
}

func TestFilesToCache_WriteFileErrorPath(t *testing.T) {
	resetEmbedHooks()
	t.Cleanup(resetEmbedHooks)

	oldCache := helper.CacheDir
	helper.CacheDir = t.TempDir()
	t.Cleanup(func() { helper.CacheDir = oldCache })

	fromEmbeddedFS = func(embeddedFS embed.FS, sub string) (readDirFS, error) {
		base := fstest.MapFS{"f.txt": &fstest.MapFile{Data: []byte("x")}}
		return readFileStubFS{
			FS:       base,
			readFile: base.ReadFile,
		}, nil
	}

	writeFile = func(name string, data []byte, perm os.FileMode) error {
		return errors.New("write failed")
	}

	filesToCache(GenericFiles, "generic")
}

func TestAllToCache_Idempotent(t *testing.T) {
	resetEmbedHooks()
	t.Cleanup(resetEmbedHooks)

	oldCache := helper.CacheDir
	helper.CacheDir = t.TempDir()
	t.Cleanup(func() { helper.CacheDir = oldCache })

	AllToCache()
	entries1, err := os.ReadDir(helper.CacheDir)
	if err != nil {
		t.Fatalf("first ReadDir failed: %v", err)
	}

	AllToCache()
	entries2, err := os.ReadDir(helper.CacheDir)
	if err != nil {
		t.Fatalf("second ReadDir failed: %v", err)
	}

	if len(entries1) != len(entries2) {
		t.Fatalf("expected same number of entries after second call, got %d vs %d", len(entries1), len(entries2))
	}
}

func TestAllToCache_RemoveAllErrorCallsFatal(t *testing.T) {
	resetEmbedHooks()
	t.Cleanup(resetEmbedHooks)

	oldCache := helper.CacheDir
	helper.CacheDir = t.TempDir()
	t.Cleanup(func() { helper.CacheDir = oldCache })

	var capturedErr error
	removeAll = func(path string) error {
		return errors.New("remove all failed")
	}
	fatalErr = func(err error) {
		capturedErr = err
	}

	AllToCache()

	if capturedErr == nil {
		t.Fatal("expected AllToCache to call fatalErr when removeAll fails")
	}
}

func TestDefaultFatalErr_NoPanic(t *testing.T) {
	resetEmbedHooks()
	t.Cleanup(resetEmbedHooks)

	fatalErr(errors.New("boom"))
}

func TestAllToCache_SkipsExistingFiles(t *testing.T) {
	resetEmbedHooks()
	t.Cleanup(resetEmbedHooks)

	oldCache := helper.CacheDir
	helper.CacheDir = t.TempDir()
	t.Cleanup(func() { helper.CacheDir = oldCache })

	AllToCache()

	sizes := map[string]int64{}
	err := filepath.Walk(helper.CacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			sizes[path] = info.Size()
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk failed: %v", err)
	}

	AllToCache()

	for path, size := range sizes {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("file %s missing after second AllToCache: %v", path, err)
		}
		if info.Size() != size {
			t.Fatalf("file %s size changed: %d -> %d", path, size, info.Size())
		}
	}
}

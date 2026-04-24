package cmd

import (
	"io/fs"
	"testing"
)

// Covers the `return fmt.Errorf("error walking directory ...")` branch in
// defaultContainersGenListWalk by passing a non-existent path and a walkFunc
// that propagates the lstat error.
func TestDefaultContainersGenListWalk_WalkDirError(t *testing.T) {
	walkFn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return nil
	}
	err := defaultContainersGenListWalk([]string{"/definitely/does/not/exist/forgetool/abc"}, walkFn)
	if err == nil {
		t.Fatalf("expected walk error for missing path")
	}
}

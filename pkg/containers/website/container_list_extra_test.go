package website

import (
	"path/filepath"
	"testing"
)

// Cover the parseBakeVariables error branch returned by GetContainerData
// (lines 68-70 in container_list.go) by handing in a fakeDirEntry whose
// Name() reports the bake filename but whose path points at a non-existent
// file so os.Open fails.
func TestGetContainerData_ParseBakeVariablesError(t *testing.T) {
	o := &ContainerListOptions{}
	td := t.TempDir()
	missing := filepath.Join(td, "does-not-exist", "docker-bake.hcl")
	if err := o.GetContainerData(missing, fakeDirEntry{name: "docker-bake.hcl"}, nil); err == nil {
		t.Fatalf("expected error from parseBakeVariables on missing file")
	}
}

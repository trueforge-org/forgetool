package talassist

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

func TestGenSchemaAndNewSecretBundle(t *testing.T) {
	oldClusterPath := helper.ClusterPath
	helper.ClusterPath = t.TempDir()
	t.Cleanup(func() {
		helper.ClusterPath = oldClusterPath
	})

	if err := GenSchema(); err != nil {
		t.Fatalf("GenSchema failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(helper.ClusterPath, "talos", "talconfig.json")); err != nil {
		t.Fatalf("expected generated schema file: %v", err)
	}

	LatestTalosVersion = "v1.7.0"
	bundle := NewSecretBundle()
	if bundle == nil {
		t.Fatalf("expected non-nil secret bundle")
	}
}

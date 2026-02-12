package gencmd

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

func TestManifestPathsShape(t *testing.T) {
	if len(manifestPaths) != 3 {
		t.Fatalf("expected 3 manifest paths, got %d", len(manifestPaths))
	}
	expected := []string{"sopssecret.secret.yaml", "deploykey.secret.yaml", "clustersettings.secret.yaml"}
	for i, p := range manifestPaths {
		if !strings.HasSuffix(p, expected[i]) {
			t.Fatalf("unexpected manifest path at %d: %s", i, p)
		}
	}
	if !strings.Contains(manifestPaths[0], filepath.Join("flux-system", "flux")) {
		t.Fatalf("expected flux-system/flux segment in first manifest path")
	}
	_ = helper.KubernetesPath
}

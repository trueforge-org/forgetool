package gencmd

import (
	"path/filepath"
	"strings"
	"testing"

	talhelperCfg "github.com/budimanjojo/talhelper/v3/pkg/config"
	"github.com/trueforge-org/forgetool/pkg/helper"
	"github.com/trueforge-org/forgetool/pkg/talassist"
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

func TestRunBootstrapPanicsWithoutNodes(t *testing.T) {
	oldTalConfig := talassist.TalConfig
	t.Cleanup(func() {
		talassist.TalConfig = oldTalConfig
	})

	talassist.TalConfig = &talhelperCfg.TalhelperConfig{}

	defer func() {
		if recover() == nil {
			t.Fatalf("expected panic when TalConfig has no nodes")
		}
	}()

	RunBootstrap([]string{"bootstrap", "--dry"})
}

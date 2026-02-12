package gencmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/helper"
	"github.com/trueforge-org/forgetool/pkg/talassist"
)

func setupGenTalSecretTest(t *testing.T, generatedDir string) {
	t.Helper()
	oldTalos := helper.TalosGenerated
	oldTalSecret := helper.TalSecretFile
	oldVersion := talassist.LatestTalosVersion
	t.Cleanup(func() {
		helper.TalosGenerated = oldTalos
		helper.TalSecretFile = oldTalSecret
		talassist.LatestTalosVersion = oldVersion
	})

	helper.TalosGenerated = generatedDir
	helper.TalSecretFile = filepath.Join(generatedDir, "talsecret.yaml")
	talassist.LatestTalosVersion = "v1.7.0"
}

func TestGenTalSecretCreatesNewFile(t *testing.T) {
	td := t.TempDir()
	setupGenTalSecretTest(t, filepath.Join(td, "generated"))

	if err := genTalSecret(); err != nil {
		t.Fatalf("genTalSecret returned error: %v", err)
	}

	info, err := os.Stat(helper.TalSecretFile)
	if err != nil {
		t.Fatalf("expected TalSecretFile to exist: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("expected TalSecretFile to be non-empty")
	}
}

func TestGenTalSecretDirAlreadyExists(t *testing.T) {
	td := t.TempDir()
	setupGenTalSecretTest(t, filepath.Join(td, "generated"))

	// Pre-create the directory so MkdirAll is a no-op
	if err := os.MkdirAll(helper.TalosGenerated, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := genTalSecret(); err != nil {
		t.Fatalf("genTalSecret returned error: %v", err)
	}

	info, err := os.Stat(helper.TalSecretFile)
	if err != nil {
		t.Fatalf("expected TalSecretFile to exist: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("expected TalSecretFile to be non-empty")
	}
}

func TestGenTalSecretDeeplyNestedDir(t *testing.T) {
	td := t.TempDir()
	setupGenTalSecretTest(t, filepath.Join(td, "a", "b", "c", "generated"))

	if err := genTalSecret(); err != nil {
		t.Fatalf("genTalSecret returned error: %v", err)
	}

	info, err := os.Stat(helper.TalSecretFile)
	if err != nil {
		t.Fatalf("expected TalSecretFile to exist: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("expected TalSecretFile to be non-empty")
	}
}

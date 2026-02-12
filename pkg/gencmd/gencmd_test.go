package gencmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

func TestGenTalSecretCreatesFile(t *testing.T) {
	td := t.TempDir()
	// override helper paths
	oldTalos := helper.TalosGenerated
	oldTalSecret := helper.TalSecretFile
	defer func() { helper.TalosGenerated = oldTalos; helper.TalSecretFile = oldTalSecret }()

	helper.TalosGenerated = filepath.Join(td, "generated")
	helper.TalSecretFile = filepath.Join(helper.TalosGenerated, "talsecret.yaml")

	// create directory and a dummy tal secret file so genTalSecret takes the "already exists" branch
	if err := os.MkdirAll(helper.TalosGenerated, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(helper.TalSecretFile, []byte("dummy"), 0644); err != nil {
		t.Fatalf("write dummy tal secret: %v", err)
	}

	if err := genTalSecret(); err != nil {
		t.Fatalf("genTalSecret returned error when file exists: %v", err)
	}
}

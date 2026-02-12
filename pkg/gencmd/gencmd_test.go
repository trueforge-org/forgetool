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

    // ensure no file exists
    if _, err := os.Stat(helper.TalSecretFile); err == nil {
        os.Remove(helper.TalSecretFile)
    }

    if err := genTalSecret(); err != nil {
        t.Fatalf("genTalSecret returned error: %v", err)
    }

    if _, err := os.Stat(helper.TalSecretFile); err != nil {
        t.Fatalf("expected tal secret file to exist: %v", err)
    }
}
package gencmd

import "testing"

func TestSmokeGencmd(t *testing.T) {}

package gencmd

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	talhelperCfg "github.com/budimanjojo/talhelper/v3/pkg/config"
)

func TestGenUpgradeCoverage(t *testing.T) {
	resetGencmdHooks(t)
	generateUpgradeCommandFn = func(_ *talhelperCfg.TalhelperConfig, _ string, _ string, _ []string, _ bool) error {
		_, _ = fmt.Fprint(os.Stdout, "talosctl one;\ntalosctl two;\n")
		return nil
	}
	cmds := GenUpgrade("1.2.3.4", []string{"--x"})
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands, got %v", cmds)
	}
	if !strings.HasPrefix(cmds[0], "talosctl") {
		t.Fatalf("expected talosctl prefix, got %s", cmds[0])
	}

	resetGencmdHooks(t)
	called := false
	generateUpgradeCommandFn = func(_ *talhelperCfg.TalhelperConfig, _ string, _ string, _ []string, _ bool) error {
		return errors.New("boom")
	}
	upgradeFatalFn = func(error) { called = true }
	_ = GenUpgrade("1.2.3.4", nil)
	if !called {
		t.Fatal("expected fatal hook to be called")
	}

	resetGencmdHooks(t)
	generateUpgradeCommandFn = func(_ *talhelperCfg.TalhelperConfig, _ string, _ string, _ []string, _ bool) error {
		return errors.New("boom")
	}
	osExitFn = func(int) { panic(exitPanic{}) }
	expectExitPanic(t, func() {
		_ = GenUpgrade("1.2.3.4", nil)
	})
}

package gencmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/budimanjojo/talhelper/v3/pkg/generate"
	"github.com/trueforge-org/forgetool/pkg/helper"
	"github.com/trueforge-org/forgetool/pkg/talassist"
	talosctlpkg "github.com/trueforge-org/forgetool/pkg/talosctl"
)

func defaultUpgradeFatal(err error) {
	log.Error().Err(err).Msgf("failed to generate talosctl upgrade command: %s", err)
}

var (
	generateUpgradeCommandFn = generate.GenerateUpgradeCommand
	upgradeFatalFn           = defaultUpgradeFatal
)

// TODO: remove talhelper dependency for cmd creation
func GenUpgrade(node string, extraFlags []string) []string {
	// TODO: get rid of this, due to double uncontrollable log output

	upgradeStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	extraFlags = append(extraFlags, "--preserve")
	err := generateUpgradeCommandFn(talassist.TalConfig, helper.TalosGenerated, node, extraFlags, false)

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = upgradeStdout

	sliceOut := strings.Split(string(out), ";\n")
	talosPath := talosctlpkg.CommandPrefix()
	var slice []string
	for _, str := range sliceOut {
		if str != "" {
			str = strings.Replace(str, "talosctl", talosPath, 1)
			slice = append(slice, str)
		}

	}

	if err != nil {
		upgradeFatalFn(fmt.Errorf("failed to generate talosctl upgrade command: %w", err))
		osExitFn(1)
	}
	return slice
}

func GenKubeUpgrade(node string) string {
	talosPath := talosctlpkg.CommandPrefix()
	strout := talosPath + " upgrade-k8s --talosconfig " + helper.TalosConfigFile + " -n " + node
	return strout
}

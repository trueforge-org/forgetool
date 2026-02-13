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

func defaultUpgradeCloseReader(file *os.File) error {
	return file.Close()
}

func defaultUpgradeCloseWriter(file *os.File) error {
	return file.Close()
}

var (
	generateUpgradeCommandFn = generate.GenerateUpgradeCommand
	upgradeFatalFn           = defaultUpgradeFatal
	upgradePipeFn            = os.Pipe
	upgradeReadAllFn         = io.ReadAll
	upgradeCloseReaderFn     = defaultUpgradeCloseReader
	upgradeCloseWriterFn     = defaultUpgradeCloseWriter
)

// TODO: remove talhelper dependency for cmd creation
func GenUpgrade(node string, extraFlags []string) []string {
	// TODO: get rid of this, due to double uncontrollable log output

	upgradeStdout := os.Stdout
	r, w, pipeErr := upgradePipeFn()
	if pipeErr != nil {
		upgradeFatalFn(fmt.Errorf("failed to create pipe: %w", pipeErr))
		osExitFn(1)
	}
	defer func() {
		os.Stdout = upgradeStdout
		if closeErr := upgradeCloseReaderFn(r); closeErr != nil {
			log.Warn().Err(closeErr).Msg("failed to close pipe reader")
		}
	}()

	os.Stdout = w

	extraFlags = append(extraFlags, "--preserve")
	err := generateUpgradeCommandFn(talassist.TalConfig, helper.TalosGenerated, node, extraFlags, false)

	if closeErr := upgradeCloseWriterFn(w); closeErr != nil {
		log.Warn().Err(closeErr).Msg("failed to close pipe writer")
	}
	out, readErr := upgradeReadAllFn(r)
	if readErr != nil {
		upgradeFatalFn(fmt.Errorf("failed to read command output: %w", readErr))
		osExitFn(1)
	}

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

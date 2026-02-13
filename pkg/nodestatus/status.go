package nodestatus

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/trueforge-org/forgetool/pkg/helper"
	talosctlpkg "github.com/trueforge-org/forgetool/pkg/talosctl"
)

func hasUnknownAuthorityError(out string, err error) bool {
	if strings.Contains(out, "certificate signed by unknown authority") {
		return true
	}
	if err != nil && strings.Contains(err.Error(), "certificate signed by unknown authority") {
		return true
	}
	return false
}

func formatStatusError(out string, err error) error {
	if err == nil {
		return errors.New("status: " + out)
	}
	return fmt.Errorf("status: %s error: %w", out, err)
}

func baseStatusCMD(node string) []string {
	argsslice := [...]string{talosctlpkg.CommandPrefix(), "--talosconfig=" + path.Join(helper.ClusterPath, "talos", "generated", "talosconfig"), "-n", node, "-e", node, "get", "machinestatus"}

	log.Debug().Strs("command", argsslice[:]).Msg("Constructed base command for machine status")
	return argsslice[:]
}

func CheckNeedBootstrap(node string) (bool, error) {
	log.Info().Str("node", node).Msg("Checking if bootstrap is needed")

	argsslice := append(baseStatusCMD(node), "-o", "jsonpath={.spec.stage}")
	out, err := talosctlpkg.Run(argsslice[1:], true)
	if err != nil {
		log.Warn().Err(err).Str("output", string(out)).Msg("Error running command, checking for certificate issue")
		if hasUnknownAuthorityError(string(out), err) {
			log.Debug().Msg("Certificate signed by unknown authority; retrying with insecure flag")
			argsslice := append(baseStatusCMD(node), "-o", "jsonpath={.spec.stage}", "--insecure")
			out2, err2 := talosctlpkg.Run(argsslice[1:], true)
			if err2 != nil {
				formattedErr := formatStatusError(string(out), err2)
				log.Error().Err(formattedErr).Msg("Failed to get machine status with insecure fallback")
				return false, formattedErr
			}
			normalizedOut := strings.TrimSpace(string(out2))
			if normalizedOut != "" && strings.Contains(normalizedOut, "maintenance") {
				log.Info().Msg("Node is in maintenance; bootstrap needed")
				return true, nil
			}
		} else {
			formattedErr := formatStatusError(string(out), err)
			log.Error().Err(formattedErr).Msg("Failed to get machine status")
			return false, formattedErr
		}
	}
	log.Debug().Str("output", string(out)).Msg("No bootstrap needed; returning false")
	return false, nil
}

func CheckStatus(node string) (string, error) {
	log.Info().Str("node", node).Msg("Checking node status")

	argsslice := append(baseStatusCMD(node), "-o", "jsonpath={.spec.stage}")
	out, err := talosctlpkg.Run(argsslice[1:], true)
	if err != nil {
		log.Debug().Err(err).Str("output", string(out)).Msg("Error running command, checking for certificate issue")
		if hasUnknownAuthorityError(string(out), err) {
			log.Debug().Msg("Certificate signed by unknown authority; retrying with insecure flag")
			argsslice = append(baseStatusCMD(node), "-o", "jsonpath={.spec.stage}", "--insecure")
			out2, err2 := talosctlpkg.Run(argsslice[1:], true)
			if err2 != nil {
				formattedErr := formatStatusError(string(out), err2)
				log.Error().Err(formattedErr).Msg("Failed to get machine status with insecure fallback")
				return "ERROR", formattedErr
			}
			normalizedOut := strings.TrimSpace(string(out2))
			log.Info().Msg("Successfully retrieved node status with insecure flag")
			return normalizedOut, nil
		} else {
			formattedErr := formatStatusError(string(out), err)
			log.Error().Err(formattedErr).Msg("Failed to get machine status")
			return "ERROR", formattedErr
		}
	}
	normalizedOut := strings.TrimSpace(string(out))
	log.Debug().
		Int("statusLen", len(out)).
		Str("statusRaw", fmt.Sprintf("%q", out)).
		Str("statusNormalized", normalizedOut).
		Msg("Raw node status payload")
	log.Info().Str("status", normalizedOut).Msg("Node status retrieved successfully")
	return normalizedOut, nil
}

func CheckReadyStatus(node string, silent bool) (string, error) {
	log.Info().Str("node", node).Msg("Checking node readiness status")

	argsslice := append(baseStatusCMD(node), "-o", "jsonpath={.spec.status.ready}")
	out, err := talosctlpkg.Run(argsslice[1:], true)

	if err != nil {
		errstring := "status: " + string(out) + " error: " + err.Error()
		if !silent {
			log.Error().Msg(errstring)
		}
		return "ERROR", errors.New(errstring)
	}
	normalizedOut := strings.TrimSpace(string(out))
	if strings.Contains(normalizedOut, "true") {
		log.Info().Msg("Node is ready")
	} else {
		log.Warn().Msg("Node is not ready")
	}
	return normalizedOut, nil
}

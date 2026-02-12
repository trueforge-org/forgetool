package nodestatus

import (
	"errors"
	"strings"

	"github.com/rs/zerolog/log"
)

func CheckHealth(node string, status string, silent bool) error {
	log.Debug().Str("node", node).Str("expectedStatus", status).Msg("Starting health check")

	out, err := CheckStatus(node)
	if err != nil {
		errstring := "healthcheck failed. status: " + string(out) + " error: " + err.Error()
		if !silent {
			log.Error().Msgf("Healthcheck: check on node : failed %v", node)
			log.Error().Msgf("failed with error:  %s", errstring)
		}
		log.Error().Err(err).Str("node", node).Msg("Healthcheck failed")
		return errors.New(errstring)
	}

	out = strings.TrimSpace(out)
	if !silent {
		log.Info().Msgf("Healthcheck: node currently reporting status:  %v %v", node, out)
	}

	if err = evaluateHealthStatus(node, status, out, silent); err != nil {
		return err
	}
	log.Debug().Str("node", node).Msg("Health check completed successfully")
	return nil
}

func evaluateHealthStatus(node string, status string, out string, silent bool) error {
	if status != "" && strings.Contains(out, status) {
		if !silent {
			response := "Healthcheck: detected node " + node + " in mode " + status + " , continuing..."
			log.Info().Msg(response)
		}
		return nil
	}

	if status == "" && strings.Contains(out, "maintenance") {
		response := "Healthcheck: WARN detected node " + node + " in mode " + "maintenance" + ".\nLikely a new node, so trying commands anyway. Continuing..."
		log.Warn().Msg(response)
		return nil
	}

	if status == "" && strings.Contains(out, "running") {
		return validateReadyStatus(node, out, silent)
	}

	if !silent {
		log.Info().Msgf("Healthcheck: check on node : failed %v", node)
		log.Error().Str("node", node).Msg("Healthcheck failed with unexpected status")
	}

	return errors.New("healthcheck failed")
}

func validateReadyStatus(node string, out string, silent bool) error {
	_, err := CheckReadyStatus(node, silent)
	if err != nil {
		errstring := "healthcheck failed. status: " + out + " error: " + err.Error()
		if !silent {
			log.Error().Err(err).Str("node", node).Msg("Healthcheck failed while checking readiness")
		}
		return errors.New(errstring)
	}

	return nil
}

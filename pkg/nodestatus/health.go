package nodestatus

import (
	"errors"
	"strings"

	"github.com/rs/zerolog/log"
)

func CheckHealth(node string, status string, silent bool) error {
	log.Debug().Str("node", node).Str("expectedStatus", status).Bool("silent", silent).Msg("Starting health check")

	out, err := CheckStatus(node)
	if err != nil {
		log.Debug().Str("node", node).Err(err).Msg("CheckStatus returned error")
		errstring := "healthcheck failed. status: " + string(out) + " error: " + err.Error()
		if !silent {
			log.Error().Msgf("Healthcheck: check on node : failed %v", node)
			log.Error().Msgf("failed with error:  %s", errstring)
		} else {
			log.Debug().Str("node", node).Str("error", errstring).Msg("Silent mode enabled; suppressing non-debug health check failure logs")
		}
		log.Error().Err(err).Str("node", node).Msg("Healthcheck failed")
		return errors.New(errstring)
	}
	log.Debug().Str("node", node).Str("rawStatusOutput", out).Msg("CheckStatus completed")

	out = strings.TrimSpace(out)
	log.Debug().Str("node", node).Str("trimmedStatusOutput", out).Msg("Normalized status output")
	if !silent {
		log.Info().Msgf("Healthcheck: node currently reporting status:  %v %v", node, out)
	} else {
		log.Debug().Str("node", node).Str("currentStatus", out).Msg("Silent mode enabled; suppressing non-debug status report")
	}

	log.Debug().Str("node", node).Str("expectedStatus", status).Msg("Evaluating health status")
	if err = evaluateHealthStatus(node, status, out, silent); err != nil {
		log.Debug().Str("node", node).Err(err).Msg("Health status evaluation failed")
		return err
	}
	log.Debug().Str("node", node).Msg("Health check completed successfully")
	return nil
}

func evaluateHealthStatus(node string, status string, out string, silent bool) error {
	log.Debug().
		Str("node", node).
		Str("expectedStatus", status).
		Str("currentStatus", out).
		Bool("silent", silent).
		Msg("Entered health status evaluator")

	normalizedOut := strings.TrimSpace(out)
	if normalizedOut == "" || strings.EqualFold(normalizedOut, "null") {
		log.Error().Str("node", node).Str("currentStatus", normalizedOut).Msg("Healthcheck failed: null or empty status output")
		return errors.New("healthcheck failed: null or empty status output")
	}

	if status != "" && strings.Contains(normalizedOut, status) {
		log.Debug().Str("node", node).Str("matchedStatus", status).Msg("Expected status found in node output")
		if !silent {
			response := "Healthcheck: detected node " + node + " in mode " + status + " , continuing..."
			log.Info().Msg(response)
		} else {
			log.Debug().Str("node", node).Str("matchedStatus", status).Msg("Silent mode enabled; suppressing non-debug matched-status message")
		}
		return nil
	}

	if status == "" && strings.Contains(normalizedOut, "maintenance") {
		log.Debug().Str("node", node).Msg("Node in maintenance mode with no expected status configured")
		response := "Healthcheck: WARN detected node " + node + " in mode " + "maintenance" + ".\nLikely a new node, so trying commands anyway. Continuing..."
		log.Warn().Msg(response)
		return nil
	}

	if status == "" && strings.Contains(normalizedOut, "running") {
		log.Debug().Str("node", node).Msg("Node appears running; validating readiness")
		return validateReadyStatus(node, normalizedOut, silent)
	}

	log.Debug().
		Str("node", node).
		Str("expectedStatus", status).
		Str("currentStatus", out).
		Msg("No matching health status rule found")

	if !silent {
		log.Info().Msgf("Healthcheck: check on node : failed %v", node)
		log.Error().Str("node", node).Msg("Healthcheck failed with unexpected status")
	} else {
		log.Debug().
			Str("node", node).
			Str("currentStatus", normalizedOut).
			Msg("Silent mode enabled; suppressing non-debug unexpected-status logs")
	}

	return errors.New("healthcheck failed")
}

func validateReadyStatus(node string, out string, silent bool) error {
	log.Debug().Str("node", node).Msg("Starting readiness validation")
	_, err := CheckReadyStatus(node, silent)
	if err != nil {
		log.Debug().Str("node", node).Err(err).Msg("CheckReadyStatus returned error")
		errstring := "healthcheck failed. status: " + out + " error: " + err.Error()
		if !silent {
			log.Error().Err(err).Str("node", node).Msg("Healthcheck failed while checking readiness")
		} else {
			log.Debug().Str("node", node).Str("error", errstring).Msg("Silent mode enabled; suppressing non-debug readiness failure log")
		}
		return errors.New(errstring)
	}
	log.Debug().Str("node", node).Msg("Readiness validation succeeded")

	return nil
}

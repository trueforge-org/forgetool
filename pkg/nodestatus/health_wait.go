package nodestatus

import (
	"errors"
	"time"

	"github.com/rs/zerolog/log"
)

func WaitForHealth(node string, status []string) (string, error) {
	statusmsg, checks := buildStatusChecks(status)

	log.Info().Msgf("Healthcheck: Waiting for Node %s to reach status: %s", node, statusmsg)

	// Duration constants
	checkInterval := 10 * time.Second
	maxDuration := 15 * time.Minute

	// Create a ticker to run CheckHealth every 10 seconds
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	// Create a timer to stop the process after 15 minutes
	timer := time.NewTimer(maxDuration)
	defer timer.Stop()

	// Initial health check before starting the ticker
	for _, check := range checks {
		log.Debug().Str("node", node).Str("check", check).Msg("Performing initial health check")
		err := CheckHealth(node, check, true)
		if err == nil {
			log.Debug().Str("node", node).Str("status", check).Msg("Initial health check passed")
			return check, nil
		}
	}

	// Loop to run CheckHealth every 10 seconds for a maximum of 15 minutes
	for {
		select {
		case <-ticker.C:
			log.Debug().Msg("Running periodic health checks")
			for _, check := range checks {
				err := CheckHealth(node, check, true)
				if err == nil {
					log.Info().Str("node", node).Str("status", check).Msg("Periodic health check passed")
					return check, nil
				}
			}

		case <-timer.C:
			log.Info().Msg("Max duration reached. Stopping health checks.")
			return "ERROR", errors.New("timeout waiting for Node to boot")
		}
	}
}

func buildStatusChecks(status []string) (string, []string) {
	if len(status) == 0 {
		return "running", []string{""}
	}

	statusmsg := ""
	for _, check := range status {
		statusmsg += ", " + check
	}

	return statusmsg, status
}

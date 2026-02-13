package helper

import (
	"os"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/beevik/ntp"
)

var (
	checkSystemTimeNTPTimeFn = ntp.Time
	checkSystemTimeNowFn     = time.Now
	checkSystemTimeExitFn    = os.Exit
)

// IsTimeWithinThreshold checks whether the difference between two times is
// within the given threshold. This is extracted so it can be unit-tested
// without requiring NTP access.
func IsTimeWithinThreshold(systemTime, referenceTime time.Time, threshold time.Duration) bool {
	timeDifference := systemTime.Sub(referenceTime)
	return timeDifference > -threshold && timeDifference < threshold
}

// checkSystemTime compares the system time with an NTP server time and returns whether it's correct within the given threshold
func CheckSystemTime() bool {
	log.Info().Msg("Checking if System Time is correct...")
	threshold := 10 * time.Second

	// Get the time from an NTP server
	ntpTime, err := checkSystemTimeNTPTimeFn("pool.ntp.org")
	if err != nil {
		log.Info().Msgf("Failed to get NTP time: %v", err)
		return true
	}

	// Get the current system time
	systemTime := checkSystemTimeNowFn()

	// Check if the time difference is within the acceptable threshold
	if IsTimeWithinThreshold(systemTime, ntpTime, threshold) {
		log.Info().Msg("System Time is correct...")
	} else {
		log.Info().Msg("ERROR: System Time incorrect, please correct your systemtime:")
		log.Info().Msgf("System time: %v", systemTime)
		log.Info().Msgf("NTP time: %v", ntpTime)
		log.Info().Msgf("Aborting command!")
		checkSystemTimeExitFn(1)
	}
	return IsTimeWithinThreshold(systemTime, ntpTime, threshold)
}

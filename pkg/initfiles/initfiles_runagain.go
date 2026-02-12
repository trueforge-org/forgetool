package initfiles

import (
	"os"

	"github.com/rs/zerolog/log"
)

// Create the "RUNAGAIN" file
func createRunAgainFile() {
	file, err := os.Create("RUNAGAIN")
	if err != nil {
		log.Err(err).Msg("error creating runagain file...")
		return
	}
	defer file.Close()
	return
}

// Remove the "RUNAGAIN" file if it exists
func removeRunAgainFile() error {
	if CheckRunAgainFileExists() {
		err := os.Remove("RUNAGAIN")
		if err != nil {
			log.Err(err).Msg("error removing runagain file...")
			return err
		}
		log.Debug().Msg("RUNAGAIN file removed.")
	} else {
		log.Debug().Msg("RUNAGAIN file does not exist.")
	}
	return nil
}

// Check if the "RUNAGAIN" file exists
func CheckRunAgainFileExists() bool {
	_, err := os.Stat("RUNAGAIN")
	return !os.IsNotExist(err)
}

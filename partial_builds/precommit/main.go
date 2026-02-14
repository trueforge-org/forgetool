package main

import (
	"github.com/rs/zerolog/log"
	"github.com/trueforge-org/forgetool/pkg/sops"
)

func main() {
	if err := sops.CheckFilesAndReportEncryption(true, true); err != nil {
		log.Fatal().Err(err).Msg("Error checking files")
	}
}

package helper

import (
	"net"

	"github.com/rs/zerolog/log"
)

func checkDNSResolution(domain string) bool {
	_, err := net.LookupHost(domain)
	if err != nil {
		return false
	}
	return true
}

func checkAllDomains(domains []string, verbose bool) {
	for _, domain := range domains {
		if !checkDNSResolution(domain) {
			log.Fatal().Msgf("DNS for %s does not resolve.", domain)
		} else {
			if verbose {
				log.Info().Msgf("DNS for %s resolves.\n", domain)
			}
		}
	}
}

func CheckReqDomains() {
	domains := []string{
		"oci.trueforge.org",
		"ghcr.io",
		"github.com",
		"docker.com",
	}

	checkAllDomains(domains, false)
}

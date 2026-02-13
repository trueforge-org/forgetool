package talhelperutil

import (
	"os"

	"github.com/rs/zerolog/log"

	"github.com/trueforge-org/forgetool/pkg/helper"
	"gopkg.in/yaml.v3"
)

var (
	talhelperutilFatalFn = defaultTalhelperutilFatal
	talhelperutilExitFn  = os.Exit
)

func defaultTalhelperutilFatal(err error, msg string) {
	log.Error().Err(err).Msg(msg)
}

// Node represents the structure of each node in the YAML file
type Node struct {
	Hostname     string `yaml:"hostname"`
	IPAddress    string `yaml:"ipAddress"`
	ControlPlane bool   `yaml:"controlPlane"`
}

// Config represents the structure of the YAML file
type Config struct {
	Nodes []Node `yaml:"nodes"`
}

func ExtractIPs() {
	log.Trace().Msg("Starting the ExtractIPs function")

	config := loadTalConfig()
	resetIPBuckets()
	populateIPBuckets(config.Nodes)

	log.Trace().Msg("Finished processing nodes")
	log.Info().Int("totalIPs", len(helper.AllIPs)).Msg("Total IPs extracted")
}

func loadTalConfig() Config {
	file, err := os.ReadFile("config.yaml")
	if err != nil {
		talhelperutilFatalFn(err, "Failed to read file: config.yaml")
		talhelperutilExitFn(1)
		return Config{}
	}
	log.Debug().Msg("Successfully read the YAML file")

	var config Config
	err = yaml.Unmarshal(file, &config)
	if err != nil {
		talhelperutilFatalFn(err, "Failed to unmarshal YAML")
		talhelperutilExitFn(1)
		return Config{}
	}
	log.Debug().Msg("Successfully unmarshaled YAML into Config struct")

	return config
}

func resetIPBuckets() {
	log.Info().Msg("Resetting global IP storage variables")
	helper.AllIPs = []string{}
	helper.ControlPlaneIPs = []string{}
	helper.WorkerIPs = []string{}
}

func populateIPBuckets(nodes []Node) {
	log.Debug().Msg("Looping through nodes to segregate IP addresses")
	for _, node := range nodes {
		log.Debug().
			Str("hostname", node.Hostname).
			Str("ipAddress", node.IPAddress).
			Bool("controlPlane", node.ControlPlane).
			Msg("Processing node")

		helper.AllIPs = append(helper.AllIPs, node.IPAddress)
		if node.ControlPlane {
			helper.ControlPlaneIPs = append(helper.ControlPlaneIPs, node.IPAddress)
			log.Info().Str("ipAddress", node.IPAddress).Msg("Added to ControlPlaneIPs")
			continue
		}

		helper.WorkerIPs = append(helper.WorkerIPs, node.IPAddress)
		log.Info().Str("ipAddress", node.IPAddress).Msg("Added to WorkerIPs")
	}
}

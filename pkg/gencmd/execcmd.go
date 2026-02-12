package gencmd

import (
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/trueforge-org/forgetool/pkg/helper"
	"github.com/trueforge-org/forgetool/pkg/nodestatus"
)

func ExecCmd(cmd string) {
	argslice := strings.Split(cmd, " ")
	log.Trace().Msgf("command %v", argslice)

	// log.Info().Msg("test", strings.Join(argslice, " "))
	//nolint:ineffassign
	out, err := helper.RunCommand(argslice, false)
	if err != nil {
		log.Info().Msgf("err:  %v", err)
		if strings.Contains(cmd, "bootstrap") {
			log.Info().Msg("Bootstrap: Fail, retrying...")
			time.Sleep(5 * time.Second)
			out, err = helper.RunCommand(argslice, false)

			if err != nil && strings.Contains(string(out), "bootstrap is not available yet") {
				start := time.Now()
				timeout := 2 * time.Minute

				for {
					log.Info().Msg("Bootstrap: Fail, retrying...")
					time.Sleep(5 * time.Second)

					out, err = helper.RunCommand(argslice, false)
					if err != nil || !strings.Contains(string(out), "bootstrap is not available yet") {
						break
					}
					if time.Since(start) >= timeout {
						log.Info().Msg("Timeout reached: Node not ready for bootstrap within 2 minutes.")
						break
					}
				}
			}
		}

	}
}

func ExecCmds(taloscmds []string, healthcheck bool) error {
	log.Info().Msg("Regenerating config prior to commands...")
	GenConfig([]string{})
	todocmds, skipped := buildTodoCmds(taloscmds, healthcheck)

	log.Info().Msg("Executing Cmds...")
	for _, command := range todocmds {
		node := helper.ExtractNode(command)
		runNodeCommand(command, node)
		time.Sleep(15 * time.Second)

		if healthcheck {
			checkNodePostCommandHealth(node)
		}
	}

	if healthcheck && len(taloscmds) > 0 && !skipped && !strings.Contains(taloscmds[0], "upgrade") {
		log.Info().Msg("Checking if cluster is healthy after commands...")
		healthcmd := GenPlain("health", helper.TalEnv["VIP_IP"], []string{})
		ExecCmd(healthcmd[0])
	}
	return nil
}

func buildTodoCmds(taloscmds []string, healthcheck bool) ([]string, bool) {
	if !healthcheck {
		return taloscmds, false
	}

	log.Info().Msg("Pre-Run Healthchecks...")
	todocmds := make([]string, 0, len(taloscmds))
	skipped := false

	for _, command := range taloscmds {
		node := helper.ExtractNode(command)
		log.Info().Msgf("checking node availability:  %v", node)
		err := nodestatus.CheckHealth(node, "", false)
		if err != nil {
			log.Info().Msgf("node seems not to be runnign correctly and cannot be used %v", node)
			log.Info().Msgf("node This will also make it impossible to poll total-cluster-health as well... %v", node)
			if !helper.GetYesOrNo("Do you want to continue without this node? (yes/no) [y/n]: ") {
				log.Info().Msg("Exiting...")
				os.Exit(1)
			}
			skipped = true
		}
		todocmds = append(todocmds, command)
	}

	if skipped {
		log.Info().Msg("skipping cluster health check due to unhealthy nodes being ignored...")
		return todocmds, true
	}

	if helper.GetYesOrNo("Do you want to check the health of the cluster? (yes/no) [y/n]: ") {
		log.Info().Msg("Checking if cluster is healthy...")
		healthcmd := GenPlain("health", helper.TalEnv["VIP_IP"], []string{})
		ExecCmd(healthcmd[0])
		return todocmds, false
	}

	return todocmds, true
}

func runNodeCommand(command string, node string) {
	log.Info().Msgf("Executing commands on node:  %v", node)
	argslice := strings.Split(command, " ")
	log.Debug().Msgf("running command: %s", command)
	out, err := helper.RunCommand(argslice, false)
	if err == nil {
		return
	}

	if strings.Contains(string(out), "certificate signed by unknown authority") {
		argslice = append(argslice, "--insecure")
		log.Debug().Msgf("Re-Running command using insecure flag: %s", command)
		if _, err2 := helper.RunCommand(argslice, false); err2 != nil {
			log.Info().Msgf("err:  %v", err2)
		}
		return
	}

	log.Info().Msgf("err:  %v", err)
	log.Info().Msgf("node has thrown an error... %v", node)
	if !helper.GetYesOrNo("Are you sure you want to continue applying this to other nodes? (yes/no) [y/n]: ") {
		log.Info().Msg("Exiting...")
		os.Exit(1)
	}
}

func checkNodePostCommandHealth(node string) {
	log.Info().Msgf("checking if node is back online:  %v", node)
	err := nodestatus.CheckHealth(node, "", false)
	if err == nil {
		return
	}

	log.Info().Msgf("node seems not to be running correctly... %v", node)
	if !helper.GetYesOrNo("Are you sure you want to continue applying this to other nodes? (yes/no) [y/n]: ") {
		log.Info().Msg("Exiting...")
		os.Exit(1)
	}
}

package gencmd

import (
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/trueforge-org/forgetool/pkg/helper"
	"github.com/trueforge-org/forgetool/pkg/nodestatus"
	talosctlpkg "github.com/trueforge-org/forgetool/pkg/talosctl"
)

var (
	runTalosctlCommandFn  = talosctlpkg.RunCommand
	sleepFn               = time.Sleep
	nowFn                 = time.Now
	sinceFn               = time.Since
	bootstrapRetryTimeout = 2 * time.Minute
	genConfigFn           = GenConfig
	extractNodeFn         = helper.ExtractNode
	checkNodeHealthFn     = nodestatus.CheckHealth
	getYesOrNoFn          = helper.GetYesOrNo
)

func ExecCmd(cmd string) {
	argslice := strings.Split(cmd, " ")
	log.Trace().Msgf("command %v", argslice)

	// log.Info().Msg("test", strings.Join(argslice, " "))
	//nolint:ineffassign
	out, err := runTalosctlCommandFn(argslice, false)
	if err != nil {
		log.Info().Msgf("err:  %v", err)
		if strings.Contains(cmd, "bootstrap") {
			log.Info().Msg("Bootstrap: Fail, retrying...")
			sleepFn(5 * time.Second)
			out, err = runTalosctlCommandFn(argslice, false)

			if err != nil && strings.Contains(string(out), "bootstrap is not available yet") {
				start := nowFn()
				timeout := bootstrapRetryTimeout

				for {
					log.Info().Msg("Bootstrap: Fail, retrying...")
					sleepFn(5 * time.Second)

					out, err = runTalosctlCommandFn(argslice, false)
					if err != nil || !strings.Contains(string(out), "bootstrap is not available yet") {
						break
					}
					if sinceFn(start) >= timeout {
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
	genConfigFn([]string{})
	todocmds, skipped := buildTodoCmds(taloscmds, healthcheck)

	log.Info().Msg("Executing Cmds...")
	for _, command := range todocmds {
		node := extractNodeFn(command)
		runNodeCommand(command, node)
		sleepFn(15 * time.Second)

		if healthcheck {
			checkNodePostCommandHealth(node)
		}
	}

	if healthcheck && len(taloscmds) > 0 && !skipped && !strings.Contains(taloscmds[0], "upgrade") {
		log.Info().Msg("Checking if cluster is healthy after commands...")
		healthcmd := genPlainFn("health", helper.TalEnv["VIP_IP"], []string{})
		execCmdFn(healthcmd[0])
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
		node := extractNodeFn(command)
		log.Info().Msgf("checking node availability:  %v", node)
		err := checkNodeHealthFn(node, "", false)
		if err != nil {
			log.Info().Msgf("node seems not to be runnign correctly and cannot be used %v", node)
			log.Info().Msgf("node This will also make it impossible to poll total-cluster-health as well... %v", node)
			if !getYesOrNoFn("Do you want to continue without this node? (yes/no) [y/n]: ", true) {
				log.Info().Msg("Exiting...")
				osExitFn(1)
			}
			skipped = true
		}
		todocmds = append(todocmds, command)
	}

	if skipped {
		log.Info().Msg("skipping cluster health check due to unhealthy nodes being ignored...")
		return todocmds, true
	}

	if getYesOrNoFn("Do you want to check the health of the cluster? (yes/no) [y/n]: ", true) {
		log.Info().Msg("Checking if cluster is healthy...")
		healthcmd := genPlainFn("health", helper.TalEnv["VIP_IP"], []string{})
		execCmdFn(healthcmd[0])
		return todocmds, false
	}

	return todocmds, true
}

func runNodeCommand(command string, node string) {
	log.Info().Msgf("Executing commands on node:  %v", node)
	argslice := strings.Split(command, " ")
	log.Debug().Msgf("running command: %s", command)
	out, err := runTalosctlCommandFn(argslice, false)
	if err == nil {
		return
	}

	if strings.Contains(string(out), "certificate signed by unknown authority") {
		argslice = append(argslice, "--insecure")
		log.Debug().Msgf("Re-Running command using insecure flag: %s", command)
		if _, err2 := runTalosctlCommandFn(argslice, false); err2 != nil {
			log.Info().Msgf("err:  %v", err2)
		}
		return
	}

	if strings.Contains(err.Error(), "certificate signed by unknown authority") {
		argslice = append(argslice, "--insecure")
		log.Debug().Msgf("Re-Running command using insecure flag: %s", command)
		if _, err2 := runTalosctlCommandFn(argslice, false); err2 != nil {
			log.Info().Msgf("err:  %v", err2)
		}
		return
	}

	log.Info().Msgf("err:  %v", err)
	log.Info().Msgf("node has thrown an error... %v", node)
	if !getYesOrNoFn("Are you sure you want to continue applying this to other nodes? (yes/no) [y/n]: ", true) {
		log.Info().Msg("Exiting...")
		osExitFn(1)
	}
}

func checkNodePostCommandHealth(node string) {
	log.Info().Msgf("checking if node is back online:  %v", node)
	err := checkNodeHealthFn(node, "", false)
	if err == nil {
		return
	}

	log.Info().Msgf("node seems not to be running correctly... %v", node)
	if !getYesOrNoFn("Are you sure you want to continue applying this to other nodes? (yes/no) [y/n]: ", true) {
		log.Info().Msg("Exiting...")
		osExitFn(1)
	}
}

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/go-logr/zerologr"
	"github.com/trueforge-org/forgetool/cmd"
	"github.com/trueforge-org/forgetool/embed"
	"github.com/trueforge-org/forgetool/pkg/helper"
	k8slog "sigs.k8s.io/controller-runtime/pkg/log"
)

var Version = "dev"
var noColor = false

func main() {
	configureLogging()
	printBanner()
	embed.AllToCache()
	helper.CheckSystemTime()
	helper.CheckReqDomains()
	runCommand()
}

func configureLogging() {
	zerolog.DurationFieldUnit = time.Second
	if os.Getenv("DEBUG") != "" {
		noColor = true
	}

	zerolog.SetGlobalLevel(parseLogLevel(os.Getenv("LOGLEVEL")))
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
		NoColor:    noColor,
	})

	zlogger := zerolog.New(os.Stderr).With().Timestamp().Logger()
	k8slog.SetLogger(zerologr.New(&zlogger))
}

func parseLogLevel(level string) zerolog.Level {
	switch level {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	case "info":
		return zerolog.InfoLevel
	default:
		return zerolog.InfoLevel
	}
}

func printBanner() {
	fmt.Printf("\n%s\n", helper.Logo)
	fmt.Printf("---\nForgetool Version: %s\n---\n", Version)
}

func runCommand() {
	if err := cmd.Execute(); err != nil {
		log.Fatal().Err(err).Msg("Failed to execute command")
	}
}

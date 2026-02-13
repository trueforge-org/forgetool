package cmd

import (
	"errors"
	"os"
	"os/exec"
	"testing"
)

func TestHelperDefaultErrorHandlerProcess(t *testing.T) {
	if os.Getenv("GO_WANT_CMD_DEFAULT_ONERROR_HELPER") != "1" {
		return
	}

	err := errors.New("boom")
	switch os.Getenv("CMD_ONERROR_HANDLER") {
	case "charts_bump":
		chartsBumpOnError(err)
	case "charts_tagclean":
		chartsTagCleanOnError(err)
	case "charts_deps":
		chartsDepsOnError(err)
	case "charts_genchartlist":
		chartsGenChartListOnError(err)
	case "charts_genchangelog":
		chartsGenChangelogOnError(err)
	case "charts_genmeta":
		chartsGenMetaOnError(err)
	case "adv_gentooldocs":
		genToolDocsOnError(err)
	default:
		os.Exit(2)
	}

	os.Exit(0)
}

func TestDefaultErrorHandlersExitNonZero(t *testing.T) {
	handlers := []string{
		"charts_bump",
		"charts_tagclean",
		"charts_deps",
		"charts_genchartlist",
		"charts_genmeta",
		"adv_gentooldocs",
	}

	for _, handler := range handlers {
		handler := handler
		t.Run(handler, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=TestHelperDefaultErrorHandlerProcess")
			cmd.Env = append(os.Environ(),
				"GO_WANT_CMD_DEFAULT_ONERROR_HELPER=1",
				"CMD_ONERROR_HANDLER="+handler,
			)

			err := cmd.Run()
			if err == nil {
				t.Fatalf("expected non-zero exit for handler %s", handler)
			}
			exitErr, ok := err.(*exec.ExitError)
			if !ok || exitErr.ExitCode() == 0 {
				t.Fatalf("expected non-zero exit code for handler %s, got %v", handler, err)
			}
		})
	}
}

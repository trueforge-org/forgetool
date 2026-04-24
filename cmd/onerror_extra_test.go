package cmd

import (
	"errors"
	"os"
	"os/exec"
	"testing"
)

// onErrorLambdas maps env-variable trigger names to the production OnError
// lambdas. Each lambda calls log.Fatal* which terminates the process; we
// therefore exercise them in a re-execed subprocess and assert non-zero exit.
func onErrorLambdas() map[string]func(error) {
	return map[string]func(error){
		"ADV_GENTOOLDOCS":         genToolDocsOnError,
		"CHARTS_BUMP":             chartsBumpOnError,
		"CHARTS_DEPS":             chartsDepsOnError,
		"CHARTS_GENCHARTLIST":     chartsGenChartListOnError,
		"CHARTS_GENDOCS":          chartsGenDocsOnError,
		"CHARTS_GENMETA":          chartsGenMetaOnError,
		"CHARTS_TAGCLEAN":         chartsTagCleanOnError,
		"CONTAINERS_GENCHANGELOG": containersGenChangelogOnError,
		"CONTAINERS_GENDOCS":      containersGenDocsOnError,
		"CONTAINERS_GENLIST":      containersGenListOnError,
	}
}

// TestHelperProcessOnErrorFatal is the subprocess helper. When invoked with
// GO_WANT_ONERROR_FATAL=<key>, it calls the corresponding OnError lambda and
// is expected to exit non-zero via log.Fatal.
func TestHelperProcessOnErrorFatal(t *testing.T) {
	key := os.Getenv("GO_WANT_ONERROR_FATAL")
	if key == "" {
		return
	}
	fn, ok := onErrorLambdas()[key]
	if !ok {
		os.Exit(0)
	}
	fn(errors.New("forced test failure"))
	os.Exit(0)
}

// onErrorNonFatal lists keys whose lambda chains log.Fatal() without a
// terminating .Msg/.Send/.Msgf, so the process does NOT exit.
var onErrorNonFatal = map[string]bool{
	"CONTAINERS_GENCHANGELOG": true,
}

func TestOnError_FatalSubprocess(t *testing.T) {
	for key := range onErrorLambdas() {
		key := key
		t.Run(key, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcessOnErrorFatal")
			cmd.Env = append(os.Environ(), "GO_WANT_ONERROR_FATAL="+key)
			err := cmd.Run()
			if onErrorNonFatal[key] {
				if err != nil {
					t.Fatalf("%s: expected clean exit; got %v", key, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("%s: expected non-zero exit from log.Fatal", key)
			}
			if _, ok := err.(*exec.ExitError); !ok {
				t.Fatalf("%s: unexpected error type: %T", key, err)
			}
		})
	}
}

package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/siderolabs/talos/cmd/talosctl/cmd/mgmt"
	_ "github.com/siderolabs/talos/cmd/talosctl/cmd/mgmt/cluster/create" // import to get the command registered via the init() function.
	"github.com/siderolabs/talos/cmd/talosctl/cmd/talos"
	talosctlpkg "github.com/trueforge-org/forgetool/pkg/talosctl"
)

func captureProcessIO(run func() error) (string, error) {
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return "", err
	}
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		_ = stdoutReader.Close()
		_ = stdoutWriter.Close()
		return "", err
	}

	oldStdout := os.Stdout
	oldStderr := os.Stderr
	os.Stdout = stdoutWriter
	os.Stderr = stderrWriter

	var stdoutBuf, stderrBuf bytes.Buffer
	stdoutDone := make(chan struct{})
	stderrDone := make(chan struct{})

	go func() {
		_, _ = io.Copy(&stdoutBuf, stdoutReader)
		close(stdoutDone)
	}()

	go func() {
		_, _ = io.Copy(&stderrBuf, stderrReader)
		close(stderrDone)
	}()

	runErr := run()

	_ = stdoutWriter.Close()
	_ = stderrWriter.Close()
	<-stdoutDone
	<-stderrDone

	os.Stdout = oldStdout
	os.Stderr = oldStderr

	_ = stdoutReader.Close()
	_ = stderrReader.Close()

	return stdoutBuf.String() + stderrBuf.String(), runErr
}

var talosctlLongHelp = strings.TrimSpace(`
These are all talosctlanced commands that should generally not be needed

`)

var internalTalosctlMu sync.Mutex
var internalTalosctl = buildInternalTalosctlCommand()

var talosctl = &cobra.Command{
	Use:           "talosctl",
	Short:         "A CLI for out-of-band management of Kubernetes nodes created by Talos",
	Example:       "forgetool talosctl <bootstrap/health/precommit>",
	Long:          talosctlLongHelp,
	SilenceUsage:  true,
	SilenceErrors: true,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		out, err := runTalosctlArgs(args, false)
		if out != "" {
			fmt.Fprint(cmd.OutOrStdout(), out)
		}
		return err
	},
}

func buildInternalTalosctlCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "talosctl",
		Short:         "A CLI for out-of-band management of Kubernetes nodes created by Talos",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	for _, command := range mgmt.Commands {
		root.AddCommand(command)
	}

	for _, command := range talos.Commands {
		root.AddCommand(command)
	}

	return root
}

func runTalosctlArgs(args []string, silent bool) (string, error) {
	internalTalosctlMu.Lock()
	defer internalTalosctlMu.Unlock()

	var stdoutBuf, stderrBuf bytes.Buffer
	stdoutWriter := io.Writer(&stdoutBuf)
	stderrWriter := io.Writer(&stderrBuf)

	if !silent {
		stdoutWriter = io.MultiWriter(&stdoutBuf, os.Stdout)
		stderrWriter = io.MultiWriter(&stderrBuf, os.Stderr)
	}

	internalTalosctl.SetOut(stdoutWriter)
	internalTalosctl.SetErr(stderrWriter)
	internalTalosctl.SetArgs(args)

	var err error
	processOut := ""
	if silent {
		processOut, err = captureProcessIO(func() error {
			return internalTalosctl.Execute()
		})
	} else {
		err = internalTalosctl.Execute()
	}
	internalTalosctl.SetArgs(nil)

	return stdoutBuf.String() + stderrBuf.String() + processOut, err
}

func init() {
	talosctlpkg.SetExecutor(runTalosctlArgs)

	RootCmd.AddCommand(talosctl)
}

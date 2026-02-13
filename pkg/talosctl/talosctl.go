package talosctl

import (
	"fmt"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

var executor = func(args []string, silent bool) (string, error) {
	commandSlice := append([]string{CommandPrefix()}, args...)
	return helper.RunCommand(commandSlice, silent)
}

func SetExecutor(customExecutor func(args []string, silent bool) (string, error)) {
	if customExecutor == nil {
		return
	}

	executor = customExecutor
}

func CommandPrefix() string {
	return "talosctl"
}

func Run(args []string, silent bool) (string, error) {
	return executor(args, silent)
}

func RunCommand(commandSlice []string, silent bool) (string, error) {
	if len(commandSlice) == 0 {
		return "", fmt.Errorf("commandSlice cannot be empty")
	}

	if commandSlice[0] == CommandPrefix() {
		return Run(commandSlice[1:], silent)
	}

	return helper.RunCommand(commandSlice, silent)
}

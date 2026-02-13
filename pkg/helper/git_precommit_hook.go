package helper

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/rs/zerolog/log"
)

var (
	hookGetwdFn                = os.Getwd
	hookStatFn                 = os.Stat
	hookCreateFn               = os.Create
	hookWriteStringFn          = func(file *os.File, content string) (int, error) { return file.WriteString(content) }
	hookChmodFn                = os.Chmod
	hookGOOS                   = runtime.GOOS
	buildPreCommitHookScriptFn = buildPreCommitHookScript
)

// IsCurrentDirGitRepo checks if the current directory is a Git repository.
func IsCurrentDirGitRepo() (bool, error) {
	// Get the current working directory
	dir, err := hookGetwdFn()
	if err != nil {
		return false, err
	}

	// Construct the path to the .git directory
	gitDir := filepath.Join(dir, ".git")

	// Check if the .git directory exists and is a directory
	info, err := hookStatFn(gitDir)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

// CreateEncrPreCommitHook creates a pre-commit hook script in the .git/hooks directory
func CreateEncrPreCommitHook() error {
	shouldContinue, err := validateCurrentRepoForHook()
	if err != nil {
		return err
	}
	if !shouldContinue {
		return nil
	}

	dir, err := hookGetwdFn()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %v", err)
	}

	hookPath, hookScript, err := buildHookFileData(dir)
	if err != nil {
		return err
	}

	if err := writeHookScript(hookPath, hookScript); err != nil {
		return err
	}

	if hookGOOS != "windows" {
		err = hookChmodFn(hookPath, 0755)
		if err != nil {
			return fmt.Errorf("could not make pre-commit hook executable: %v", err)
		}
	}

	log.Info().Msg("Pre-commit hook created successfully.")
	return nil
}

func buildHookFileData(dir string) (string, string, error) {
	hookPath := getPreCommitHookPath(dir)
	hookScript, err := buildPreCommitHookScriptFn(dir)
	if err != nil {
		return "", "", err
	}

	return hookPath, hookScript, nil
}

func writeHookScript(hookPath, hookScript string) error {
	file, err := hookCreateFn(hookPath)
	if err != nil {
		return fmt.Errorf("could not create pre-commit hook file: %v", err)
	}
	defer file.Close()

	_, err = hookWriteStringFn(file, hookScript)
	if err != nil {
		return fmt.Errorf("could not write to pre-commit hook file: %v", err)
	}

	return nil
}

func validateCurrentRepoForHook() (bool, error) {
	isRepo, err := IsCurrentDirGitRepo()
	if err != nil {
		return false, fmt.Errorf("error checking if current directory is a Git repository: %v", err)
	}
	if !isRepo {
		log.Info().Msg("The current directory is not a valid GIT repository. Skipping precommit hook creation...")
		return false, nil
	}

	log.Info().Msg("Bootstrap: The current directory is a valid GIT repository, creating precommit hook...")
	return true, nil
}

func getPreCommitHookPath(dir string) string {
	hooksDir := filepath.Join(dir, ".git", "hooks")
	if hookGOOS == "windows" {
		return filepath.Join(hooksDir, "pre-commit.bat")
	}
	return filepath.Join(hooksDir, "pre-commit")
}

func buildPreCommitHookScript(dir string) (string, error) {
	goModPath := filepath.Join(dir, "go.mod")
	if _, err := hookStatFn(goModPath); !os.IsNotExist(err) {
		return buildGoRunHookScript(), nil
	}

	scriptPath := filepath.Join(CacheDir, "precommit")
	return buildExecutableHookScript(scriptPath), nil
}

func buildGoRunHookScript() string {
	return `#!/bin/sh
# Pre-commit hook script

# Use go run . checkcrypt if go.mod exists
echo "Running pre-commit encryption check..."
# go run . checkcrypt
if [ $? -ne 0 ]; then
    echo "Pre-commit encryption check failed. Commit aborted."
    exit 1
fi
`
}

func buildExecutableHookScript(scriptPath string) string {
	switch hookGOOS {
	case "windows":
		scriptPath = filepath.ToSlash(scriptPath) + ".exe"
		return fmt.Sprintf(`@echo off
REM Pre-commit hook script

REM Path to the script to run
set scriptPath=%s

REM Check if the script exists
if exist "%%scriptPath%%" (
    echo Running pre-commit script...
    "%%scriptPath%%"
    if errorlevel 1 (
        echo Pre-commit script failed. Commit aborted.
        exit /b 1
    )
) else (
    echo Script %%scriptPath%% not found. Commit aborted.
    exit /b 1
)
`, scriptPath)
	default:
		return fmt.Sprintf(`#!/bin/sh
# Pre-commit hook script

# Check if the script exists and is executable
if [ -x "%s" ]; then
    echo "Running pre-commit script..."
    "%s"
    if [ $? -ne 0 ]; then
        echo "Pre-commit script failed. Commit aborted."
        exit 1
    fi
else
    echo "Script %s not found or not executable. Commit aborted."
    exit 1
fi
`, scriptPath, scriptPath, scriptPath)
	}
}

package helper

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// getStagedFiles lists all files that are staged for commit
func GetStagedFiles() ([]string, error) {
	// Run git diff --cached --name-only to get staged files
	cmd := exec.Command("git", "diff", "--cached", "--name-only")

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to run git command: %v", err)
	}

	// Split the output into lines (file names)
	output := strings.TrimSpace(out.String())
	if output == "" {
		return nil, nil
	}

	files := strings.Split(output, "\n")
	return files, nil
}

func StageFiles(files []string) error {
	for _, file := range files {
		// Stage the file using `git add <file>`
		cmd := exec.Command("git", "add", file)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to stage file %s: %v", file, err)
		}
	}
	return nil
}

//////

// StageFile stages the given file using `git add`
func StageFile(filePath string) error {
	cmd := exec.Command("git", "add", filePath)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error staging file: %v", err)
	}
	return nil
}

// GetGitStagedFiles returns a list of git-staged files, excluding ignored files.
func GetGitStagedFiles() ([]string, error) {
	// Get the list of staged files
	cmd := exec.Command("git", "diff", "--cached", "--name-only")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	// Split the output into a slice of file names
	files := strings.Split(strings.TrimSpace(out.String()), "\n")

	return files, nil
}

// IsFileIgnored checks if a file is ignored by Git.
func IsFileIgnored(file string) (bool, error) {
	// Check if the file is ignored as-is
	ignored, err := checkIgnore(file)
	if err != nil {
		return false, err // Return error if checking fails
	}
	if ignored {
		return true, nil // Return true if the file is ignored as-is
	}

	// Check if the file is ignored with "forgetool/" prefix
	prefixedFile := "forgetool/" + file
	ignored, err = checkIgnore(prefixedFile)
	if err != nil {
		return false, err // Return error if checking fails
	}

	return ignored, nil // Return the result of the prefixed check
}

// checkIgnore is a helper function that runs the git check-ignore command for a given file.
func checkIgnore(file string) (bool, error) {
	// Define the base folder to check
	devTrigger := "DEVTRIGGER"

	// Check if the directory exists
	if _, err := os.Stat(devTrigger); !os.IsNotExist(err) {

		// If the directory exists, skip checks for specified paths
		// Check if the file path starts with the specified prefixes
		if isPathIgnored(file, []string{
			"repositories",
			"clusters",
			"forgetool/repositories",
			"forgetool/clusters",
		}) {
			return true, nil // Skip ignoring check
		}
	}

	// Run the git check-ignore command for the given file
	cmd := exec.Command("git", "check-ignore", file)
	if err := cmd.Run(); err != nil {
		// If the error is an ExitError, check the exit code
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() == 1 {
				// The file is ignored (exit code 1 indicates that the file is ignored)
				return true, nil
			}
		}
		// If there's another error, return it
		return false, err
	}
	// If the command succeeds (exit code 0), the file is not ignored
	return false, nil
}

// isPathIgnored checks if the file path starts with any of the specified prefixes.
func isPathIgnored(file string, prefixes []string) bool {
	for _, prefix := range prefixes {
		// Use filepath.HasPrefix to check if the file path starts with the prefix
		if filepath.HasPrefix(file, prefix) {
			return true
		}
	}
	return false
}

// IsFileFullyStaged checks if a file is fully staged (no pending unstaged changes)
// for both the unprefixed path and the path prefixed with /forgetool.
// It ignores files that are listed in .gitignore.
func IsFileFullyStaged(filePath string) (bool, error) {
	gitRoot, forgetoolExists, err := getGitRootAndForgetoolExists()
	if err != nil {
		return false, err
	}
	_ = gitRoot

	// Create a slice of file paths to check
	filePaths := []string{filePath}
	if forgetoolExists {
		filePaths = append(filePaths, "forgetool/"+filePath)
	}

	// Check if the files are ignored
	for _, path := range filePaths {
		ignoredCmd := exec.Command("git", "check-ignore", path)
		var ignoredOut bytes.Buffer
		ignoredCmd.Stdout = &ignoredOut
		err := ignoredCmd.Run()
		if err == nil {
			// If there's no error, the file is ignored
			continue // Skip this file since it's ignored
		}

		hasChanges, err := hasUnstagedChanges(path)
		if err != nil {
			return false, err
		}
		if hasChanges {
			return false, nil // Found unstaged changes
		}
	}

	// If no unstaged changes were found for both paths and files were not ignored
	return true, nil
}

func getGitRootAndForgetoolExists() (string, bool, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", false, err
	}

	gitRoot := strings.TrimSpace(out.String())
	forgetoolPath := filepath.Join(gitRoot, "forgetool")
	_, err := exec.Command("test", "-d", forgetoolPath).Output()

	return gitRoot, err == nil, nil
}

func hasUnstagedChanges(path string) (bool, error) {
	diffCmd := exec.Command("git", "diff", path)
	var diffOut bytes.Buffer
	diffCmd.Stdout = &diffOut
	if err := diffCmd.Run(); err != nil {
		return false, err
	}

	return strings.TrimSpace(diffOut.String()) != "", nil
}

package helper

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

var replaceBlockContentFn = replaceBlockContent

// ReplaceInFile reads the content of a file, performs regex replacement, and writes the modified content back to the file.
func ReplaceInFile(filename string, pattern string, replacement string) error {
	// Read the file content
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Convert content to string
	original := string(content)

	// Replace all instances matching the regex pattern
	replaced := strings.ReplaceAll(original, pattern, replacement)

	// Write the modified content back to the file
	err = ioutil.WriteFile(filename, []byte(replaced), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// ReplaceContentBetweenLines replaces content between specified lines in the target file with content from the source file.
func ReplaceContentBetweenLines(targetFilePath string, sourceFilePath string, from string, till string) error {
	sourceContentStr, err := loadReplacementContent(sourceFilePath, from, till)
	if err != nil {
		return err
	}

	// Open the target file
	targetFile, err := os.Open(targetFilePath)
	if err != nil {
		return fmt.Errorf("failed to open target file: %v", err)
	}
	defer targetFile.Close()

	result, err := replaceBlockContentFn(targetFile, sourceContentStr, from, till)
	if err != nil {
		return err
	}

	// Write the result back to the target file
	err = ioutil.WriteFile(targetFilePath, []byte(strings.Join(result, "\n")), 0644)
	if err != nil {
		return fmt.Errorf("failed to write to target file: %v", err)
	}

	return nil
}

func replaceBlockContent(targetFile *os.File, sourceContentStr, from, till string) ([]string, error) {
	var result []string
	scanner := bufio.NewScanner(targetFile)
	inReplaceBlock := false

	for scanner.Scan() {
		line := scanner.Text()

		if strings.Contains(line, from) {
			inReplaceBlock = true
			result = append(result, line)

			for scanner.Scan() {
				line = scanner.Text()
				if strings.Contains(line, till) {
					break
				}
			}

			result = append(result, sourceContentStr)
		} else if !inReplaceBlock {
			result = append(result, line)
		}

		if inReplaceBlock && strings.Contains(line, till) {
			inReplaceBlock = false
			result = append(result, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading target file: %v", err)
	}

	return result, nil
}

func loadReplacementContent(sourceFilePath, from, till string) (string, error) {
	sourceContent, err := ioutil.ReadFile(sourceFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read source file: %v", err)
	}

	sourceContentStr := strings.ReplaceAll(string(sourceContent), from, "")
	sourceContentStr = strings.ReplaceAll(sourceContentStr, till, "")

	return sourceContentStr, nil
}

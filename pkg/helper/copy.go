package helper

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
)

// copyDirInternal copies files and directories from src to dest, preserving the directory structure.
// If replaceExisting is true, it will overwrite existing files in the destination.
// The filter string specifies files to be included (can be a regex pattern).
func copyDirInternal(src, dest string, replaceExisting bool, filter string) error {
	regexFilter, err := compileCopyFilter(filter)
	if err != nil {
		return err
	}

	err = filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Determine the new path relative to the source directory
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		// Add debug output to verify the files being processed
		// log.Info().Msgf("Processing: %s\n", relPath)

		skip, skipErr := shouldSkipByFilter(info, relPath, regexFilter)
		if skipErr != nil {
			return skipErr
		}
		if skip {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		destPath := getDestinationPath(dest, relPath)
		if err := copyPathEntry(path, destPath, info, replaceExisting); err != nil {
			return err
		}
		return nil
	})
	return err
}

func compileCopyFilter(filter string) (*regexp.Regexp, error) {
	if filter == "" {
		return nil, nil
	}

	return regexp.Compile(filter)
}

func copyPathEntry(sourcePath, destPath string, info os.FileInfo, replaceExisting bool) error {
	if info.IsDir() {
		return os.MkdirAll(destPath, os.ModePerm)
	}

	if _, err := os.Stat(destPath); os.IsNotExist(err) || replaceExisting {
		return CopyFile(sourcePath, destPath, replaceExisting)
	}

	return nil
}

func shouldSkipByFilter(info os.FileInfo, relPath string, regexFilter *regexp.Regexp) (bool, error) {
	if regexFilter == nil {
		return false, nil
	}

	if regexFilter.MatchString(relPath) {
		return false, nil
	}

	return true, nil
}

func getDestinationPath(dest, relPath string) string {
	return ReplaceDotInFilename(filepath.Join(dest, relPath))
}

// replaceDotInFilename replaces DOTREPLACE with a dot (.) in the given filename.
func ReplaceDotInFilename(filename string) string {
	return strings.ReplaceAll(filename, "DOTREPLACE", ".")
}

func CopyDir(src, dest string, replaceExisting bool) error {
	if err := copyDirInternal(src, dest, replaceExisting, ""); err != nil {
		return err
	}
	return nil
}

func CopyDirFiltered(src, dest string, replaceExisting bool, filter string) error {
	if err := copyDirInternal(src, dest, replaceExisting, filter); err != nil {
		return err
	}
	return nil
}

// CopyFile copies a file from source to destination. If replaceExisting is true, it will overwrite existing files in the destination.
func CopyFile(source, destination string, replaceExisting bool) error {
	if !replaceExisting {
		if _, err := os.Stat(destination); err == nil {
			log.Info().Msgf("Skipping existing file: %s\n", destination)
			return nil
		} else if !os.IsNotExist(err) {
			return err
		}
	}

	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	return err
}

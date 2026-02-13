package sops

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/trueforge-org/forgetool/pkg/helper"
)

// EncrFileData holds information about a file and its encryption status.
type EncrFileData struct {
	Path      string
	Encrypted bool
	Staged    bool
}

func ExecuteCheck(useStagedFiles bool) ([]EncrFileData, error) {
	log.Debug().Msg("Starting ExecuteCheck")

	// Step 1: Load the SOPS configuration.
	config, err := sopsLoadSopsConfigFn()
	if err != nil {
		log.Error().Err(err).Msg("Failed to load SOPS config")
		return nil, err
	}
	log.Trace().Msg("SOPS configuration loaded successfully")

	// Step 2: Get the files from .sops.yaml configuration.
	allFiles, err := sopsFilesToCheckFn(config)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get files to check")
		return nil, err
	}
	log.Debug().Msgf("Files to check: %v", allFiles)

	filesToCheck, err := sopsSelectFilesForCheckFn(allFiles, useStagedFiles)
	if err != nil {
		return nil, err
	}

	// Step 5: Check the encryption status of each file.
	for i, file := range filesToCheck {
		data, err := sopsReadFileFn(file.Path)
		if err != nil {
			log.Error().Err(err).Msgf("Error reading file %s", file.Path)
			return nil, fmt.Errorf("error reading file %s: %v", file.Path, err)
		}
		log.Trace().Msgf("Read file %s successfully", file.Path)

		// Check if the file is encrypted based on the criteria defined in .sops.yaml.
		filesToCheck[i].Encrypted = sopsIsEncryptedFn(data, file.Path)
		log.Debug().Msgf("File %s encrypted status: %v", file.Path, filesToCheck[i].Encrypted)
	}

	return filesToCheck, nil
}

func CheckFilesAndReportEncryption(tryEncrypt bool, checkStaged bool) error {
	log.Debug().Msg("Starting CheckFilesAndReportEncryption")

	// Step 1: Run the encryption check based on the toggle.
	files, err := sopsExecuteCheckFn(checkStaged)
	if err != nil {
		log.Error().Err(err).Msg("Error executing check")
		return fmt.Errorf("error executing check: %v", err)
	}

	unencryptedFiles := filterUnencryptedFiles(files)

	// Step 3: If there are any unencrypted files, handle based on the tryEncrypt flag.
	if len(unencryptedFiles) > 0 {
		sopsHandleUnencryptedFilesFn(unencryptedFiles, tryEncrypt)
	}

	// Step 6: If no unencrypted files are found, print a success message and exit with code 0.
	log.Info().Msg("All files are encrypted.")
	fmt.Println("All files are encrypted.")
	sopsExitFn(0)

	return nil
}

func selectFilesForCheck(allFiles []EncrFileData, useStagedFiles bool) ([]EncrFileData, error) {
	if !useStagedFiles {
		log.Debug().Msgf("Using all files: %v", allFiles)
		return allFiles, nil
	}

	stagedFiles, err := sopsGetStagedFilesFn()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get staged files")
		return nil, err
	}

	if len(stagedFiles) == 0 {
		log.Warn().Msg("No staged files to check")
		return nil, fmt.Errorf("no staged files to check")
	}
	log.Info().Msgf("Staged files: %v", stagedFiles)

	filtered := filterStagedSopsFiles(allFiles, stagedFiles)
	if err = sopsStageFilteredFilesFn(filtered); err != nil {
		return nil, err
	}

	return filtered, nil
}

func filterStagedSopsFiles(allFiles []EncrFileData, stagedFiles []string) []EncrFileData {
	var filesToCheck []EncrFileData
	for _, file := range allFiles {
		checkPath := file.Path
		if _, err := sopsStatFn("./DEVTRIGGER"); err == nil {
			checkPath = filepath.Join("forgetool", checkPath)
		}
		for _, stagedFile := range stagedFiles {
			if checkPath == stagedFile {
				filesToCheck = append(filesToCheck, file)
				break
			}
		}
	}
	return filesToCheck
}

func stageFilteredFiles(files []EncrFileData) error {
	var filePaths []string
	for _, file := range files {
		filePaths = append(filePaths, file.Path)
	}

	if err := sopsStageFilesFn(filePaths); err != nil {
		log.Error().Err(err).Msg("Error staging files")
		return fmt.Errorf("error staging files: %v", err)
	}
	log.Info().Msg("All staged files processed successfully")
	return nil
}

func filterUnencryptedFiles(files []EncrFileData) []EncrFileData {
	var unencryptedFiles []EncrFileData
	for _, file := range files {
		if !file.Encrypted {
			unencryptedFiles = append(unencryptedFiles, file)
		}
	}
	return unencryptedFiles
}

func handleUnencryptedFiles(unencryptedFiles []EncrFileData, tryEncrypt bool) {
	log.Warn().Msg("Found unencrypted files")
	fmt.Println("The following files are not encrypted:")

	for _, file := range unencryptedFiles {
		fmt.Println(file.Path)
		if tryEncrypt {
			sopsTryEncryptAndStageFileFn(file)
		}
	}

	if !tryEncrypt {
		log.Debug().Msg("tryEncrypt is false")
		sopsFatalFn("Exiting due to unencrypted files")
		sopsExitFn(1)
	}

	stillUnencrypted := sopsFindStillUnencryptedFn(unencryptedFiles)
	if len(stillUnencrypted) > 0 {
		log.Warn().Msg("The following files could not be encrypted:")
		fmt.Println("The following files could not be encrypted:")
		for _, file := range stillUnencrypted {
			fmt.Println(file)
		}
		sopsFatalFn("Exiting due to unencrypted files after encryption attempt")
		sopsExitFn(1)
	}

	shamCheck()
}

func tryEncryptAndStageFile(file EncrFileData) {
	err := sopsProcessFileEncryptionFn(file)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to encrypt file %s", file.Path)
		return
	}

	log.Info().Msgf("File %s encrypted successfully.", file.Path)
	if err = sopsStageFileFn(file.Path); err != nil {
		log.Error().Err(err).Msgf("Failed to stage file %s after encryption", file.Path)
		return
	}

	log.Info().Msgf("File %s staged successfully after encryption.", file.Path)
}

func findStillUnencrypted(files []EncrFileData) []string {
	var stillUnencrypted []string
	for _, file := range files {
		data, err := sopsReadFileFn(file.Path)
		if err != nil {
			log.Error().Err(err).Msgf("Error reading file %s", file.Path)
			stillUnencrypted = append(stillUnencrypted, file.Path)
			continue
		}
		if !sopsIsEncryptedFn(data, file.Path) {
			stillUnencrypted = append(stillUnencrypted, file.Path)
		}
	}
	return stillUnencrypted
}

// shamCheck checks if clusterenv contains the phrase "shamir_threshold" indicating it is indeed encrypted
func shamCheck() {
	log.Debug().Msg("Checking if clusterenv contains shamir_threshold to ensure encryption...")
	file, err := sopsOpenFn(helper.ClusterEnvFile)
	if err != nil {
		log.Error().Err(err).Msgf("error opening file: %v", err)
		sopsExitFn(1)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "shamir_threshold") {
			log.Debug().Msg("clusterenv contains shamir_threshold, continuing...")
			return
		}
	}

	if err := sopsScannerErrFn(scanner); err != nil {
		log.Error().Err(err).Msgf("error reading file: %v", err)
	}

	log.Error().Msg("clusterenv is NOT encrypted and encryption-check failed!\n DO NOT UPLOAD!")
	sopsExitFn(1)
}

// filesToCheck returns a list of files to check for encryption based on the logic in .sops.yaml.
func filesToCheck(config SopsConfig) ([]EncrFileData, error) {
	log.Debug().Msg("Starting filesToCheck")

	var files []EncrFileData
	for _, rule := range config.CreationRules {
		pathRegex, err := regexp.Compile(rule.PathRegex)
		if err != nil {
			log.Error().Err(err).Msg("Invalid path regex in .sops.yaml")
			return nil, fmt.Errorf("invalid path regex in .sops.yaml: %v", err)
		}

		err = sopsWalkRuleFilesFn(pathRegex, &files)
		if err != nil {
			log.Error().Err(err).Msg("Error walking file paths")
			return nil, err
		}
	}

	return files, nil
}

func walkRuleFiles(pathRegex *regexp.Regexp, files *[]EncrFileData) error {
	return sopsFilepathWalkFn(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && pathRegex.MatchString(path) {
			*files = append(*files, EncrFileData{Path: path, Encrypted: false})
			log.Debug().Msgf("Matched file: %s", path)
		}
		return nil
	})
}

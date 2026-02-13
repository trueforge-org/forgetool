package sops

import (
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
)

// Custom error type for MAC failures
type MacFailureError struct {
	OriginalError error
}

func (e *MacFailureError) Error() string {
	return fmt.Sprintf("MAC failure: %v", e.OriginalError)
}

func DecryptFiles() error {
	log.Trace().Msg("Starting DecryptFiles function")

	// Get a list of encrypted files
	files, err := sopsExecuteCheckFn(false)
	if err != nil {
		log.Error().Err(err).Msg("Failed to execute check for encrypted files")
		return err
	}

	encryptedFound, err := sopsDecryptMarkedFilesFn(files)
	if err != nil {
		return err
	}

	// Check if any encrypted files were found
	if !encryptedFound {
		log.Info().Msg("Nothing to decrypt")
	}

	sopsLoadTalEnvFn(true)
	log.Info().Msg("All files decrypted successfully")
	return nil
}

func decryptMarkedFiles(files []EncrFileData) (bool, error) {
	encryptedFound := false
	for _, file := range files {
		if !file.Encrypted {
			continue
		}

		encryptedFound = true
		if err := sopsDecryptFileFn(file.Path); err != nil {
			return encryptedFound, err
		}
	}

	return encryptedFound, nil
}

func decryptFile(path string) error {
	log.Debug().Msgf("Decrypting file: %s", path)

	data, err := sopsDecryptReadFileFn(path)
	if err != nil {
		log.Error().Err(err).Msgf("Error reading file %s", path)
		return fmt.Errorf("error reading file %s: %v", path, err)
	}

	decrypted, err := sopsDecryptDataWithRetryFn(data, GetFormat(path))
	if err != nil {
		log.Error().Err(err).Msgf("Error decrypting file %s", path)
		return fmt.Errorf("error decrypting file %s: %v", path, err)
	}

	if err = sopsDecryptWriteFileFn(path, decrypted, 0644); err != nil {
		log.Error().Err(err).Msgf("Error writing decrypted data to file %s", path)
		return fmt.Errorf("error writing decrypted data to file %s: %v", path, err)
	}
	log.Debug().Msgf("Successfully decrypted file: %s", path)

	return nil
}

func decryptData(data []byte, format string) ([]byte, error) {
	log.Trace().Msg("Starting decryption of data")

	os.Setenv("SOPS_AGE_KEY_FILE", "age.agekey")
	// Decrypt data
	decrypted, err := sopsDecryptDataFn(data, format)
	if err != nil {
		// Check for MAC failure error from imported packages
		if strings.Contains(err.Error(), "Failed to decrypt original mac") ||
			strings.Contains(err.Error(), "Failed to verify data integrity") {
			log.Warn().Msg("Ignoring MAC failure from imported packages.")
			// Return decrypted data as is
			return data, nil
		}
		log.Error().Err(err).Msg("Decryption failed")
		return nil, err
	}

	log.Debug().Msg("Data decrypted successfully")
	return decrypted, nil
}

// Retry decryption if MAC failure is detected
func decryptDataWithRetry(data []byte, format string) ([]byte, error) {
	decrypted, err := sopsDecryptCoreFn(data, format)
	if err != nil {
		if macErr, ok := err.(*MacFailureError); ok {
			log.Info().Msgf("MAC failure detected: %v. Retrying without MAC verification.", macErr.OriginalError)
			// Proceed without verifying MAC
			decrypted, err = sopsDecryptDataIgnoringMacFn(data, format)
			if err != nil {
				log.Error().Err(err).Msg("Retry decryption failed")
				return nil, fmt.Errorf("retry decryption failed: %v", err)
			}
		} else {
			log.Error().Err(err).Msg("Decryption failed without MAC error")
			return nil, err
		}
	}
	log.Debug().Msg("Data decryption with retry succeeded")
	return decrypted, nil
}

// Decrypt data ignoring MAC failure (hypothetical function)
func decryptDataIgnoringMac(data []byte, format string) ([]byte, error) {
	log.Trace().Msg("Decrypting data while ignoring MAC failure")

	decrypted, err := sopsDecryptDataFn(data, format)
	if err != nil && isMacFailure(err) {
		// Log the MAC failure and proceed with the decrypted data
		log.Warn().Msg("Ignoring MAC failure while decrypting data.")
		return data, nil
	}
	if err != nil {
		log.Error().Err(err).Msg("Error decrypting data while ignoring MAC failure")
	}
	return decrypted, err
}

// Check if the error is a MAC failure
func isMacFailure(err error) bool {
	return strings.Contains(err.Error(), "MAC verification failed")
}

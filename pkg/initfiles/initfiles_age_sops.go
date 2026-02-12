package initfiles

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	age "filippo.io/age"
	"github.com/rs/zerolog/log"
	"github.com/trueforge-org/forgetool/pkg/helper"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
)

func ageGen() error {
	outFlag := "age.agekey"

	if _, err := os.Stat(outFlag); err == nil {

	} else if errors.Is(err, os.ErrNotExist) {
		out := os.Stdout
		f, err := os.OpenFile(outFlag, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to open output file %q: %v")
		}
		defer func() {
			if err := f.Close(); err != nil {
				log.Fatal().Err(err).Msg("failed to close output file %q: %v")
			}
		}()
		out = f
		if fi, err := out.Stat(); err == nil && fi.Mode().IsRegular() && fi.Mode().Perm()&0004 != 0 {
			log.Info().Msgf("writing secret key to a world-readable file\n")
		}

		k, err := age.GenerateX25519Identity()
		if err != nil {
			log.Fatal().Err(err).Msg("internal error: %v")
		}

		fmt.Fprintf(out, "# created: %s\n", time.Now().Format(time.RFC3339))
		fmt.Fprintf(out, "# public key: %s\n", k.Recipient())
		fmt.Fprintf(out, "%s\n", k)

	} else {

	}

	return nil
}

func GetPubKey() (string, error) {
	// Open the file
	filename := "age.agekey"
	file, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var publicKey string

	// Read the file line by line
	for scanner.Scan() {
		line := scanner.Text()
		// Find the line with the public key
		if strings.HasPrefix(line, "# public key:") {
			parts := strings.Split(line, ": ")
			if len(parts) == 2 {
				publicKey = parts[1]
			}
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to scan file: %v", err)
	}

	if publicKey == "" {
		return "", fmt.Errorf("public key not found")
	}

	return publicKey, nil
}

// getSecretKeyFromFile reads the specified file and returns the secret key found within it.
func GetSecKey() (string, error) {
	// Open the file
	filename := "age.agekey"
	file, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var secretKey string

	// Read the file line by line
	for scanner.Scan() {
		line := scanner.Text()
		// Find the line that contains the secret key prefix
		if strings.HasPrefix(line, "AGE-SECRET-KEY-") {
			secretKey = line
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to scan file: %v", err)
	}

	if secretKey == "" {
		return "", fmt.Errorf("secret key not found")
	}

	return secretKey, nil
}

func GenSopsSecret() error {
	secretPath := filepath.Join(helper.ClusterPath, "kubernetes", "flux-system", "flux", "sopssecret.secret.yaml")
	ageSecKey, err := GetSecKey()

	// Added by Boemeltrein, for linting purposes
	if err != nil {
		return fmt.Errorf("failed to get age secret key: %w", err)
	}

	// Generate Kubernetes secret YAML content
	secret := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata": map[string]interface{}{
			"name":      "sops-age",
			"namespace": "flux-system",
		},
		"stringData": map[string]interface{}{
			"age.agekey": ageSecKey,
		},
		"type": string(corev1.SecretTypeOpaque),
	}

	secretYAML, err := yaml.Marshal(secret)
	if err != nil {
		return fmt.Errorf("failed to marshal secret to YAML: %w", err)
	}

	// Write Kubernetes secret YAML to file
	err = os.MkdirAll(filepath.Dir(secretPath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}
	err = os.WriteFile(secretPath, secretYAML, 0644)
	if err != nil {
		return fmt.Errorf("failed to write secret YAML to file: %w", err)
	}
	log.Info().Msgf("SOPS secret YAML saved to: %s\n", secretPath)
	return nil
}

package sops

import (
	"bytes"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// isEncrypted checks if the given data is encrypted based on the criteria defined in .sops.yaml.
func isEncrypted(data []byte, filePath string) bool {
	log.Trace().Msgf("Checking if file %s is encrypted", filePath)
	// Detect the file format based on the file extension
	switch filepath.Ext(filePath) {
	case ".yaml", ".yml":
		return containsSopsField(data)
	case ".json":
		return containsSopsField(data)
	case ".env", ".ini":
		return containsEncMarker(data)
	default:
		return false
	}
}

func GetFormat(filePath string) string {
	log.Trace().Msgf("Getting format for file %s", filePath)
	switch filepath.Ext(filePath) {
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	case ".env":
		return "dotenv"
	case ".ini":
		return "ini"
	default:
		return "binary"
	}
}

// containsSopsField checks if the data contains the SOPS field.
func containsSopsField(data []byte) bool {
	log.Trace().Msg("Checking for SOPS field in data")
	var content map[string]interface{}
	if err := yaml.Unmarshal(data, &content); err != nil {
		// If the YAML is invalid, consider it not encrypted.
		return false
	}
	_, ok := content["sops"]
	return ok
}

// containsEncMarker checks if the data contains an encryption marker.
func containsEncMarker(data []byte) bool {
	log.Trace().Msg("Checking for encryption marker in data")
	return bytes.Contains(data, []byte("ENC["))
}

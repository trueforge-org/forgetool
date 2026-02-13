package sops

import (
	"errors"
	"fmt"

	"github.com/getsops/sops/v3"
	"github.com/getsops/sops/v3/aes"
	"github.com/getsops/sops/v3/age"
	"github.com/getsops/sops/v3/cmd/sops/common"
	"github.com/getsops/sops/v3/keys"
	"github.com/getsops/sops/v3/version"
	"github.com/rs/zerolog/log"
	"github.com/trueforge-org/forgetool/pkg/helper"
)

//nolint:unused
var encrConfig *EncryptionConfig

//nolint:unused
const ageKeyFilePath = "./age.agekey"

func EncryptWithAgeKey(body []byte, regex string, format string) ([]byte, error) {
	log.Trace().Msg("Starting EncryptWithAgeKey function")

	// Create a cypher instance
	cypher := sopsNewCypherFn()
	log.Debug().Msg("Cypher instance created")

	sopsConfig, err := sopsLoadSopsConfigFn()
	if err != nil {
		log.Error().Err(err).Msg("Failed to load Sops config")
		return nil, err
	}
	log.Debug().Msg("Successfully loaded Sops config")

	ageKeys := sopsCollectAgeKeysFn(sopsConfig)
	log.Debug().Strs("ageKeys", ageKeys).Msg("Collected age keys from creation rules")

	groups := sopsBuildAgeKeyGroupsFn(ageKeys)

	log.Debug().Msg("Key groups created for encryption")

	// Encrypt the data using the sops key
	encryptedData, err := cypher.Encrypt(body, EncryptionConfig{
		Keys:              groups,
		UnencryptedSuffix: "",
		EncryptedSuffix:   "",
		UnencryptedRegex:  "",
		EncryptedRegex:    regex,
		ShamirThreshold:   3,
		Format:            format,
	})
	if err != nil {
		log.Error().Err(err).Msg("Error encrypting data")
		return nil, fmt.Errorf("error encrypting data: %v", err)
	}

	log.Debug().Msg("Data encrypted successfully")
	return encryptedData, nil
}

func collectAgeKeys(sopsConfig SopsConfig) []string {
	var ageKeys []string
	for _, rule := range sopsConfig.CreationRules {
		ageKeys = append(ageKeys, rule.Age)
	}

	return ageKeys
}

func buildAgeKeyGroups(ageKeys []string) []sops.KeyGroup {
	var groups []sops.KeyGroup
	for _, ageKey := range helper.UniqueNonEmptyElementsOf(ageKeys) {
		var keyGroup sops.KeyGroup
		keyGroup = append(keyGroup, NewMasterKey(ageKey))
		groups = append(groups, keyGroup)
	}

	return groups
}

/// Custom keygroup

func NewMasterKey(pubkey string) (result keys.MasterKey) {
	log.Trace().Str("pubkey", pubkey).Msg("Creating new master key")

	result, err := age.MasterKeyFromRecipient(pubkey)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create master key from recipient")
	}

	return result
}

/// IMPORTED

const (
	formatYaml = "yaml"
	formatJson = "json"
)

type Cypher interface {
	Decrypt(content []byte, config string) ([]byte, error)
	Encrypt(data []byte, config EncryptionConfig) ([]byte, error)
}

type cypher struct{}

func NewCypher() Cypher {
	log.Debug().Msg("Creating new cypher instance")
	return &cypher{}
}

func (c *cypher) Decrypt(content []byte, format string) ([]byte, error) {
	log.Trace().Msg("Decrypting content")
	decryptedData, err := sopsDecryptDataFn(content, format)
	if err != nil {
		log.Error().Err(err).Msg("Error during decryption")
		return nil, err
	}
	log.Info().Msg("Content decrypted successfully")
	return decryptedData, nil
}

type EncryptionConfig struct {
	Format            string
	Keys              []sops.KeyGroup
	UnencryptedSuffix string
	EncryptedSuffix   string
	UnencryptedRegex  string
	EncryptedRegex    string
	ShamirThreshold   int
}

func (m *cypher) Encrypt(content []byte, encrConfig EncryptionConfig) (result []byte, err error) {
	log.Trace().Msg("Starting encryption process")

	store := sopsStoreForFormatFn(encrConfig.Format)

	log.Debug().Msg("Store initialized for encryption")

	branches, err := sopsLoadPlainFileFn(store, content)
	if err != nil {
		log.Error().Err(err).Msg("Failed to load plain file for encryption")
		return nil, err
	}
	log.Debug().Msg("Plain file loaded successfully")

	tree := sops.Tree{
		Branches: branches,
		Metadata: sops.Metadata{
			KeyGroups:         encrConfig.Keys,
			UnencryptedSuffix: encrConfig.UnencryptedSuffix,
			EncryptedSuffix:   encrConfig.EncryptedSuffix,
			UnencryptedRegex:  encrConfig.UnencryptedRegex,
			EncryptedRegex:    encrConfig.EncryptedRegex,
			Version:           version.Version,
			ShamirThreshold:   encrConfig.ShamirThreshold,
		},
	}

	dataKey, errs := sopsGenerateDataKeyFn(&tree)

	if len(errs) > 0 {
		log.Error().Err(err).Msg("Could not generate data key")
		return nil, errors.New(fmt.Sprint("Could not generate data key:", errs))
	}

	log.Debug().Msg("Data key generated successfully")

	encryptTreeOpts := common.EncryptTreeOpts{
		DataKey: dataKey,
		Tree:    &tree,
		Cipher:  aes.NewCipher(),
	}

	err = sopsEncryptTreeFn(encryptTreeOpts)
	if err != nil {
		log.Error().Err(err).Msg("Error during tree encryption")
		return nil, err
	}

	log.Debug().Msg("Tree encrypted successfully")
	return sopsEmitEncryptedFileFn(store, tree)
}

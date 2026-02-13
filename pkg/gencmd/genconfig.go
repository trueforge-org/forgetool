package gencmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/rs/zerolog/log"

	"github.com/trueforge-org/forgetool/pkg/fluxhandler"
	"github.com/trueforge-org/forgetool/pkg/helper"
	"github.com/trueforge-org/forgetool/pkg/initfiles"
	"github.com/trueforge-org/forgetool/pkg/talassist"
)

func defaultGenConfigFatalExit() {
	log.Info().Msg("You need to re-run Init. Exiting...")
}

func defaultEncodeSecretBundle(secretbundle any) (data []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("yaml encode panic: %v", r)
			data = nil
		}
	}()

	buf := new(bytes.Buffer)
	encoder := helper.YamlNewEncoder(buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(secretbundle); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func defaultWriteTalSecretBytes(outfile *os.File, data []byte) (int, error) {
	return outfile.Write(data)
}

var (
	checkRunAgainFileExistsFn = initfiles.CheckRunAgainFileExists
	loadTalConfigFn           = talassist.LoadTalConfig
	genSchemaFn               = talassist.GenSchema
	genTalEnvConfigMapFn      = initfiles.GenTalEnvConfigMap
	checkEnvVariablesFn       = initfiles.CheckEnvVariables
	genTalSecretFn            = genTalSecret
	talhelperGenConfigFn      = talassist.TalhelperGenConfig
	updateGitRepoFn           = initfiles.UpdateGitRepo
	processDirectoryFn        = fluxhandler.ProcessDirectory
	createEncrPreCommitHookFn = helper.CreateEncrPreCommitHook
	genConfigFatalExitFn      = defaultGenConfigFatalExit
	createTalSecretFileFn     = os.Create
	encodeSecretBundleFn      = defaultEncodeSecretBundle
	writeTalSecretBytesFn     = defaultWriteTalSecretBytes
)

func GenConfig(args []string) error {
	if checkRunAgainFileExistsFn() {
		genConfigFatalExitFn()
		osExitFn(1)
	}
	if err := sopsDecryptFilesFn(); err != nil {
		log.Info().Msgf("Error decrypting files: %v\n", err)
	}
	loadTalConfigFn()
	genSchemaFn()
	genTalEnvConfigMapFn()
	checkEnvVariablesFn()
	genTalSecretFn()
	talhelperGenConfigFn()
	updateGitRepoFn()

	if err := processDirectoryFn(path.Join(helper.ClusterPath, "kubernetes")); err != nil {
		log.Info().Msgf("Error: %v", err)
	}
	if err := processDirectoryFn(path.Join(helper.ClusterPath, "kubernetes")); err != nil {
		log.Info().Msgf("Error: %v", err)
	} else {
		log.Info().Msgf("Kustomizations processed successfully.")
	}
	createEncrPreCommitHookFn()
	log.Info().Msg("GenConfig: Completed Successfully!")
	return nil
}

func genTalSecret() error {
	log.Info().Msg("Running TalSecret check-and-create...")
	if _, err := os.Stat(helper.TalSecretFile); err == nil {
		log.Debug().Msg("TalSecret already exists, skipping...")
	} else if errors.Is(err, os.ErrNotExist) {
		log.Info().Msg("Generating TalSecret...")
		os.MkdirAll(helper.TalosGenerated, os.ModePerm)
		outfile, err := createTalSecretFileFn(helper.TalSecretFile)
		if err != nil {
			panic(err)
		}
		defer outfile.Close()

		secretbundle := talassist.NewSecretBundle()
		data, err := encodeSecretBundleFn(secretbundle)

		if err != nil {
			return err
		}

		_, err = writeTalSecretBytesFn(outfile, data)
		if err != nil {
			panic(err)
		}

		return nil
	}
	return nil
}

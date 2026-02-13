package sops

import (
	"bufio"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/getsops/sops/v3"
	"github.com/getsops/sops/v3/aes"
	"github.com/getsops/sops/v3/cmd/sops/common"
	"github.com/getsops/sops/v3/cmd/sops/formats"
	"github.com/getsops/sops/v3/config"
	"github.com/getsops/sops/v3/decrypt"
	"github.com/getsops/sops/v3/keyservice"
	"github.com/rs/zerolog/log"
	"github.com/trueforge-org/forgetool/pkg/helper"
	"github.com/trueforge-org/forgetool/pkg/initfiles"
)

func defaultSopsFatal(msg string) { log.Error().Msg(msg) }

func defaultScannerErr(scanner *bufio.Scanner) error { return scanner.Err() }

func defaultStoreForFormat(format string) common.Store {
	if format == formatYaml {
		return common.StoreForFormat(formats.Yaml, config.NewStoresConfig())
	}
	return common.StoreForFormat(formats.Json, config.NewStoresConfig())
}

func defaultLoadPlainFile(store common.Store, content []byte) (sops.TreeBranches, error) {
	return store.LoadPlainFile(content)
}

func defaultGenerateDataKey(tree *sops.Tree) ([]byte, []error) {
	return tree.GenerateDataKeyWithKeyServices([]keyservice.KeyServiceClient{keyservice.NewLocalClient()})
}

func defaultEncryptTree(opts common.EncryptTreeOpts) error { return common.EncryptTree(opts) }

func defaultEmitEncryptedFile(store common.Store, tree sops.Tree) ([]byte, error) {
	return store.EmitEncryptedFile(tree)
}

var (
	sopsLoadSopsConfigFn         = LoadSopsConfig
	sopsFilesToCheckFn           = filesToCheck
	sopsSelectFilesForCheckFn    = selectFilesForCheck
	sopsReadFileFn               = os.ReadFile
	sopsIsEncryptedFn            = isEncrypted
	sopsGetStagedFilesFn         = helper.GetStagedFiles
	sopsStageFilteredFilesFn     = stageFilteredFiles
	sopsStatFn                   = os.Stat
	sopsStageFilesFn             = helper.StageFiles
	sopsProcessFileEncryptionFn  = processFileEncryption
	sopsStageFileFn              = helper.StageFile
	sopsFindStillUnencryptedFn   = findStillUnencrypted
	sopsTryEncryptAndStageFileFn = tryEncryptAndStageFile
	sopsHandleUnencryptedFilesFn = handleUnencryptedFiles
	sopsExecuteCheckFn           = ExecuteCheck
	sopsExitFn                   = os.Exit
	sopsFatalFn                  = defaultSopsFatal
	sopsOpenFn                   = os.Open
	sopsScannerErrFn             = defaultScannerErr
	sopsWalkRuleFilesFn          = walkRuleFiles
	sopsFilepathWalkFn           = filepath.Walk

	sopsDecryptMarkedFilesFn     = decryptMarkedFiles
	sopsDecryptFileFn            = decryptFile
	sopsLoadTalEnvFn             = initfiles.LoadTalEnv
	sopsDecryptDataFn            = decrypt.Data
	sopsDecryptCoreFn            = decryptData
	sopsDecryptDataWithRetryFn   = decryptDataWithRetry
	sopsDecryptDataIgnoringMacFn = decryptDataIgnoringMac
	sopsDecryptReadFileFn        = os.ReadFile
	sopsDecryptWriteFileFn       = os.WriteFile

	sopsIsFileFullyStagedFn = helper.IsFileFullyStaged
	sopsEncryptFileFn       = encryptFile
	sopsEncryptReadFileFn   = os.ReadFile
	sopsEncryptWriteFileFn  = os.WriteFile
	sopsMergeRegexFn        = mergeRegex
	sopsEncryptWithAgeKeyFn = EncryptWithAgeKey

	sopsLoadConfigStatFn     = os.Stat
	sopsLoadConfigReadFileFn = ioutil.ReadFile

	sopsNewCypherFn         = NewCypher
	sopsCollectAgeKeysFn    = collectAgeKeys
	sopsBuildAgeKeyGroupsFn = buildAgeKeyGroups
	sopsStoreForFormatFn    = defaultStoreForFormat
	sopsLoadPlainFileFn     = defaultLoadPlainFile
	sopsGenerateDataKeyFn   = defaultGenerateDataKey
	sopsEncryptTreeFn       = defaultEncryptTree
	sopsEmitEncryptedFileFn = defaultEmitEncryptedFile

	sopsNewCipherFn = aes.NewCipher
)

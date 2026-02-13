package sops

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/getsops/sops/v3/decrypt"
	"github.com/trueforge-org/forgetool/pkg/helper"
	"github.com/trueforge-org/forgetool/pkg/initfiles"
)

type exitPanic struct{}

func expectExitPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected exit panic")
		}
	}()
	fn()
}

func setDefaultSopsHooks() {

	sopsLoadSopsConfigFn = LoadSopsConfig
	sopsFilesToCheckFn = filesToCheck
	sopsSelectFilesForCheckFn = selectFilesForCheck
	sopsReadFileFn = os.ReadFile
	sopsIsEncryptedFn = isEncrypted
	sopsGetStagedFilesFn = helper.GetStagedFiles
	sopsStageFilteredFilesFn = stageFilteredFiles
	sopsStatFn = os.Stat
	sopsStageFilesFn = helper.StageFiles
	sopsProcessFileEncryptionFn = processFileEncryption
	sopsStageFileFn = helper.StageFile
	sopsFindStillUnencryptedFn = findStillUnencrypted
	sopsTryEncryptAndStageFileFn = tryEncryptAndStageFile
	sopsHandleUnencryptedFilesFn = handleUnencryptedFiles
	sopsExecuteCheckFn = ExecuteCheck
	sopsExitFn = os.Exit
	sopsFatalFn = defaultSopsFatal
	sopsOpenFn = os.Open
	sopsScannerErrFn = defaultScannerErr
	sopsWalkRuleFilesFn = walkRuleFiles
	sopsFilepathWalkFn = filepath.Walk

	sopsDecryptMarkedFilesFn = decryptMarkedFiles
	sopsDecryptFileFn = decryptFile
	sopsLoadTalEnvFn = initfiles.LoadTalEnv
	sopsDecryptDataFn = decrypt.Data
	sopsDecryptCoreFn = decryptData
	sopsDecryptDataWithRetryFn = decryptDataWithRetry
	sopsDecryptDataIgnoringMacFn = decryptDataIgnoringMac
	sopsDecryptReadFileFn = os.ReadFile
	sopsDecryptWriteFileFn = os.WriteFile

	sopsIsFileFullyStagedFn = helper.IsFileFullyStaged
	sopsEncryptFileFn = encryptFile
	sopsEncryptReadFileFn = os.ReadFile
	sopsEncryptWriteFileFn = os.WriteFile
	sopsMergeRegexFn = mergeRegex
	sopsEncryptWithAgeKeyFn = EncryptWithAgeKey

	sopsLoadConfigStatFn = os.Stat
	sopsLoadConfigReadFileFn = ioutil.ReadFile

	sopsNewCypherFn = NewCypher
	sopsCollectAgeKeysFn = collectAgeKeys
	sopsBuildAgeKeyGroupsFn = buildAgeKeyGroups
	sopsStoreForFormatFn = defaultStoreForFormat
	sopsLoadPlainFileFn = defaultLoadPlainFile
	sopsGenerateDataKeyFn = defaultGenerateDataKey
	sopsEncryptTreeFn = defaultEncryptTree
	sopsEmitEncryptedFileFn = defaultEmitEncryptedFile
}

func resetSopsHooks(t *testing.T) {
	t.Helper()
	setDefaultSopsHooks()
	t.Cleanup(func() {
		setDefaultSopsHooks()
	})
}

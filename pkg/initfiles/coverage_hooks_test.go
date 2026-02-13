package initfiles

import (
	"os"

	"github.com/trueforge-org/forgetool/pkg/fluxhandler"
	"github.com/trueforge-org/forgetool/pkg/helper"
	"github.com/trueforge-org/forgetool/pkg/talassist"
)

var (
	defaultClusterenvFatalMsgFn      = clusterenvFatalMsgFn
	defaultInitfilesWriteContentFn   = initfilesWriteFileContentFn
	defaultInitfilesFatalErrMsgfFn   = initfilesFatalErrMsgfFn
	defaultInitfilesFatalErrMsgFn    = initfilesFatalErrMsgFn
	defaultAgeSopsGenerateIdentityFn = ageSopsGenerateIdentityFn
	defaultAgeSopsNowFn              = ageSopsNowFn
	defaultAgeSopsYAMLMarshalFn      = ageSopsYAMLMarshalFn
	defaultAgeSopsCloseFileFn        = ageSopsCloseFileFn
	defaultAgeSopsExitFn             = ageSopsExitFn
	defaultAgeSopsFatalErrMsgFn      = ageSopsFatalErrMsgFn
)

func resetInitfilesCoverageHooks() {
	clusterenvStatFn = os.Stat
	clusterenvLoadEnvFromFileFn = helper.LoadEnvFromFile
	clusterenvExitFn = os.Exit
	clusterenvOpenFn = os.Open
	clusterenvSetenvFn = os.Setenv
	clusterenvIPInRangeFn = helper.IPInRange
	clusterenvLoadTalEnvFn = LoadTalEnv
	clusterenvValidateIPorCIDRNotInCIDRFn = helper.ValidateIPorCIDRNotInCIDR
	clusterenvValidateRangeNotInCIDRFn = helper.ValidateRangeNotInCIDR
	clusterenvFatalMsgFn = defaultClusterenvFatalMsgFn

	initfilesRemoveRunAgainFileFn = removeRunAgainFile
	initfilesAgeGenFn = ageGen
	initfilesGenRootFilesFn = genRootFiles
	initfilesGenBaseFilesFn = genBaseFiles
	initfilesUpdateRootFilesFn = UpdateRootFiles
	initfilesUpdateBaseFilesFn = UpdateBaseFiles
	initfilesGenSchemaFn = talassist.GenSchema
	initfilesGenPatchesFn = GenPatches
	initfilesGenKubernetesFn = genKubernetes
	initfilesGenTalEnvConfigMapFn = GenTalEnvConfigMap
	initfilesUpdateGitRepoFn = UpdateGitRepo
	initfilesCreateGitSecretFn = fluxhandler.CreateGitSecret
	initfilesGenSopsSecretFn = GenSopsSecret
	initfilesProcessKustomizationsFn = processKustomizations
	initfilesCreateEncrPreCommitHookFn = helper.CreateEncrPreCommitHook
	initfilesProcessDirectoryFn = fluxhandler.ProcessDirectory
	initfilesCopyDirFn = helper.CopyDir
	initfilesReplaceInFileFn = helper.ReplaceInFile
	initfilesReadFileFn = os.ReadFile
	initfilesMkdirAllFn = os.MkdirAll
	initfilesCopyFileFn = helper.CopyFile
	initfilesStatFn = os.Stat
	initfilesIsNotExistFn = os.IsNotExist
	initfilesCreateRunAgainFileFn = createRunAgainFile
	initfilesExitFn = os.Exit
	initfilesReadFilenamesInDirFn = readFilenamesInDir
	initfilesReplaceContentBetweenLinesFn = helper.ReplaceContentBetweenLines
	initfilesCheckEnvVariablesFn = CheckEnvVariables
	initfilesGetPubKeyFn = GetPubKey
	initfilesLoadTalEnvFn = LoadTalEnv
	initfilesCopyDirFilteredFn = helper.CopyDirFiltered
	initfilesEnvSubstRecursiveFn = helper.EnvSubstRecursive
	initfilesGetSecKeyFn = GetSecKey
	initfilesSetDockerFn = setDocker
	initfilesAppendContentToPatchFileFn = appendContentToPatchFile
	initfilesOpenFileFn = os.OpenFile
	initfilesWriteFileContentFn = defaultInitfilesWriteContentFn
	initfilesFatalErrMsgfFn = defaultInitfilesFatalErrMsgfFn
	initfilesFatalErrMsgFn = defaultInitfilesFatalErrMsgFn

	ageSopsStatFn = os.Stat
	ageSopsOpenFileFn = os.OpenFile
	ageSopsGenerateIdentityFn = defaultAgeSopsGenerateIdentityFn
	ageSopsNowFn = defaultAgeSopsNowFn
	ageSopsOpenFn = os.Open
	ageSopsYAMLMarshalFn = defaultAgeSopsYAMLMarshalFn
	ageSopsMkdirAllFn = os.MkdirAll
	ageSopsWriteFileFn = os.WriteFile
	ageSopsCloseFileFn = defaultAgeSopsCloseFileFn
	ageSopsExitFn = defaultAgeSopsExitFn
	ageSopsGetSecKeyFn = GetSecKey
	ageSopsFatalErrMsgFn = defaultAgeSopsFatalErrMsgFn
}

package helper

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/beevik/ntp"
)

func resetHelperHooks(t *testing.T) {
	t.Helper()

	promptNewReaderFn = func() *bufio.Reader { return bufio.NewReader(os.Stdin) }
	promptReadStringFn = func(reader *bufio.Reader) (string, error) { return reader.ReadString('\n') }

	checkSystemTimeNTPTimeFn = ntp.Time
	checkSystemTimeNowFn = time.Now
	checkSystemTimeExitFn = os.Exit

	hookGetwdFn = os.Getwd
	hookStatFn = os.Stat
	hookCreateFn = os.Create
	hookWriteStringFn = func(file *os.File, content string) (int, error) { return file.WriteString(content) }
	hookChmodFn = os.Chmod
	hookGOOS = runtime.GOOS
	buildPreCommitHookScriptFn = buildPreCommitHookScript

	checkIgnoreFn = checkIgnore
	hasUnstagedChangesInGitFn = hasUnstagedChanges

	copyRelFn = filepath.Rel
	copyShouldSkipByFilterFn = shouldSkipByFilter
	copyPathEntryFn = copyPathEntry

	replaceBlockContentFn = replaceBlockContent

	toolDocsProcessFilesFn = processFiles
	toolDocsMoveMatchingFilesFn = moveMatchingFilesToSubdirs
	toolDocsRenameForgetoolFn = renameForgetoolToIndex

	processFilesReadDirFn = os.ReadDir
	processFilesReadFileFn = os.ReadFile
	processFilesWriteToFileFn = writeToFile
	processFilesRemoveFn = os.Remove

	writeToFileMkdirAllFn = os.MkdirAll
	writeToFileWriteFileFn = os.WriteFile
	writeToFileChmodFn = os.Chmod

	moveMatchingReadDirFn = os.ReadDir
	moveMatchingStatFn = os.Stat
	moveMatchingRenameFn = os.Rename

	renameForgetoolStatFn = os.Stat
	renameForgetoolRenameFn = os.Rename
}

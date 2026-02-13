package sops

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/getsops/sops/v3"
	"github.com/getsops/sops/v3/cmd/sops/common"
	"github.com/trueforge-org/forgetool/pkg/helper"
)

type fakeCypher struct {
	encryptFn func([]byte, EncryptionConfig) ([]byte, error)
}

func (f *fakeCypher) Decrypt(content []byte, config string) ([]byte, error) {
	return content, nil
}

func (f *fakeCypher) Encrypt(data []byte, cfg EncryptionConfig) ([]byte, error) {
	return f.encryptFn(data, cfg)
}

func TestCheckFilesAndSelectBranches(t *testing.T) {
	resetSopsHooks(t)
	sopsLoadSopsConfigFn = func() (SopsConfig, error) { return SopsConfig{}, errors.New("cfg") }
	if _, err := ExecuteCheck(false); err == nil {
		t.Fatal("expected ExecuteCheck config error")
	}

	resetSopsHooks(t)
	sopsLoadSopsConfigFn = func() (SopsConfig, error) { return SopsConfig{}, nil }
	sopsFilesToCheckFn = func(SopsConfig) ([]EncrFileData, error) { return nil, errors.New("files") }
	if _, err := ExecuteCheck(false); err == nil {
		t.Fatal("expected ExecuteCheck files error")
	}

	resetSopsHooks(t)
	sopsLoadSopsConfigFn = func() (SopsConfig, error) { return SopsConfig{}, nil }
	sopsFilesToCheckFn = func(SopsConfig) ([]EncrFileData, error) { return []EncrFileData{{Path: "a"}}, nil }
	sopsSelectFilesForCheckFn = func([]EncrFileData, bool) ([]EncrFileData, error) { return nil, errors.New("select") }
	if _, err := ExecuteCheck(false); err == nil {
		t.Fatal("expected ExecuteCheck select error")
	}

	resetSopsHooks(t)
	sopsLoadSopsConfigFn = func() (SopsConfig, error) { return SopsConfig{}, nil }
	sopsFilesToCheckFn = func(SopsConfig) ([]EncrFileData, error) { return []EncrFileData{{Path: "a"}}, nil }
	sopsSelectFilesForCheckFn = func([]EncrFileData, bool) ([]EncrFileData, error) { return []EncrFileData{{Path: "a"}}, nil }
	sopsReadFileFn = func(string) ([]byte, error) { return nil, errors.New("read") }
	if _, err := ExecuteCheck(false); err == nil {
		t.Fatal("expected ExecuteCheck read error")
	}

	resetSopsHooks(t)
	sopsExecuteCheckFn = func(bool) ([]EncrFileData, error) { return nil, errors.New("boom") }
	if err := CheckFilesAndReportEncryption(false, false); err == nil {
		t.Fatal("expected execute check error")
	}

	resetSopsHooks(t)
	sopsExecuteCheckFn = func(bool) ([]EncrFileData, error) {
		return []EncrFileData{{Path: "a", Encrypted: true}}, nil
	}
	sopsExitFn = func(int) { panic(exitPanic{}) }
	expectExitPanic(t, func() { _ = CheckFilesAndReportEncryption(false, false) })

	resetSopsHooks(t)
	handled := false
	sopsHandleUnencryptedFilesFn = func([]EncrFileData, bool) { handled = true }
	sopsExecuteCheckFn = func(bool) ([]EncrFileData, error) {
		return []EncrFileData{{Path: "a", Encrypted: false}}, nil
	}
	sopsExitFn = func(int) { panic(exitPanic{}) }
	expectExitPanic(t, func() { _ = CheckFilesAndReportEncryption(true, false) })
	if !handled {
		t.Fatal("expected unencrypted handler to be called")
	}

	resetSopsHooks(t)
	all := []EncrFileData{{Path: "a"}}
	selected, err := selectFilesForCheck(all, false)
	if err != nil || len(selected) != 1 {
		t.Fatalf("expected all files when staged disabled, got %v %v", selected, err)
	}

	resetSopsHooks(t)
	sopsGetStagedFilesFn = func() ([]string, error) { return nil, errors.New("git") }
	if _, err = selectFilesForCheck(all, true); err == nil {
		t.Fatal("expected staged files error")
	}

	resetSopsHooks(t)
	sopsGetStagedFilesFn = func() ([]string, error) { return []string{}, nil }
	if _, err = selectFilesForCheck(all, true); err == nil {
		t.Fatal("expected no staged files error")
	}

	resetSopsHooks(t)
	sopsGetStagedFilesFn = func() ([]string, error) { return []string{"a"}, nil }
	sopsStageFilteredFilesFn = func([]EncrFileData) error { return errors.New("stage") }
	if _, err = selectFilesForCheck(all, true); err == nil {
		t.Fatal("expected stage filtered error")
	}

	resetSopsHooks(t)
	sopsGetStagedFilesFn = func() ([]string, error) { return []string{"a"}, nil }
	sopsStageFilteredFilesFn = func([]EncrFileData) error { return nil }
	if got, err := selectFilesForCheck(all, true); err != nil || len(got) != 1 {
		t.Fatalf("expected staged success, got %v err=%v", got, err)
	}
}

func TestCheckEncryptHelpersAndSham(t *testing.T) {
	resetSopsHooks(t)
	if got := filterUnencryptedFiles([]EncrFileData{{Path: "a", Encrypted: true}, {Path: "b", Encrypted: false}}); len(got) != 1 {
		t.Fatalf("expected 1 unencrypted, got %d", len(got))
	}

	resetSopsHooks(t)
	sopsStageFilesFn = func([]string) error { return errors.New("fail") }
	if err := stageFilteredFiles([]EncrFileData{{Path: "a"}}); err == nil {
		t.Fatal("expected stage files error")
	}
	resetSopsHooks(t)
	sopsStageFilesFn = func([]string) error { return nil }
	if err := stageFilteredFiles([]EncrFileData{{Path: "a"}}); err != nil {
		t.Fatalf("unexpected stage files error: %v", err)
	}

	td := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	_ = os.Chdir(td)

	_ = os.WriteFile("DEVTRIGGER", []byte("x"), 0o644)
	filtered := filterStagedSopsFiles([]EncrFileData{{Path: "x.yaml"}}, []string{filepath.Join("forgetool", "x.yaml")})
	if len(filtered) != 1 {
		t.Fatalf("expected DEVTRIGGER path match, got %v", filtered)
	}

	resetSopsHooks(t)
	sopsProcessFileEncryptionFn = func(EncrFileData) error { return errors.New("encrypt") }
	tryEncryptAndStageFile(EncrFileData{Path: "a"})

	resetSopsHooks(t)
	sopsStageFileFn = func(string) error { return errors.New("stage") }
	sopsProcessFileEncryptionFn = func(EncrFileData) error { return nil }
	tryEncryptAndStageFile(EncrFileData{Path: "a"})

	resetSopsHooks(t)
	sopsStageFileFn = func(string) error { return nil }
	sopsProcessFileEncryptionFn = func(EncrFileData) error { return nil }
	tryEncryptAndStageFile(EncrFileData{Path: "a"})

	resetSopsHooks(t)
	sopsReadFileFn = func(path string) ([]byte, error) {
		if strings.Contains(path, "err") {
			return nil, errors.New("read")
		}
		if strings.Contains(path, "plain") {
			return []byte("plain"), nil
		}
		return []byte("enc"), nil
	}
	sopsIsEncryptedFn = func(data []byte, _ string) bool { return string(data) == "enc" }
	still := findStillUnencrypted([]EncrFileData{{Path: "err"}, {Path: "plain"}, {Path: "enc"}})
	if len(still) != 2 {
		t.Fatalf("expected 2 still unencrypted, got %v", still)
	}

	resetSopsHooks(t)
	sopsFatalFn = func(string) {}
	sopsExitFn = func(int) { panic(exitPanic{}) }
	expectExitPanic(t, func() { handleUnencryptedFiles([]EncrFileData{{Path: "a"}}, false) })

	resetSopsHooks(t)
	sopsFatalFn = func(string) {}
	sopsExitFn = func(int) { panic(exitPanic{}) }
	sopsTryEncryptAndStageFileFn = func(EncrFileData) {}
	sopsFindStillUnencryptedFn = func([]EncrFileData) []string { return []string{"a"} }
	expectExitPanic(t, func() { handleUnencryptedFiles([]EncrFileData{{Path: "a"}}, true) })

	resetSopsHooks(t)
	oldClusterEnv := helper.ClusterEnvFile
	helper.ClusterEnvFile = filepath.Join(td, "cluster.env")
	defer func() { helper.ClusterEnvFile = oldClusterEnv }()
	_ = os.WriteFile(helper.ClusterEnvFile, []byte("shamir_threshold: 1\n"), 0o644)
	sopsTryEncryptAndStageFileFn = func(EncrFileData) {}
	sopsFindStillUnencryptedFn = func([]EncrFileData) []string { return nil }
	handleUnencryptedFiles([]EncrFileData{{Path: "a"}}, true)

	resetSopsHooks(t)
	sopsExitFn = func(int) { panic(exitPanic{}) }
	oldClusterEnv = helper.ClusterEnvFile
	helper.ClusterEnvFile = filepath.Join(td, "missing.env")
	expectExitPanic(t, shamCheck)
	helper.ClusterEnvFile = oldClusterEnv

	resetSopsHooks(t)
	sopsExitFn = func(int) { panic(exitPanic{}) }
	helper.ClusterEnvFile = filepath.Join(td, "no-threshold.env")
	_ = os.WriteFile(helper.ClusterEnvFile, []byte("key: value\n"), 0o644)
	expectExitPanic(t, shamCheck)

	resetSopsHooks(t)
	sopsExitFn = func(int) { panic(exitPanic{}) }
	sopsScannerErrFn = func(*bufio.Scanner) error { return errors.New("scan") }
	helper.ClusterEnvFile = filepath.Join(td, "scan-err.env")
	_ = os.WriteFile(helper.ClusterEnvFile, []byte("key: value\n"), 0o644)
	expectExitPanic(t, shamCheck)
}

func TestFilesToCheckAndWalkRuleBranches(t *testing.T) {
	resetSopsHooks(t)
	sopsWalkRuleFilesFn = func(_ *regexp.Regexp, _ *[]EncrFileData) error { return errors.New("walk") }
	cfg := SopsConfig{}
	cfg.CreationRules = []struct {
		PathRegex      string `yaml:"path_regex"`
		EncryptedRegex string `yaml:"encrypted_regex,omitempty"`
		Age            string `yaml:"age"`
	}{{PathRegex: `.*`, Age: "a"}}
	if _, err := filesToCheck(cfg); err == nil {
		t.Fatal("expected walk rule error")
	}

	resetSopsHooks(t)
	sopsFilepathWalkFn = func(_ string, walkFn filepath.WalkFunc) error {
		return walkFn("x", nil, errors.New("walk error"))
	}
	if err := walkRuleFiles(regexp.MustCompile(`.*`), &[]EncrFileData{}); err == nil {
		t.Fatal("expected walk callback error")
	}
}

func TestDecryptBranches(t *testing.T) {
	resetSopsHooks(t)
	sopsExecuteCheckFn = func(bool) ([]EncrFileData, error) { return nil, errors.New("exec") }
	if err := DecryptFiles(); err == nil {
		t.Fatal("expected decrypt execute error")
	}

	resetSopsHooks(t)
	sopsExecuteCheckFn = func(bool) ([]EncrFileData, error) { return []EncrFileData{{Path: "a", Encrypted: true}}, nil }
	sopsDecryptMarkedFilesFn = func([]EncrFileData) (bool, error) { return true, errors.New("marked") }
	if err := DecryptFiles(); err == nil {
		t.Fatal("expected decrypt marked error")
	}

	resetSopsHooks(t)
	loaded := false
	sopsLoadTalEnvFn = func(bool) error { loaded = true; return nil }
	sopsExecuteCheckFn = func(bool) ([]EncrFileData, error) { return []EncrFileData{{Path: "a", Encrypted: false}}, nil }
	sopsDecryptMarkedFilesFn = func([]EncrFileData) (bool, error) { return false, nil }
	if err := DecryptFiles(); err != nil {
		t.Fatalf("unexpected decrypt files error: %v", err)
	}
	if !loaded {
		t.Fatal("expected tal env load")
	}

	resetSopsHooks(t)
	called := 0
	sopsDecryptFileFn = func(string) error {
		called++
		if called == 1 {
			return errors.New("fail")
		}
		return nil
	}
	found, err := decryptMarkedFiles([]EncrFileData{{Path: "a", Encrypted: false}, {Path: "b", Encrypted: true}})
	if err == nil || !found {
		t.Fatal("expected decryptMarkedFiles to report error with encrypted file")
	}

	resetSopsHooks(t)
	sopsDecryptFileFn = func(string) error { return nil }
	found, err = decryptMarkedFiles([]EncrFileData{{Path: "a", Encrypted: true}})
	if err != nil || !found {
		t.Fatalf("expected decryptMarkedFiles success with encrypted file, found=%v err=%v", found, err)
	}

	resetSopsHooks(t)
	sopsDecryptReadFileFn = func(string) ([]byte, error) { return nil, errors.New("read") }
	if err := decryptFile("a"); err == nil {
		t.Fatal("expected decryptFile read error")
	}

	resetSopsHooks(t)
	sopsDecryptReadFileFn = func(string) ([]byte, error) { return []byte("x"), nil }
	sopsDecryptDataWithRetryFn = func([]byte, string) ([]byte, error) { return nil, errors.New("decrypt") }
	if err := decryptFile("a.yaml"); err == nil {
		t.Fatal("expected decryptFile decrypt error")
	}

	resetSopsHooks(t)
	sopsDecryptReadFileFn = func(string) ([]byte, error) { return []byte("x"), nil }
	sopsDecryptDataWithRetryFn = func([]byte, string) ([]byte, error) { return []byte("ok"), nil }
	sopsDecryptWriteFileFn = func(string, []byte, os.FileMode) error { return errors.New("write") }
	if err := decryptFile("a.yaml"); err == nil {
		t.Fatal("expected decryptFile write error")
	}

	resetSopsHooks(t)
	sopsDecryptReadFileFn = func(string) ([]byte, error) { return []byte("x"), nil }
	sopsDecryptDataWithRetryFn = func([]byte, string) ([]byte, error) { return []byte("ok"), nil }
	sopsDecryptWriteFileFn = func(string, []byte, os.FileMode) error { return nil }
	if err := decryptFile("a.yaml"); err != nil {
		t.Fatalf("unexpected decryptFile error: %v", err)
	}
}

func TestDecryptDataBranches(t *testing.T) {
	resetSopsHooks(t)
	sopsDecryptDataFn = func([]byte, string) ([]byte, error) {
		return nil, errors.New("Failed to decrypt original mac")
	}
	out, err := decryptData([]byte("src"), "yaml")
	if err != nil || string(out) != "src" {
		t.Fatal("expected MAC ignore branch to return original data")
	}

	resetSopsHooks(t)
	sopsDecryptDataFn = func([]byte, string) ([]byte, error) { return nil, errors.New("other") }
	if _, err = decryptData([]byte("src"), "yaml"); err == nil {
		t.Fatal("expected decryptData generic error")
	}

	resetSopsHooks(t)
	sopsDecryptDataFn = func([]byte, string) ([]byte, error) { return []byte("ok"), nil }
	if out, err = decryptData([]byte("src"), "yaml"); err != nil || string(out) != "ok" {
		t.Fatal("expected decryptData success")
	}

	resetSopsHooks(t)
	sopsDecryptCoreFn = func([]byte, string) ([]byte, error) { return nil, &MacFailureError{OriginalError: errors.New("mac")} }
	sopsDecryptDataIgnoringMacFn = func([]byte, string) ([]byte, error) { return nil, errors.New("retry") }
	if _, err = decryptDataWithRetry([]byte("x"), "yaml"); err == nil {
		t.Fatal("expected retry decryption error")
	}

	resetSopsHooks(t)
	sopsDecryptCoreFn = func([]byte, string) ([]byte, error) { return nil, &MacFailureError{OriginalError: errors.New("mac")} }
	sopsDecryptDataIgnoringMacFn = func([]byte, string) ([]byte, error) { return []byte("ok"), nil }
	if out, err = decryptDataWithRetry([]byte("x"), "yaml"); err != nil || string(out) != "ok" {
		t.Fatal("expected retry path success")
	}

	resetSopsHooks(t)
	sopsDecryptCoreFn = func([]byte, string) ([]byte, error) { return nil, errors.New("plain") }
	if _, err = decryptDataWithRetry([]byte("x"), "yaml"); err == nil {
		t.Fatal("expected non-mac retry error passthrough")
	}

	resetSopsHooks(t)
	sopsDecryptDataFn = func([]byte, string) ([]byte, error) { return nil, errors.New("MAC verification failed") }
	if out, err = decryptDataIgnoringMac([]byte("d"), "yaml"); err != nil || string(out) != "d" {
		t.Fatal("expected ignoring mac branch")
	}

	resetSopsHooks(t)
	sopsDecryptDataFn = func([]byte, string) ([]byte, error) { return nil, errors.New("bad") }
	if _, err = decryptDataIgnoringMac([]byte("d"), "yaml"); err == nil {
		t.Fatal("expected decryptDataIgnoringMac generic error")
	}
}

func TestEncryptAndLoadSopsBranches(t *testing.T) {
	resetSopsHooks(t)
	sopsExecuteCheckFn = func(bool) ([]EncrFileData, error) { return nil, errors.New("exec") }
	if err := EncryptAllFiles(); err == nil {
		t.Fatal("expected EncryptAllFiles execute error")
	}

	resetSopsHooks(t)
	sopsExecuteCheckFn = func(bool) ([]EncrFileData, error) { return []EncrFileData{{Path: "a"}}, nil }
	sopsProcessFileEncryptionFn = func(EncrFileData) error { return errors.New("process") }
	if err := EncryptAllFiles(); err == nil {
		t.Fatal("expected EncryptAllFiles process error")
	}

	resetSopsHooks(t)
	sopsExecuteCheckFn = func(bool) ([]EncrFileData, error) { return []EncrFileData{{Path: "a", Encrypted: true}}, nil }
	sopsProcessFileEncryptionFn = func(EncrFileData) error { return nil }
	if err := EncryptAllFiles(); err != nil {
		t.Fatalf("unexpected EncryptAllFiles error: %v", err)
	}

	resetSopsHooks(t)
	if err := processFileEncryption(EncrFileData{Path: "a", Encrypted: true}); err != nil {
		t.Fatalf("expected encrypted skip nil error: %v", err)
	}

	resetSopsHooks(t)
	sopsIsFileFullyStagedFn = func(string) (bool, error) { return false, errors.New("stage-check") }
	if err := processFileEncryption(EncrFileData{Path: "a"}); err == nil {
		t.Fatal("expected staged check error")
	}

	resetSopsHooks(t)
	sopsIsFileFullyStagedFn = func(string) (bool, error) { return false, nil }
	sopsStageFileFn = func(string) error { return errors.New("stage") }
	if err := processFileEncryption(EncrFileData{Path: "a"}); err == nil {
		t.Fatal("expected stage error")
	}

	resetSopsHooks(t)
	sopsIsFileFullyStagedFn = func(string) (bool, error) { return false, nil }
	sopsStageFileFn = func(string) error { return nil }
	sopsEncryptFileFn = func(string) error { return nil }
	if err := processFileEncryption(EncrFileData{Path: "a"}); err != nil {
		t.Fatalf("expected partial stage success path, got %v", err)
	}

	resetSopsHooks(t)
	sopsIsFileFullyStagedFn = func(string) (bool, error) { return true, nil }
	sopsEncryptFileFn = func(string) error { return errors.New("encrypt") }
	if err := processFileEncryption(EncrFileData{Path: "a"}); err == nil {
		t.Fatal("expected encrypt error")
	}

	resetSopsHooks(t)
	sopsIsFileFullyStagedFn = func(string) (bool, error) { return true, nil }
	sopsEncryptFileFn = func(string) error { return nil }
	if err := processFileEncryption(EncrFileData{Path: "a"}); err != nil {
		t.Fatalf("unexpected processFileEncryption error: %v", err)
	}

	resetSopsHooks(t)
	sopsEncryptReadFileFn = func(string) ([]byte, error) { return nil, errors.New("read") }
	if err := encryptFile("a.yaml"); err == nil {
		t.Fatal("expected encryptFile read error")
	}

	resetSopsHooks(t)
	sopsEncryptReadFileFn = func(string) ([]byte, error) { return []byte("x"), nil }
	sopsLoadSopsConfigFn = func() (SopsConfig, error) { return SopsConfig{}, errors.New("cfg") }
	if err := encryptFile("a.yaml"); err == nil {
		t.Fatal("expected encryptFile config error")
	}

	resetSopsHooks(t)
	sopsEncryptReadFileFn = func(string) ([]byte, error) { return []byte("x"), nil }
	sopsLoadSopsConfigFn = func() (SopsConfig, error) { return SopsConfig{}, nil }
	sopsMergeRegexFn = func(string, SopsConfig) string { return "" }
	sopsEncryptWithAgeKeyFn = func([]byte, string, string) ([]byte, error) { return nil, errors.New("enc") }
	if err := encryptFile("a.yaml"); err == nil {
		t.Fatal("expected encrypt data error")
	}

	resetSopsHooks(t)
	sopsEncryptReadFileFn = func(string) ([]byte, error) { return []byte("x"), nil }
	sopsLoadSopsConfigFn = func() (SopsConfig, error) { return SopsConfig{}, nil }
	sopsMergeRegexFn = func(string, SopsConfig) string { return "" }
	sopsEncryptWithAgeKeyFn = func([]byte, string, string) ([]byte, error) { return []byte("ok"), nil }
	sopsEncryptWriteFileFn = func(string, []byte, os.FileMode) error { return errors.New("write") }
	if err := encryptFile("a.yaml"); err == nil {
		t.Fatal("expected encrypt write error")
	}

	resetSopsHooks(t)
	sopsEncryptReadFileFn = func(string) ([]byte, error) { return []byte("x"), nil }
	sopsLoadSopsConfigFn = func() (SopsConfig, error) { return SopsConfig{}, nil }
	sopsMergeRegexFn = func(string, SopsConfig) string { return "" }
	sopsEncryptWithAgeKeyFn = func([]byte, string, string) ([]byte, error) { return []byte("ok"), nil }
	sopsEncryptWriteFileFn = func(string, []byte, os.FileMode) error { return nil }
	if err := encryptFile("a.yaml"); err != nil {
		t.Fatalf("unexpected encryptFile error: %v", err)
	}

	resetSopsHooks(t)
	sopsLoadConfigStatFn = func(string) (os.FileInfo, error) { return nil, errors.New("stat") }
	sopsLoadConfigReadFileFn = func(string) ([]byte, error) {
		return []byte("creation_rules:\n  - path_regex: \".*\"\n    age: \"a\"\n"), nil
	}
	if _, err := LoadSopsConfig(); err != nil {
		t.Fatalf("expected LoadSopsConfig to continue after stat error branch: %v", err)
	}

	resetSopsHooks(t)
	sopsLoadConfigStatFn = func(string) (os.FileInfo, error) { return nil, nil }
	sopsLoadConfigReadFileFn = func(string) ([]byte, error) { return nil, errors.New("read") }
	if _, err := LoadSopsConfig(); err == nil {
		t.Fatal("expected LoadSopsConfig read error")
	}
}

func TestWrapperBranches(t *testing.T) {
	resetSopsHooks(t)
	sopsLoadSopsConfigFn = func() (SopsConfig, error) { return SopsConfig{}, errors.New("cfg") }
	if _, err := EncryptWithAgeKey([]byte("x"), "", "yaml"); err == nil {
		t.Fatal("expected EncryptWithAgeKey config error")
	}

	resetSopsHooks(t)
	sopsLoadSopsConfigFn = func() (SopsConfig, error) {
		cfg := SopsConfig{}
		cfg.CreationRules = []struct {
			PathRegex      string `yaml:"path_regex"`
			EncryptedRegex string `yaml:"encrypted_regex,omitempty"`
			Age            string `yaml:"age"`
		}{{PathRegex: ".*", Age: "age1x"}}
		return cfg, nil
	}
	sopsNewCypherFn = func() Cypher {
		return &fakeCypher{encryptFn: func([]byte, EncryptionConfig) ([]byte, error) {
			return nil, errors.New("enc")
		}}
	}
	if _, err := EncryptWithAgeKey([]byte("x"), "", "yaml"); err == nil {
		t.Fatal("expected EncryptWithAgeKey cypher error")
	}

	resetSopsHooks(t)
	sopsLoadSopsConfigFn = func() (SopsConfig, error) {
		cfg := SopsConfig{}
		cfg.CreationRules = []struct {
			PathRegex      string `yaml:"path_regex"`
			EncryptedRegex string `yaml:"encrypted_regex,omitempty"`
			Age            string `yaml:"age"`
		}{{PathRegex: ".*", Age: "age1x"}}
		return cfg, nil
	}
	sopsCollectAgeKeysFn = func(SopsConfig) []string { return []string{"age1x", ""} }
	sopsBuildAgeKeyGroupsFn = func([]string) []sops.KeyGroup { return []sops.KeyGroup{} }
	sopsNewCypherFn = func() Cypher {
		return &fakeCypher{encryptFn: func([]byte, EncryptionConfig) ([]byte, error) {
			return []byte("ok"), nil
		}}
	}
	if out, err := EncryptWithAgeKey([]byte("x"), "", "yaml"); err != nil || string(out) != "ok" {
		t.Fatal("expected EncryptWithAgeKey success")
	}

	if len(collectAgeKeys(SopsConfig{})) != 0 {
		t.Fatal("expected no age keys from empty config")
	}
	_ = buildAgeKeyGroups([]string{"not-valid", "not-valid", ""})

	resetSopsHooks(t)
	c := NewCypher()
	sopsDecryptDataFn = func([]byte, string) ([]byte, error) { return nil, errors.New("dec") }
	if _, err := c.Decrypt([]byte("x"), "yaml"); err == nil {
		t.Fatal("expected cypher decrypt error")
	}
	sopsDecryptDataFn = func([]byte, string) ([]byte, error) { return []byte("ok"), nil }
	if out, err := c.Decrypt([]byte("x"), "yaml"); err != nil || string(out) != "ok" {
		t.Fatal("expected cypher decrypt success")
	}

	resetSopsHooks(t)
	cy := &cypher{}
	sopsLoadPlainFileFn = func(common.Store, []byte) (sops.TreeBranches, error) { return nil, errors.New("plain") }
	if _, err := cy.Encrypt([]byte("x"), EncryptionConfig{Format: "yaml"}); err == nil {
		t.Fatal("expected cypher encrypt load plain error")
	}

	resetSopsHooks(t)
	sopsLoadPlainFileFn = func(common.Store, []byte) (sops.TreeBranches, error) { return nil, nil }
	sopsGenerateDataKeyFn = func(*sops.Tree) ([]byte, []error) { return nil, []error{errors.New("key")} }
	if _, err := cy.Encrypt([]byte("x"), EncryptionConfig{Format: "yaml"}); err == nil {
		t.Fatal("expected cypher encrypt data key error")
	}

	resetSopsHooks(t)
	sopsLoadPlainFileFn = func(common.Store, []byte) (sops.TreeBranches, error) { return nil, nil }
	sopsGenerateDataKeyFn = func(*sops.Tree) ([]byte, []error) { return []byte("k"), nil }
	sopsEncryptTreeFn = func(common.EncryptTreeOpts) error { return errors.New("tree") }
	if _, err := cy.Encrypt([]byte("x"), EncryptionConfig{Format: "yaml"}); err == nil {
		t.Fatal("expected cypher encrypt tree error")
	}

	resetSopsHooks(t)
	sopsLoadPlainFileFn = func(common.Store, []byte) (sops.TreeBranches, error) { return nil, nil }
	sopsGenerateDataKeyFn = func(*sops.Tree) ([]byte, []error) { return []byte("k"), nil }
	sopsEncryptTreeFn = func(common.EncryptTreeOpts) error { return nil }
	sopsEmitEncryptedFileFn = func(common.Store, sops.Tree) ([]byte, error) { return nil, fmt.Errorf("emit") }
	if _, err := cy.Encrypt([]byte("x"), EncryptionConfig{Format: "yaml"}); err == nil {
		t.Fatal("expected cypher encrypt emit error")
	}

	resetSopsHooks(t)
	sopsLoadPlainFileFn = func(common.Store, []byte) (sops.TreeBranches, error) { return nil, nil }
	sopsGenerateDataKeyFn = func(*sops.Tree) ([]byte, []error) { return []byte("k"), nil }
	sopsEncryptTreeFn = func(common.EncryptTreeOpts) error { return nil }
	sopsEmitEncryptedFileFn = func(common.Store, sops.Tree) ([]byte, error) { return []byte("ok"), nil }
	if out, err := cy.Encrypt([]byte("x"), EncryptionConfig{Format: "yaml"}); err != nil || string(out) != "ok" {
		t.Fatal("expected cypher encrypt success")
	}
}

func TestHooksDefaultHelpers(t *testing.T) {
	defaultSopsFatal("test")

	store := defaultStoreForFormat(formatJson)
	_, _ = defaultLoadPlainFile(store, []byte("{}"))

	tree := &sops.Tree{}
	_, _ = defaultGenerateDataKey(tree)

	func() {
		defer func() { _ = recover() }()
		_ = defaultEncryptTree(common.EncryptTreeOpts{
			DataKey: []byte("k"),
			Tree:    &sops.Tree{},
			Cipher:  sopsNewCipherFn(),
		})
	}()

	func() {
		defer func() { _ = recover() }()
		_, _ = defaultEmitEncryptedFile(store, sops.Tree{})
	}()
}

package initfiles

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	age "filippo.io/age"
	"github.com/trueforge-org/forgetool/pkg/helper"
)

func mustPanic(t *testing.T, fn func()) {
	t.Helper()
	didPanic := false
	defer func() {
		if recover() != nil {
			didPanic = true
		}
		if !didPanic {
			t.Fatalf("expected panic")
		}
	}()
	fn()
}

func setupInitfilesTempEnv(t *testing.T) string {
	t.Helper()
	resetInitfilesCoverageHooks()
	td := t.TempDir()
	oldClusterPath := helper.ClusterPath
	oldClusterEnv := helper.ClusterEnvFile
	oldClusterName := helper.ClusterName
	oldTalEnv := helper.TalEnv

	helper.ClusterPath = td
	helper.ClusterEnvFile = filepath.Join(td, "clusterenv.yaml")
	helper.ClusterName = "ut"
	helper.TalEnv = map[string]string{}

	t.Cleanup(func() {
		helper.ClusterPath = oldClusterPath
		helper.ClusterEnvFile = oldClusterEnv
		helper.ClusterName = oldClusterName
		helper.TalEnv = oldTalEnv
		resetInitfilesCoverageHooks()
	})

	return td
}

func TestLoadTalEnv_Branches(t *testing.T) {
	t.Run("missing file noFail", func(t *testing.T) {
		setupInitfilesTempEnv(t)
		clusterenvStatFn = func(string) (os.FileInfo, error) { return nil, os.ErrNotExist }
		if err := LoadTalEnv(true); err != nil {
			t.Fatalf("expected nil err, got %v", err)
		}
	})

	t.Run("load error exits", func(t *testing.T) {
		setupInitfilesTempEnv(t)
		clusterenvStatFn = func(string) (os.FileInfo, error) { return nil, nil }
		clusterenvLoadEnvFromFileFn = func(string, map[string]string) error { return errors.New("boom") }
		clusterenvExitFn = func(int) { panic("exit") }
		mustPanic(t, func() { _ = LoadTalEnv(false) })
	})

	t.Run("stat unexpected error exits", func(t *testing.T) {
		setupInitfilesTempEnv(t)
		clusterenvStatFn = func(string) (os.FileInfo, error) { return nil, errors.New("stat") }
		clusterenvExitFn = func(int) { panic("exit") }
		mustPanic(t, func() { _ = LoadTalEnv(false) })
	})

	t.Run("missing file noFail false", func(t *testing.T) {
		setupInitfilesTempEnv(t)
		clusterenvStatFn = func(string) (os.FileInfo, error) { return nil, os.ErrNotExist }
		clusterenvFatalMsgFn = func(string) {}
		clusterenvExitFn = func(int) { panic("exit") }
		mustPanic(t, func() { _ = LoadTalEnv(false) })
	})

	t.Run("full success", func(t *testing.T) {
		td := setupInitfilesTempEnv(t)
		content := "VIP: \"10.0.0.10\"\nMASTER1IP: \"10.0.0.11\"\n"
		if err := os.WriteFile(filepath.Join(td, "clusterenv.yaml"), []byte(content), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		clusterenvLoadEnvFromFileFn = func(_ string, env map[string]string) error {
			env["VIP"] = "10.0.0.10"
			env["MASTER1IP"] = "10.0.0.11"
			return nil
		}
		if err := LoadTalEnv(false); err != nil {
			t.Fatalf("LoadTalEnv error: %v", err)
		}
	})
}

func TestCheckQuotedNumbersInFile_Branches(t *testing.T) {
	t.Run("open error", func(t *testing.T) {
		setupInitfilesTempEnv(t)
		clusterenvOpenFn = func(string) (*os.File, error) { return nil, errors.New("open") }
		clusterenvExitFn = func(int) { panic("exit") }
		mustPanic(t, func() { _, _ = checkQuotedNumbersInFile() })
	})

	t.Run("unquoted number exits", func(t *testing.T) {
		td := setupInitfilesTempEnv(t)
		if err := os.WriteFile(filepath.Join(td, "clusterenv.yaml"), []byte("X: 123\n"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		clusterenvExitFn = func(int) { panic("exit") }
		mustPanic(t, func() { _, _ = checkQuotedNumbersInFile() })
	})

	t.Run("scanner err exits", func(t *testing.T) {
		td := setupInitfilesTempEnv(t)
		longLine := strings.Repeat("a", 1024*128)
		if err := os.WriteFile(filepath.Join(td, "clusterenv.yaml"), []byte("X: \""+longLine+"\""), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		clusterenvExitFn = func(int) { panic("exit") }
		mustPanic(t, func() { _, _ = checkQuotedNumbersInFile() })
	})
}

func TestLineHasUnquotedNumber(t *testing.T) {
	re := regexp.MustCompile(`:\s*(.+)`)
	if lineHasUnquotedNumber("1: 123", re) {
		t.Fatalf("numeric line key should be ignored")
	}
	if !lineHasUnquotedNumber("KEY: 123", re) {
		t.Fatalf("expected true for unquoted number")
	}
	if lineHasUnquotedNumber("KEY: \"123\"", re) {
		t.Fatalf("quoted numbers should be false")
	}
	if lineHasUnquotedNumber("KEY", re) {
		t.Fatalf("line without separator should be false")
	}
}

func TestClusterenvValidatorsAndChecks(t *testing.T) {
	setupInitfilesTempEnv(t)
	helper.TalEnv = map[string]string{
		"VIP":                       "10.0.0.10",
		"MASTER1IP_IP":              "10.0.0.11",
		"MASTER1IP_NETMASK":         "24",
		"MASTER1IP_CIDR":            "10.0.0.11/24",
		"DASHBOARD_IP":              "10.0.0.50",
		"GATEWAY":                   "10.0.0.1",
		"METALLB_RANGE":             "10.0.0.100-10.0.0.110",
		"PODNET":                    "10.244.0.0/16",
		"SVCNET":                    "10.96.0.0/12",
		"DOMAIN_0":                  "example.com",
		"DOMAIN_0_EMAIL":            "ops@example.com",
		"DOMAIN_0_CLOUDFLARE_TOKEN": "tok",
		"VIP_IP":                    "10.0.0.10",
	}

	called := 0
	clusterenvLoadTalEnvFn = func(bool) error { return nil }
	clusterenvIPInRangeFn = func(ip string, _ string) (bool, error) {
		if ip == helper.TalEnv["DASHBOARD_IP"] {
			return true, nil
		}
		return false, nil
	}
	clusterenvValidateIPorCIDRNotInCIDRFn = func(_, _, _, _ string) { called++ }
	clusterenvValidateRangeNotInCIDRFn = func(_, _, _, _ string) { called++ }

	CheckEnvVariables()
	if called != 8 {
		t.Fatalf("expected 8 cidr validations, got %d", called)
	}

	clusterenvExitFn = func(int) { panic("exit") }
	helper.TalEnv["DOMAIN_0"] = ""
	mustPanic(t, validateRequiredTalEnvKeys)

	calls := 0
	clusterenvExitFn = func(int) { calls++ }
	validateNodeAndGatewayIPs("10.0.0.10", "10.0.0.11", "10.0.0.11")
	if calls != 1 {
		t.Fatalf("expected MASTER1IP/GATEWAY exit branch hit once, got %d", calls)
	}

	clusterenvIPInRangeFn = func(string, string) (bool, error) { return false, errors.New("range") }
	clusterenvExitFn = func(int) { panic("exit") }
	mustPanic(t, func() { validateIPNotInMetalLBRange("10.0.0.1", "VIP") })

	clusterenvIPInRangeFn = func(string, string) (bool, error) { return true, nil }
	mustPanic(t, func() { validateIPNotInMetalLBRange("10.0.0.1", "VIP") })

	helper.TalEnv["DASHBOARD_IP"] = ""
	validateDashboardInMetalLBRange()
	clusterenvIPInRangeFn = func(string, string) (bool, error) { return false, nil }
	helper.TalEnv["DASHBOARD_IP"] = "10.0.0.50"
	mustPanic(t, validateDashboardInMetalLBRange)
	clusterenvIPInRangeFn = func(string, string) (bool, error) { return false, errors.New("range") }
	mustPanic(t, validateDashboardInMetalLBRange)
}

func TestInitFiles_Orchestration(t *testing.T) {
	setupInitfilesTempEnv(t)
	order := []string{}
	push := func(name string) { order = append(order, name) }

	initfilesRemoveRunAgainFileFn = func() error { push("remove"); return nil }
	initfilesAgeGenFn = func() error { push("age"); return nil }
	initfilesGenRootFilesFn = func() error { push("root"); return nil }
	initfilesGenBaseFilesFn = func() error { push("base"); return nil }
	initfilesUpdateRootFilesFn = func() error { push("updateroot"); return nil }
	initfilesUpdateBaseFilesFn = func() error { push("updatebase"); return nil }
	initfilesGenSchemaFn = func() error { push("schema"); return nil }
	initfilesGenPatchesFn = func() error { push("patches"); return nil }
	initfilesGenKubernetesFn = func() error { push("k8s"); return nil }
	initfilesGenTalEnvConfigMapFn = func() error { push("cm"); return nil }
	initfilesUpdateGitRepoFn = func() { push("gitrepo") }
	initfilesCreateGitSecretFn = func(string) error { push("gitsecret"); return nil }
	initfilesGenSopsSecretFn = func() error { push("sops"); return nil }
	initfilesProcessKustomizationsFn = func(string) { push("kustomize") }
	initfilesCreateEncrPreCommitHookFn = func() error { push("hook"); return nil }

	if err := InitFiles(); err != nil {
		t.Fatalf("InitFiles error: %v", err)
	}
	if len(order) != 15 {
		t.Fatalf("unexpected call count: %d", len(order))
	}
}

func TestProcessKustomizationsBranches(t *testing.T) {
	setupInitfilesTempEnv(t)
	count := 0
	initfilesProcessDirectoryFn = func(string) error {
		count++
		if count == 1 {
			return errors.New("first")
		}
		return nil
	}
	processKustomizations("x")

	count = 0
	initfilesProcessDirectoryFn = func(string) error {
		count++
		if count == 2 {
			return errors.New("second")
		}
		return nil
	}
	processKustomizations("x")
}

func TestInitfilesCoreFunctions(t *testing.T) {
	t.Run("genKubernetes fatal branch and success", func(t *testing.T) {
		setupInitfilesTempEnv(t)
		initfilesCopyDirFn = func(string, string, bool) error { return errors.New("copy") }
		initfilesFatalErrMsgfFn = func(error, string, ...interface{}) { panic("fatal") }
		mustPanic(t, func() { _ = genKubernetes() })

		resetInitfilesCoverageHooks()
		setupInitfilesTempEnv(t)
		initfilesCopyDirFn = func(string, string, bool) error { return nil }
		initfilesReplaceInFileFn = func(string, string, string) error { return nil }
		if err := genKubernetes(); err != nil {
			t.Fatalf("genKubernetes error: %v", err)
		}
	})

	t.Run("GenTalEnvConfigMap read/copy branches", func(t *testing.T) {
		td := setupInitfilesTempEnv(t)
		if err := os.WriteFile(helper.ClusterEnvFile, []byte("A: B\n"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		helper.KubeCache = filepath.Join(td, "cache")
		_ = os.MkdirAll(filepath.Join(helper.KubeCache, "flux-system", "flux"), 0o755)
		src := filepath.Join(helper.KubeCache, "flux-system", "flux", "clustersettings.secret.yaml")
		if err := os.WriteFile(src, []byte("REPLACEWITHENV"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		if err := GenTalEnvConfigMap(); err != nil {
			t.Fatalf("GenTalEnvConfigMap error: %v", err)
		}

		initfilesReadFileFn = func(string) ([]byte, error) { return nil, errors.New("read") }
		if err := GenTalEnvConfigMap(); err == nil {
			t.Fatalf("expected read error")
		}

		initfilesReadFileFn = os.ReadFile
		initfilesCopyFileFn = func(string, string, bool) error { return errors.New("copy") }
		initfilesFatalErrMsgFn = func(error, string) { panic("fatal") }
		mustPanic(t, func() { _ = GenTalEnvConfigMap() })
	})

	t.Run("genBaseFiles branches", func(t *testing.T) {
		setupInitfilesTempEnv(t)
		initfilesStatFn = func(string) (os.FileInfo, error) { return nil, nil }
		initfilesCopyDirFn = func(string, string, bool) error { return errors.New("copy") }
		if err := genBaseFiles(); err != nil {
			t.Fatalf("expected nil error on existing env, got %v", err)
		}

		setupInitfilesTempEnv(t)
		initfilesStatFn = func(string) (os.FileInfo, error) { return nil, os.ErrNotExist }
		initfilesCreateRunAgainFileFn = func() {}
		initfilesCopyDirFn = func(string, string, bool) error { return nil }
		initfilesExitFn = func(int) { panic("exit") }
		mustPanic(t, func() { _ = genBaseFiles() })

		setupInitfilesTempEnv(t)
		initfilesStatFn = func(string) (os.FileInfo, error) { return nil, errors.New("stat") }
		initfilesIsNotExistFn = func(error) bool { return false }
		initfilesFatalErrMsgfFn = func(error, string, ...interface{}) {}
		if err := genBaseFiles(); err == nil {
			t.Fatalf("expected stat error")
		}
	})

	t.Run("UpdateBaseFiles and UpdateRootFiles", func(t *testing.T) {
		setupInitfilesTempEnv(t)
		initfilesReadFilenamesInDirFn = func(string) ([]string, error) { return nil, errors.New("rd") }
		if err := UpdateBaseFiles(); err == nil {
			t.Fatalf("expected UpdateBaseFiles error")
		}

		initfilesReadFilenamesInDirFn = func(string) ([]string, error) { return []string{"f"}, nil }
		initfilesReplaceContentBetweenLinesFn = func(string, string, string, string) error { return nil }
		initfilesCheckEnvVariablesFn = func() {}
		if err := UpdateBaseFiles(); err != nil {
			t.Fatalf("UpdateBaseFiles error: %v", err)
		}

		initfilesReadFilenamesInDirFn = func(string) ([]string, error) { return nil, errors.New("rd") }
		if err := UpdateRootFiles(); err == nil {
			t.Fatalf("expected UpdateRootFiles error")
		}

		initfilesReadFilenamesInDirFn = func(string) ([]string, error) { return []string{"f"}, nil }
		initfilesGetPubKeyFn = func() (string, error) { return "", errors.New("pub") }
		initfilesFatalErrMsgFn = func(error, string) { panic("fatal") }
		mustPanic(t, func() { _ = UpdateRootFiles() })

		setupInitfilesTempEnv(t)
		initfilesReadFilenamesInDirFn = func(string) ([]string, error) { return []string{"f"}, nil }
		initfilesReplaceContentBetweenLinesFn = func(string, string, string, string) error { return nil }
		initfilesGetPubKeyFn = func() (string, error) { return "", errors.New("pub") }
		initfilesFatalErrMsgFn = func(error, string) {}
		initfilesReplaceInFileFn = func(string, string, string) error { return nil }
		initfilesCheckEnvVariablesFn = func() {}
		if err := UpdateRootFiles(); err != nil {
			t.Fatalf("UpdateRootFiles error: %v", err)
		}
	})

	t.Run("genRootFiles and ResetBootstrapValues and GenPatches", func(t *testing.T) {
		setupInitfilesTempEnv(t)
		initfilesCopyDirFn = func(string, string, bool) error { return errors.New("copy") }
		initfilesGetPubKeyFn = func() (string, error) { return "", errors.New("pub") }
		initfilesFatalErrMsgFn = func(error, string) {}
		initfilesReplaceInFileFn = func(string, string, string) error { return nil }
		if err := genRootFiles(); err != nil {
			t.Fatalf("genRootFiles error: %v", err)
		}

		setupInitfilesTempEnv(t)
		initfilesCopyDirFn = func(string, string, bool) error { return nil }
		initfilesGetPubKeyFn = func() (string, error) { return "age1abc", nil }
		initfilesReplaceInFileFn = func(string, string, string) error { return nil }
		if err := genRootFiles(); err != nil {
			t.Fatalf("genRootFiles error: %v", err)
		}

		initfilesGetPubKeyFn = func() (string, error) { return "", errors.New("pub") }
		initfilesFatalErrMsgFn = func(error, string) { panic("fatal") }
		mustPanic(t, func() { _ = genRootFiles() })

		setupInitfilesTempEnv(t)
		initfilesLoadTalEnvFn = func(bool) error { return nil }
		initfilesCopyDirFilteredFn = func(string, string, bool, string) error { return errors.New("copy") }
		initfilesEnvSubstRecursiveFn = func(string, string, map[string]string) error { return errors.New("envsubst") }
		if err := ResetBootstrapValues(); err != nil {
			t.Fatalf("ResetBootstrapValues error: %v", err)
		}

		setupInitfilesTempEnv(t)
		initfilesCopyDirFn = func(string, string, bool) error { return nil }
		initfilesGetSecKeyFn = func() (string, error) { return "secret", nil }
		initfilesSetDockerFn = func() {}
		initfilesReplaceInFileFn = func(string, string, string) error { return nil }
		if err := GenPatches(); err != nil {
			t.Fatalf("GenPatches error: %v", err)
		}

		setupInitfilesTempEnv(t)
		initfilesCopyDirFn = func(string, string, bool) error { return errors.New("copy") }
		initfilesGetSecKeyFn = func() (string, error) { return "secret", nil }
		initfilesSetDockerFn = func() {}
		initfilesReplaceInFileFn = func(string, string, string) error { return nil }
		if err := GenPatches(); err != nil {
			t.Fatalf("GenPatches error: %v", err)
		}

		initfilesGetSecKeyFn = func() (string, error) { return "", errors.New("sec") }
		initfilesFatalErrMsgFn = func(error, string) { panic("fatal") }
		mustPanic(t, func() { _ = GenPatches() })
	})

	t.Run("setDocker and appendContentToPatchFile", func(t *testing.T) {
		td := setupInitfilesTempEnv(t)
		_ = os.MkdirAll(filepath.Join(td, "talos", "patches"), 0o755)
		path := filepath.Join(td, "talos", "patches", "all.yaml")

		helper.TalEnv["DOCKERHUB_USER"] = "u"
		helper.TalEnv["DOCKERHUB_PASSWORD"] = "p"
		setDocker()
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if !strings.Contains(string(b), "docker.io") {
			t.Fatalf("expected docker config content")
		}

		helper.TalEnv["DOCKERHUB_USER"] = ""
		helper.TalEnv["DOCKERHUB_PASSWORD"] = ""
		setDocker()

		initfilesFatalErrMsgFn = func(error, string) { panic("fatal") }
		mustPanic(t, func() { appendContentToPatchFile(td, "x") })

		fpath := filepath.Join(td, "writeerr.txt")
		initfilesOpenFileFn = os.OpenFile
		initfilesWriteFileContentFn = func(*os.File, string) (int, error) { return 0, errors.New("write") }
		initfilesFatalErrMsgFn = func(error, string) { panic("fatal") }
		mustPanic(t, func() { appendContentToPatchFile(fpath, "x") })
	})
}

func TestAgeSopsFunctions(t *testing.T) {
	t.Run("ageGen and key readers", func(t *testing.T) {
		td := setupInitfilesTempEnv(t)
		cwd, _ := os.Getwd()
		defer os.Chdir(cwd)
		if err := os.Chdir(td); err != nil {
			t.Fatalf("chdir: %v", err)
		}

		if err := ageGen(); err != nil {
			t.Fatalf("ageGen error: %v", err)
		}
		if err := ageGen(); err != nil {
			t.Fatalf("ageGen second call error: %v", err)
		}

		if _, err := GetPubKey(); err != nil {
			t.Fatalf("GetPubKey error: %v", err)
		}
		if _, err := GetSecKey(); err != nil {
			t.Fatalf("GetSecKey error: %v", err)
		}
	})

	t.Run("ageGen fatal hooks", func(t *testing.T) {
		setupInitfilesTempEnv(t)
		ageSopsStatFn = func(string) (os.FileInfo, error) { return nil, os.ErrNotExist }
		ageSopsOpenFileFn = func(string, int, os.FileMode) (*os.File, error) { return nil, errors.New("open") }
		ageSopsFatalErrMsgFn = func(error, string) { panic("fatal") }
		mustPanic(t, func() { _ = ageGen() })

		ageSopsStatFn = func(string) (os.FileInfo, error) { return nil, errors.New("other") }
		if err := ageGen(); err != nil {
			t.Fatalf("ageGen error: %v", err)
		}
	})

	t.Run("ageGen world-readable and close/generate branches", func(t *testing.T) {
		td := setupInitfilesTempEnv(t)
		cwd, _ := os.Getwd()
		defer os.Chdir(cwd)
		if err := os.Chdir(td); err != nil {
			t.Fatalf("chdir: %v", err)
		}

		ageSopsStatFn = func(string) (os.FileInfo, error) { return nil, os.ErrNotExist }
		ageSopsOpenFileFn = func(name string, flag int, _ os.FileMode) (*os.File, error) {
			return os.OpenFile(name, flag, 0o644)
		}
		if err := ageGen(); err != nil {
			t.Fatalf("ageGen world-readable error: %v", err)
		}

		_ = os.Remove("age.agekey")
		ageSopsOpenFileFn = os.OpenFile
		ageSopsGenerateIdentityFn = func() (*age.X25519Identity, error) { return nil, errors.New("gen") }
		ageSopsFatalErrMsgFn = func(error, string) { panic("fatal") }
		mustPanic(t, func() { _ = ageGen() })

		_ = os.Remove("age.agekey")
		ageSopsGenerateIdentityFn = age.GenerateX25519Identity
		ageSopsCloseFileFn = func(*os.File) error { return errors.New("close") }
		ageSopsFatalErrMsgFn = func(error, string) { panic("fatal") }
		mustPanic(t, func() { _ = ageGen() })
	})

	t.Run("GetPubKey and GetSecKey scanner err", func(t *testing.T) {
		td := setupInitfilesTempEnv(t)
		cwd, _ := os.Getwd()
		defer os.Chdir(cwd)
		if err := os.Chdir(td); err != nil {
			t.Fatalf("chdir: %v", err)
		}

		if err := os.WriteFile("age.agekey", []byte(strings.Repeat("A", 1024*128)), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
		if _, err := GetPubKey(); err == nil {
			t.Fatalf("expected scanner error for GetPubKey")
		}
		if _, err := GetSecKey(); err == nil {
			t.Fatalf("expected scanner error for GetSecKey")
		}
	})

	t.Run("GenSopsSecret branches", func(t *testing.T) {
		td := setupInitfilesTempEnv(t)
		helper.ClusterPath = td

		ageSopsGetSecKeyFn = func() (string, error) { return "", errors.New("sec") }
		if err := GenSopsSecret(); err == nil {
			t.Fatalf("expected sec key error")
		}

		ageSopsGetSecKeyFn = func() (string, error) { return "AGE-SECRET-KEY-TEST", nil }
		ageSopsYAMLMarshalFn = func(any) ([]byte, error) { return nil, errors.New("marshal") }
		if err := GenSopsSecret(); err == nil {
			t.Fatalf("expected marshal error")
		}

		ageSopsYAMLMarshalFn = func(v any) ([]byte, error) { return []byte(fmt.Sprintf("%v", v)), nil }
		ageSopsMkdirAllFn = func(string, os.FileMode) error { return errors.New("mkdir") }
		if err := GenSopsSecret(); err == nil {
			t.Fatalf("expected mkdir error")
		}

		ageSopsMkdirAllFn = os.MkdirAll
		ageSopsWriteFileFn = func(string, []byte, os.FileMode) error { return errors.New("write") }
		if err := GenSopsSecret(); err == nil {
			t.Fatalf("expected write error")
		}

		ageSopsWriteFileFn = os.WriteFile
		if err := GenSopsSecret(); err != nil {
			t.Fatalf("GenSopsSecret error: %v", err)
		}
	})
}

func TestRunAgainErrorBranches(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	if err := os.Mkdir("RUNAGAIN", 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	createRunAgainFile()

	if err := os.WriteFile(filepath.Join("RUNAGAIN", "x"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := removeRunAgainFile(); err == nil {
		t.Fatalf("expected removeRunAgainFile error for non-empty dir")
	}
}

func TestDefaultFatalHooks(t *testing.T) {
	setupInitfilesTempEnv(t)
	initfilesExitFn = func(int) { panic("exit") }
	mustPanic(t, func() { initfilesFatalErrMsgfFn(errors.New("x"), "msg %s", "x") })
	mustPanic(t, func() { initfilesFatalErrMsgFn(errors.New("x"), "msg") })

	ageSopsExitFn = func(int) { panic("exit") }
	mustPanic(t, func() { ageSopsFatalErrMsgFn(errors.New("x"), "msg") })

	clusterenvExitFn = func(int) { panic("exit") }
	mustPanic(t, func() { clusterenvFatalMsgFn("msg") })
}

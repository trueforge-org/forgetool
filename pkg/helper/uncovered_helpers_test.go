package helper

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestHelperProcessExitPaths(t *testing.T) {
	mode := os.Getenv("GO_WANT_HELPER_EXIT_TEST")
	switch mode {
	case "dns-fail":
		checkAllDomains([]string{"invalid.invalid.invalid"}, false)
		os.Exit(0)
	case "time":
		_ = CheckSystemTime()
		os.Exit(0)
	default:
		return
	}
}

func TestCheckDNSResolutionAndCheckAllDomainsSuccess(t *testing.T) {
	if !checkDNSResolution("localhost") {
		t.Fatalf("expected localhost to resolve")
	}
	if checkDNSResolution("invalid.invalid.invalid") {
		t.Fatalf("expected invalid domain not to resolve")
	}
	checkAllDomains([]string{"localhost"}, false)
	checkAllDomains([]string{"localhost"}, true)
}

func TestCheckAllDomainsFailureSubprocess(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcessExitPaths")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_EXIT_TEST=dns-fail")
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected non-zero exit for failing DNS")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() == 0 {
		t.Fatalf("expected non-zero exit code, got %v", err)
	}
}

func TestCheckSystemTimeSubprocess(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcessExitPaths")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_EXIT_TEST=time")
	err := cmd.Run()
	if err == nil {
		return
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("unexpected error type: %T", err)
	}
	if exitErr.ExitCode() != 1 {
		t.Fatalf("expected exit code 0 or 1, got %d", exitErr.ExitCode())
	}
}

func TestGetYesOrNoAndVarToFileAndYamlEncoder(t *testing.T) {
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe create failed: %v", err)
	}
	os.Stdin = inR
	if _, err := inW.Write([]byte("maybe\ny\n")); err != nil {
		t.Fatalf("stdin write failed: %v", err)
	}
	_ = inW.Close()

	if !GetYesOrNo("continue? ", false) {
		t.Fatalf("expected yes after invalid then y input")
	}

	inR2, inW2, err := os.Pipe()
	if err != nil {
		t.Fatalf("second pipe create failed: %v", err)
	}
	os.Stdin = inR2
	if _, err := inW2.Write([]byte("n\n")); err != nil {
		t.Fatalf("stdin2 write failed: %v", err)
	}
	_ = inW2.Close()
	if GetYesOrNo("continue? ", false) {
		t.Fatalf("expected no for n input")
	}

	f := filepath.Join(t.TempDir(), "v.txt")
	if err := VarToFile(f, "abc"); err != nil {
		t.Fatalf("VarToFile create failed: %v", err)
	}
	if err := VarToFile(f, "changed"); err != nil {
		t.Fatalf("VarToFile existing file should still succeed: %v", err)
	}
	b, err := os.ReadFile(f)
	if err != nil {
		t.Fatalf("read VarToFile output failed: %v", err)
	}
	if string(b) != "abc" {
		t.Fatalf("existing file should not be overwritten, got %q", string(b))
	}

	var buf bytes.Buffer
	enc := YamlNewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(map[string]interface{}{"a": 1}); err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	if !strings.Contains(buf.String(), "a: 1") {
		t.Fatalf("unexpected yaml output: %q", buf.String())
	}
}

func TestExtractHelpersAndIPHostnameMap(t *testing.T) {
	cmd := "talosctl apply machineconfig -n 10.0.0.3"
	if got := ExtractNode(cmd); got != "10.0.0.3" {
		t.Fatalf("ExtractNode -n failed: %s", got)
	}
	if got := ExtractNode("talosctl --nodes=10.0.0.9 version"); got != "10.0.0.9" {
		t.Fatalf("ExtractNode --nodes failed: %s", got)
	}
	if got := ExtractSchematic("talosctl upgrade --image=factory.talos.dev/installer/abcd1234:v1.8.0"); got != "abcd1234" {
		t.Fatalf("ExtractSchematic failed: %s", got)
	}

	td := t.TempDir()
	oldClusterPath := ClusterPath
	oldTalEnv := TalEnv
	ClusterPath = td
	TalEnv = map[string]string{}
	t.Cleanup(func() {
		ClusterPath = oldClusterPath
		TalEnv = oldTalEnv
	})

	cfgPath := filepath.Join(td, "talos", "talconfig.yaml")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
		t.Fatalf("mkdir talos dir failed: %v", err)
	}
	content := "nodes:\n  - hostname: cp1\n    ipAddress: 10.0.0.10\n  - hostname: wk1\n    ipAddress: 10.0.0.11\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("write talconfig failed: %v", err)
	}

	m, err := CreateIPHostnameMap()
	if err != nil {
		t.Fatalf("CreateIPHostnameMap failed: %v", err)
	}
	if m["10.0.0.10"] != "cp1" || m["10.0.0.11"] != "wk1" {
		t.Fatalf("unexpected map content: %+v", m)
	}
}

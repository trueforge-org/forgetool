package kubectlcmds

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestGetKubeClient_NoKubeconfig(t *testing.T) {
	forceNoKubeconfigEnv(t)

	_, err := getKubeClient()
	if err == nil {
		t.Fatal("expected error when no kubeconfig is available, got nil")
	}
	if !strings.Contains(err.Error(), "kubeconfig") {
		t.Fatalf("expected kubeconfig-related error, got: %s", err.Error())
	}
}

func TestGetClientset_NoKubeconfig(t *testing.T) {
	forceNoKubeconfigEnv(t)

	_, err := GetClientset()
	if err == nil {
		t.Fatal("expected error when no kubeconfig is available, got nil")
	}
}

func forceNoKubeconfigEnv(t *testing.T) {
	t.Helper()

	home := t.TempDir()
	nonexistent := home + "/nonexistent-kubeconfig-" + strconv.FormatInt(time.Now().UnixNano(), 10)

	t.Setenv("HOME", home)
	t.Setenv("KUBECONFIG", nonexistent)
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	t.Setenv("KUBERNETES_SERVICE_PORT", "")
}

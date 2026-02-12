package kubectlcmds

import (
	"strings"
	"testing"
)

func TestCheckStatus_NoKubeconfig(t *testing.T) {
	forceNoKubeconfigEnv(t)

	err := CheckStatus([]string{"some-pod"}, nil, 1)
	if err == nil {
		t.Fatal("expected error when no kubeconfig is available, got nil")
	}
	if !strings.Contains(err.Error(), "kubeconfig") {
		t.Fatalf("expected kubeconfig-related error, got: %s", err.Error())
	}
}

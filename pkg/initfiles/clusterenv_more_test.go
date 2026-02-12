package initfiles

import (
	"os"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

func TestClusterNameEnvAndPostProcess(t *testing.T) {
	oldName := helper.ClusterName
	oldEnv := helper.TalEnv
	helper.ClusterName = "demo"
	helper.TalEnv = map[string]string{
		"VIP":       "10.0.0.100/24",
		"Master1IP": "10.0.0.10",
		"PODNET":    "10.244.0.0/16",
		"SVCNET":    "10.96.0.0/12",
	}
	t.Cleanup(func() {
		helper.ClusterName = oldName
		helper.TalEnv = oldEnv
	})

	clusterName()
	if helper.TalEnv["CLUSTERNAME"] != "demo" {
		t.Fatalf("expected CLUSTERNAME to be set")
	}

	PostProcessTalEnv()
	if helper.TalEnv["VIP_IP"] != "10.0.0.100" || helper.TalEnv["VIP_NETMASK"] != "24" {
		t.Fatalf("expected VIP split fields, got: %+v", helper.TalEnv)
	}
	if helper.TalEnv["Master1IP"] != "10.0.0.10/24" {
		t.Fatalf("expected Master1IP normalized with /24, got %s", helper.TalEnv["Master1IP"])
	}

	clusterEnvtoEnv()
	if got := os.Getenv("CLUSTERNAME"); got != "demo" {
		t.Fatalf("expected CLUSTERNAME env var to be set, got %q", got)
	}
}

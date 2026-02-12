package initfiles

import (
	"os"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

func TestSplitIPandNetmask(t *testing.T) {
	ip, mask, err := splitIPandNetmask("10.0.0.1/16")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ip != "10.0.0.1" || mask != "16" {
		t.Fatalf("unexpected split: %s/%s", ip, mask)
	}

	ip2, mask2, err := splitIPandNetmask("10.0.0.2")
	if err != nil {
		t.Fatalf("expected no error for plain IP, got %v", err)
	}
	if ip2 != "10.0.0.2" || mask2 != "24" {
		t.Fatalf("unexpected default mask split: %s/%s", ip2, mask2)
	}

	if _, _, err := splitIPandNetmask("notanip"); err == nil {
		t.Fatalf("expected error for invalid ip, got nil")
	}
}

func TestNormalizeIP(t *testing.T) {
	o, err := normalizeIP("10.1.2.3")
	if err != nil {
		t.Fatalf("normalizeIP failed: %v", err)
	}
	if o != "10.1.2.3/24" {
		t.Fatalf("unexpected normalized ip: %s", o)
	}

	o2, err := normalizeIP("10.1.2.0/16")
	if err != nil {
		t.Fatalf("normalizeIP failed: %v", err)
	}
	if o2 != "10.1.2.0/16" {
		t.Fatalf("unexpected normalized ip with mask: %s", o2)
	}

	if _, err := normalizeIP("notip"); err == nil {
		t.Fatalf("expected error for invalid ip")
	}
}

func TestNormalizeIPNetmask(t *testing.T) {
	if _, err := normalizeIPNetmask("10.0.0.0/8"); err != nil {
		t.Fatalf("expected valid cidr: %v", err)
	}
	if _, err := normalizeIPNetmask("notcidr"); err == nil {
		t.Fatalf("expected error for invalid cidr")
	}
}

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

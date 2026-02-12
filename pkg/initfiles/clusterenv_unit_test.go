package initfiles

import (
	"os"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

func TestValidateAndNormalizeIPsInTalEnv(t *testing.T) {
	oldEnv := helper.TalEnv
	t.Cleanup(func() { helper.TalEnv = oldEnv })

	t.Run("normalizes plain IP with default mask", func(t *testing.T) {
		helper.TalEnv = map[string]string{
			"Master1IP": "192.168.1.10",
		}
		ValidateAndNormalizeIPsInTalEnv()
		if helper.TalEnv["Master1IP"] != "192.168.1.10/24" {
			t.Fatalf("expected 192.168.1.10/24, got %s", helper.TalEnv["Master1IP"])
		}
	})

	t.Run("preserves existing CIDR", func(t *testing.T) {
		helper.TalEnv = map[string]string{
			"Master1IP": "10.0.0.5/16",
		}
		ValidateAndNormalizeIPsInTalEnv()
		if helper.TalEnv["Master1IP"] != "10.0.0.5/16" {
			t.Fatalf("expected 10.0.0.5/16, got %s", helper.TalEnv["Master1IP"])
		}
	})

	t.Run("skips missing key", func(t *testing.T) {
		helper.TalEnv = map[string]string{
			"OTHER_KEY": "somevalue",
		}
		ValidateAndNormalizeIPsInTalEnv()
		if _, exists := helper.TalEnv["Master1IP"]; exists {
			t.Fatal("Master1IP should not exist")
		}
	})

	t.Run("skips invalid IP without modifying", func(t *testing.T) {
		helper.TalEnv = map[string]string{
			"Master1IP": "not-an-ip",
		}
		ValidateAndNormalizeIPsInTalEnv()
		// Invalid IPs are skipped (logged but not modified)
		if helper.TalEnv["Master1IP"] != "not-an-ip" {
			t.Fatalf("expected invalid IP to remain unchanged, got %s", helper.TalEnv["Master1IP"])
		}
	})
}

func TestValidateAndNormalizeIPNetmaskVarsInTalEnv(t *testing.T) {
	oldEnv := helper.TalEnv
	t.Cleanup(func() { helper.TalEnv = oldEnv })

	t.Run("valid PODNET and SVCNET unchanged", func(t *testing.T) {
		helper.TalEnv = map[string]string{
			"PODNET": "10.244.0.0/16",
			"SVCNET": "10.96.0.0/12",
		}
		ValidateAndNormalizeIPNetmaskVarsInTalEnv()
		if helper.TalEnv["PODNET"] != "10.244.0.0/16" {
			t.Fatalf("expected PODNET 10.244.0.0/16, got %s", helper.TalEnv["PODNET"])
		}
		if helper.TalEnv["SVCNET"] != "10.96.0.0/12" {
			t.Fatalf("expected SVCNET 10.96.0.0/12, got %s", helper.TalEnv["SVCNET"])
		}
	})

	t.Run("skips missing keys", func(t *testing.T) {
		helper.TalEnv = map[string]string{}
		ValidateAndNormalizeIPNetmaskVarsInTalEnv()
		if len(helper.TalEnv) != 0 {
			t.Fatalf("expected empty TalEnv, got %+v", helper.TalEnv)
		}
	})

	t.Run("invalid CIDR leaves value unchanged", func(t *testing.T) {
		helper.TalEnv = map[string]string{
			"PODNET": "invalid-cidr",
		}
		ValidateAndNormalizeIPNetmaskVarsInTalEnv()
		// Invalid values are skipped, original value remains
		if helper.TalEnv["PODNET"] != "invalid-cidr" {
			t.Fatalf("expected invalid CIDR to remain unchanged, got %s", helper.TalEnv["PODNET"])
		}
	})

	t.Run("only SVCNET present", func(t *testing.T) {
		helper.TalEnv = map[string]string{
			"SVCNET": "172.16.0.0/16",
		}
		ValidateAndNormalizeIPNetmaskVarsInTalEnv()
		if helper.TalEnv["SVCNET"] != "172.16.0.0/16" {
			t.Fatalf("expected SVCNET 172.16.0.0/16, got %s", helper.TalEnv["SVCNET"])
		}
	})
}

func TestClusterEnvtoEnv(t *testing.T) {
	oldEnv := helper.TalEnv
	t.Cleanup(func() {
		helper.TalEnv = oldEnv
		os.Unsetenv("TEST_UNIT_KEY1")
		os.Unsetenv("TEST_UNIT_KEY2")
	})

	helper.TalEnv = map[string]string{
		"TEST_UNIT_KEY1": "value1",
		"TEST_UNIT_KEY2": "value2",
	}
	clusterEnvtoEnv()

	if got := os.Getenv("TEST_UNIT_KEY1"); got != "value1" {
		t.Fatalf("expected TEST_UNIT_KEY1=value1, got %q", got)
	}
	if got := os.Getenv("TEST_UNIT_KEY2"); got != "value2" {
		t.Fatalf("expected TEST_UNIT_KEY2=value2, got %q", got)
	}
}

func TestSplitIPandNetmaskEdgeCases(t *testing.T) {
	t.Run("invalid netmask value", func(t *testing.T) {
		_, _, err := splitIPandNetmask("10.0.0.1/99")
		if err == nil {
			t.Fatal("expected error for invalid netmask /99")
		}
	})

	t.Run("IPv6 address", func(t *testing.T) {
		ip, mask, err := splitIPandNetmask("::1")
		if err != nil {
			t.Fatalf("expected no error for IPv6, got %v", err)
		}
		if ip != "::1" || mask != "24" {
			t.Fatalf("unexpected result for IPv6: ip=%s mask=%s", ip, mask)
		}
	})

	t.Run("IPv6 CIDR", func(t *testing.T) {
		ip, mask, err := splitIPandNetmask("fe80::1/64")
		if err != nil {
			t.Fatalf("expected no error for IPv6 CIDR, got %v", err)
		}
		if ip != "fe80::1" || mask != "64" {
			t.Fatalf("unexpected result for IPv6 CIDR: ip=%s mask=%s", ip, mask)
		}
	})

	t.Run("empty string", func(t *testing.T) {
		_, _, err := splitIPandNetmask("")
		if err == nil {
			t.Fatal("expected error for empty string")
		}
	})

	t.Run("slash but invalid parts", func(t *testing.T) {
		_, _, err := splitIPandNetmask("abc/def")
		if err == nil {
			t.Fatal("expected error for abc/def")
		}
	})
}

func TestNormalizeIPEdgeCases(t *testing.T) {
	t.Run("invalid CIDR mask", func(t *testing.T) {
		_, err := normalizeIP("10.0.0.1/99")
		if err == nil {
			t.Fatal("expected error for invalid mask /99")
		}
	})

	t.Run("empty string", func(t *testing.T) {
		_, err := normalizeIP("")
		if err == nil {
			t.Fatal("expected error for empty string")
		}
	})

	t.Run("IPv6 plain", func(t *testing.T) {
		result, err := normalizeIP("::1")
		if err != nil {
			t.Fatalf("expected no error for IPv6, got %v", err)
		}
		if result != "::1/24" {
			t.Fatalf("expected ::1/24, got %s", result)
		}
	})
}

func TestNormalizeIPNetmaskEdgeCases(t *testing.T) {
	t.Run("valid /32", func(t *testing.T) {
		result, err := normalizeIPNetmask("10.0.0.1/32")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result != "10.0.0.1/32" {
			t.Fatalf("expected 10.0.0.1/32, got %s", result)
		}
	})

	t.Run("valid /0", func(t *testing.T) {
		result, err := normalizeIPNetmask("0.0.0.0/0")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result != "0.0.0.0/0" {
			t.Fatalf("expected 0.0.0.0/0, got %s", result)
		}
	})

	t.Run("empty string", func(t *testing.T) {
		_, err := normalizeIPNetmask("")
		if err == nil {
			t.Fatal("expected error for empty string")
		}
	})

	t.Run("IP without mask", func(t *testing.T) {
		_, err := normalizeIPNetmask("10.0.0.1")
		if err == nil {
			t.Fatal("expected error for IP without mask")
		}
	})
}

func TestPostProcessTalEnvSplitsAndNormalizes(t *testing.T) {
	oldEnv := helper.TalEnv
	t.Cleanup(func() { helper.TalEnv = oldEnv })

	t.Run("splits CIDR values into IP, NETMASK, and CIDR keys", func(t *testing.T) {
		helper.TalEnv = map[string]string{
			"GATEWAY":   "192.168.1.1/24",
			"Master1IP": "192.168.1.10",
			"PODNET":    "10.244.0.0/16",
			"SVCNET":    "10.96.0.0/12",
		}
		PostProcessTalEnv()

		if helper.TalEnv["GATEWAY_IP"] != "192.168.1.1" {
			t.Fatalf("expected GATEWAY_IP=192.168.1.1, got %s", helper.TalEnv["GATEWAY_IP"])
		}
		if helper.TalEnv["GATEWAY_NETMASK"] != "24" {
			t.Fatalf("expected GATEWAY_NETMASK=24, got %s", helper.TalEnv["GATEWAY_NETMASK"])
		}
		if helper.TalEnv["GATEWAY_CIDR"] != "192.168.1.1/24" {
			t.Fatalf("expected GATEWAY_CIDR=192.168.1.1/24, got %s", helper.TalEnv["GATEWAY_CIDR"])
		}
		// Master1IP should be normalized to include /24
		if helper.TalEnv["Master1IP"] != "192.168.1.10/24" {
			t.Fatalf("expected Master1IP=192.168.1.10/24, got %s", helper.TalEnv["Master1IP"])
		}
	})

	t.Run("non-IP values are left unchanged", func(t *testing.T) {
		helper.TalEnv = map[string]string{
			"CLUSTERNAME": "testcluster",
			"SOME_VALUE":  "hello-world",
		}
		PostProcessTalEnv()

		if helper.TalEnv["CLUSTERNAME"] != "testcluster" {
			t.Fatalf("expected CLUSTERNAME unchanged, got %s", helper.TalEnv["CLUSTERNAME"])
		}
		if helper.TalEnv["SOME_VALUE"] != "hello-world" {
			t.Fatalf("expected SOME_VALUE unchanged, got %s", helper.TalEnv["SOME_VALUE"])
		}
	})
}

func TestClusterNameSetsFromHelper(t *testing.T) {
	oldName := helper.ClusterName
	oldEnv := helper.TalEnv
	t.Cleanup(func() {
		helper.ClusterName = oldName
		helper.TalEnv = oldEnv
	})

	helper.TalEnv = map[string]string{}
	helper.ClusterName = "unit-test-cluster"
	clusterName()

	if helper.TalEnv["CLUSTERNAME"] != "unit-test-cluster" {
		t.Fatalf("expected CLUSTERNAME=unit-test-cluster, got %s", helper.TalEnv["CLUSTERNAME"])
	}
}

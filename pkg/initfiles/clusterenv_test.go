package initfiles

import "testing"

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

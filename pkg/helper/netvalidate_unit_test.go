package helper

import (
	"testing"
)

func TestIPInRangeBoundary(t *testing.T) {
	tests := []struct {
		ip       string
		ipRange  string
		expected bool
	}{
		{"192.168.1.1", "192.168.1.1-192.168.1.10", true},   // start boundary
		{"192.168.1.10", "192.168.1.1-192.168.1.10", true},  // end boundary
		{"192.168.1.5", "192.168.1.1-192.168.1.10", true},   // middle
		{"192.168.1.0", "192.168.1.1-192.168.1.10", false},  // below range
		{"192.168.1.11", "192.168.1.1-192.168.1.10", false}, // above range
	}

	for _, tt := range tests {
		got, err := IPInRange(tt.ip, tt.ipRange)
		if err != nil {
			t.Fatalf("IPInRange(%s, %s) error: %v", tt.ip, tt.ipRange, err)
		}
		if got != tt.expected {
			t.Errorf("IPInRange(%s, %s) = %v, want %v", tt.ip, tt.ipRange, got, tt.expected)
		}
	}
}

func TestIPInRangeInvalidIP(t *testing.T) {
	got, err := IPInRange("notanip", "192.168.1.1-192.168.1.10")
	if err != nil {
		t.Fatalf("expected no error for invalid IP, got %v", err)
	}
	if got {
		t.Fatalf("expected false for invalid IP")
	}
}

func TestIPInRangeInvalidRange(t *testing.T) {
	got, err := IPInRange("192.168.1.1", "invalid-range-format")
	if err != nil {
		t.Fatalf("expected no error for invalid range, got %v", err)
	}
	if got {
		t.Fatalf("expected false for invalid range format")
	}
}

func TestIPInRangeSingleDash(t *testing.T) {
	got, err := IPInRange("192.168.1.5", "192.168.1.1-192.168.1.10-extra")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got {
		t.Fatalf("expected false for range with too many parts")
	}
}

func TestCIDROverlapNonOverlapping(t *testing.T) {
	overlap, err := CIDROverlap("10.0.0.0/24", "192.168.0.0/24")
	if err != nil {
		t.Fatalf("CIDROverlap error: %v", err)
	}
	if overlap {
		t.Fatalf("expected no overlap between 10.0.0.0/24 and 192.168.0.0/24")
	}
}

func TestCIDROverlapOverlapping(t *testing.T) {
	overlap, err := CIDROverlap("10.0.0.0/8", "10.0.1.0/24")
	if err != nil {
		t.Fatalf("CIDROverlap error: %v", err)
	}
	if !overlap {
		t.Fatalf("expected overlap between 10.0.0.0/8 and 10.0.1.0/24")
	}
}

func TestCIDROverlapInvalidCIDR(t *testing.T) {
	_, err := CIDROverlap("invalid", "192.168.0.0/24")
	if err == nil {
		t.Fatalf("expected error for invalid CIDR")
	}
}

func TestIPInCIDRContained(t *testing.T) {
	got, err := IPInCIDR("10.0.0.5", "10.0.0.0/24")
	if err != nil {
		t.Fatalf("IPInCIDR error: %v", err)
	}
	if !got {
		t.Fatalf("expected 10.0.0.5 to be in 10.0.0.0/24")
	}
}

func TestIPInCIDRNotContained(t *testing.T) {
	got, err := IPInCIDR("192.168.1.1", "10.0.0.0/24")
	if err != nil {
		t.Fatalf("IPInCIDR error: %v", err)
	}
	if got {
		t.Fatalf("expected 192.168.1.1 to not be in 10.0.0.0/24")
	}
}

func TestIPInCIDRInvalidIP(t *testing.T) {
	got, err := IPInCIDR("notanip", "10.0.0.0/24")
	if err != nil {
		t.Fatalf("expected no error for invalid IP, got %v", err)
	}
	if got {
		t.Fatalf("expected false for invalid IP")
	}
}

func TestIPInCIDRInvalidCIDR(t *testing.T) {
	_, err := IPInCIDR("10.0.0.1", "invalid")
	if err == nil {
		t.Fatalf("expected error for invalid CIDR")
	}
}

func TestBytesCompareEqual(t *testing.T) {
	a := []byte{10, 0, 0, 1}
	b := []byte{10, 0, 0, 1}
	if bytesCompare(a, b) != 0 {
		t.Fatalf("expected equal IPs to return 0")
	}
}

func TestBytesCompareLess(t *testing.T) {
	a := []byte{10, 0, 0, 1}
	b := []byte{10, 0, 0, 2}
	if bytesCompare(a, b) != -1 {
		t.Fatalf("expected a < b to return -1")
	}
}

func TestBytesCompareGreater(t *testing.T) {
	a := []byte{10, 0, 0, 2}
	b := []byte{10, 0, 0, 1}
	if bytesCompare(a, b) != 1 {
		t.Fatalf("expected a > b to return 1")
	}
}

package helper

import (
	"strings"
	"testing"
)

func TestCheckIPorCIDRNotInCIDR_NotInCIDR(t *testing.T) {
	err := CheckIPorCIDRNotInCIDR("192.168.1.1", "10.0.0.0/8", "testIP", "testCIDR")
	if err != nil {
		t.Fatalf("expected no error when IP is outside CIDR, got: %v", err)
	}
}

func TestCheckIPorCIDRNotInCIDR_InCIDR(t *testing.T) {
	err := CheckIPorCIDRNotInCIDR("10.0.0.1", "10.0.0.0/8", "testIP", "testCIDR")
	if err == nil {
		t.Fatal("expected error when IP is inside CIDR, got nil")
	}
	if !strings.Contains(err.Error(), "cannot proceed") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestCheckIPorCIDRNotInCIDR_InvalidCIDR(t *testing.T) {
	err := CheckIPorCIDRNotInCIDR("10.0.0.1", "not-a-cidr", "testIP", "testCIDR")
	if err == nil {
		t.Fatal("expected error for invalid CIDR, got nil")
	}
	if !strings.Contains(err.Error(), "error validating") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestCheckIPorCIDRNotInCIDR_InvalidIP(t *testing.T) {
	// net.ParseIP returns nil for invalid IP, so IPInCIDR returns false, nil
	err := CheckIPorCIDRNotInCIDR("not-an-ip", "10.0.0.0/8", "testIP", "testCIDR")
	if err != nil {
		t.Fatalf("expected no error for invalid IP (ParseIP returns nil), got: %v", err)
	}
}

func TestCheckRangeNotInCIDR_NotInCIDR(t *testing.T) {
	err := CheckRangeNotInCIDR("192.168.1.1-192.168.1.10", "10.0.0.0/8", "testRange", "testCIDR")
	if err != nil {
		t.Fatalf("expected no error when range is outside CIDR, got: %v", err)
	}
}

func TestCheckRangeNotInCIDR_InCIDR(t *testing.T) {
	err := CheckRangeNotInCIDR("10.0.0.1-10.0.0.5", "10.0.0.0/8", "testRange", "testCIDR")
	if err == nil {
		t.Fatal("expected error when range is inside CIDR, got nil")
	}
	if !strings.Contains(err.Error(), "cannot proceed") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestCheckRangeNotInCIDR_PartialOverlap(t *testing.T) {
	err := CheckRangeNotInCIDR("10.0.0.1-192.168.1.1", "10.0.0.0/8", "testRange", "testCIDR")
	if err == nil {
		t.Fatal("expected error when range start is inside CIDR, got nil")
	}
}

func TestCheckRangeNotInCIDR_InvalidFormat(t *testing.T) {
	err := CheckRangeNotInCIDR("not-a-range", "10.0.0.0/8", "testRange", "testCIDR")
	if err == nil {
		t.Fatal("expected error for invalid range format, got nil")
	}
	if !strings.Contains(err.Error(), "invalid range format") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestCheckRangeNotInCIDR_InvalidCIDR(t *testing.T) {
	err := CheckRangeNotInCIDR("192.168.1.1-192.168.1.10", "invalid-cidr", "testRange", "testCIDR")
	if err == nil {
		t.Fatal("expected error for invalid CIDR, got nil")
	}
	if !strings.Contains(err.Error(), "error validating") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestCheckRangeNotInCIDR_ThreeParts(t *testing.T) {
	err := CheckRangeNotInCIDR("10.0.0.1-10.0.0.5-10.0.0.9", "10.0.0.0/8", "testRange", "testCIDR")
	if err == nil {
		t.Fatal("expected error for three-part range, got nil")
	}
}

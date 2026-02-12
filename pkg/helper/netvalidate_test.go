package helper

import (
	"net"
	"os"
	"os/exec"
	"testing"
)

func TestCIDROverlap(t *testing.T) {
	tests := []struct {
		a, b    string
		want    bool
		wantErr bool
	}{
		{"10.0.0.0/8", "10.1.0.0/16", true, false},
		{"192.168.1.0/24", "10.0.0.0/8", false, false},
		{"notcidr", "alsonot", false, true},
	}

	for _, tt := range tests {
		got, err := CIDROverlap(tt.a, tt.b)
		if (err != nil) != tt.wantErr {
			t.Fatalf("CIDROverlap(%q,%q) err = %v, wantErr=%v", tt.a, tt.b, err, tt.wantErr)
		}
		if got != tt.want {
			t.Fatalf("CIDROverlap(%q,%q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestIPInCIDR(t *testing.T) {
	tests := []struct {
		ip, cidr string
		want     bool
	}{
		{"10.0.0.1", "10.0.0.0/8", true},
		{"192.168.2.1", "192.168.1.0/24", false},
		{"notip", "10.0.0.0/8", false},
	}

	for _, tt := range tests {
		got, err := IPInCIDR(tt.ip, tt.cidr)
		if err != nil {
			t.Fatalf("IPInCIDR(%q,%q) returned error: %v", tt.ip, tt.cidr, err)
		}
		if got != tt.want {
			t.Fatalf("IPInCIDR(%q,%q) = %v, want %v", tt.ip, tt.cidr, got, tt.want)
		}
	}
}

func TestIPInRange(t *testing.T) {
	tests := []struct {
		ip, r string
		want  bool
	}{
		{"10.0.0.5", "10.0.0.1-10.0.0.10", true},
		{"10.0.0.11", "10.0.0.1-10.0.0.10", false},
		{"10.0.0.5", "badrange", false},
		{"notip", "10.0.0.1-10.0.0.10", false},
	}

	for _, tt := range tests {
		got, err := IPInRange(tt.ip, tt.r)
		if err != nil {
			t.Fatalf("IPInRange(%q,%q) returned error: %v", tt.ip, tt.r, err)
		}
		if got != tt.want {
			t.Fatalf("IPInRange(%q,%q) = %v, want %v", tt.ip, tt.r, got, tt.want)
		}
	}
}

func TestBytesCompare_Equal(t *testing.T) {
	a := net.ParseIP("10.0.0.1").To4()
	b := net.ParseIP("10.0.0.1").To4()
	if got := bytesCompare(a, b); got != 0 {
		t.Fatalf("bytesCompare(%v, %v) = %d, want 0", a, b, got)
	}
}

func TestBytesCompare_LessThan(t *testing.T) {
	a := net.ParseIP("10.0.0.1").To4()
	b := net.ParseIP("10.0.0.2").To4()
	if got := bytesCompare(a, b); got != -1 {
		t.Fatalf("bytesCompare(%v, %v) = %d, want -1", a, b, got)
	}
}

func TestBytesCompare_GreaterThan(t *testing.T) {
	a := net.ParseIP("10.0.0.2").To4()
	b := net.ParseIP("10.0.0.1").To4()
	if got := bytesCompare(a, b); got != 1 {
		t.Fatalf("bytesCompare(%v, %v) = %d, want 1", a, b, got)
	}
}

func TestValidateIPorCIDRNotInCIDR_Passes(t *testing.T) {
	if os.Getenv("TEST_SUBPROCESS") == "1" {
		ValidateIPorCIDRNotInCIDR("192.168.1.1", "10.0.0.0/8", "testIP", "testCIDR")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestValidateIPorCIDRNotInCIDR_Passes$")
	cmd.Env = append(os.Environ(), "TEST_SUBPROCESS=1")
	if err := cmd.Run(); err != nil {
		t.Fatalf("expected no exit when IP is not in CIDR, got: %v", err)
	}
}

func TestValidateIPorCIDRNotInCIDR_ExitsWhenInCIDR(t *testing.T) {
	if os.Getenv("TEST_SUBPROCESS") == "1" {
		ValidateIPorCIDRNotInCIDR("10.0.0.1", "10.0.0.0/8", "testIP", "testCIDR")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestValidateIPorCIDRNotInCIDR_ExitsWhenInCIDR$")
	cmd.Env = append(os.Environ(), "TEST_SUBPROCESS=1")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected os.Exit when IP is in CIDR, but process succeeded")
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() == 0 {
			t.Fatal("expected non-zero exit code")
		}
	}
}

func TestValidateRangeNotInCIDR_Passes(t *testing.T) {
	if os.Getenv("TEST_SUBPROCESS") == "1" {
		ValidateRangeNotInCIDR("192.168.1.1-192.168.1.10", "10.0.0.0/8", "testRange", "testCIDR")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestValidateRangeNotInCIDR_Passes$")
	cmd.Env = append(os.Environ(), "TEST_SUBPROCESS=1")
	if err := cmd.Run(); err != nil {
		t.Fatalf("expected no exit when range is not in CIDR, got: %v", err)
	}
}

func TestValidateRangeNotInCIDR_ExitsOnInvalidFormat(t *testing.T) {
	if os.Getenv("TEST_SUBPROCESS") == "1" {
		ValidateRangeNotInCIDR("notarange", "10.0.0.0/8", "testRange", "testCIDR")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestValidateRangeNotInCIDR_ExitsOnInvalidFormat$")
	cmd.Env = append(os.Environ(), "TEST_SUBPROCESS=1")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected os.Exit for invalid range format, but process succeeded")
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() == 0 {
			t.Fatal("expected non-zero exit code")
		}
	}
}

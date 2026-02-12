package helper

import (
	"net"
	"os"
	"os/exec"
	"testing"
)

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

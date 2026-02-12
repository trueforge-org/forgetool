package helper

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
)

// CIDROverlap checks if two CIDR annotations overlap.
func CIDROverlap(cidr1, cidr2 string) (bool, error) {
	_, ipnet1, err1 := net.ParseCIDR(cidr1)
	_, ipnet2, err2 := net.ParseCIDR(cidr2)

	if err1 != nil || err2 != nil {
		return false, err1
	}

	return ipnet1.Contains(ipnet2.IP) || ipnet2.Contains(ipnet1.IP), nil
}

// IPInCIDR checks if an IP fits into a CIDR.
func IPInCIDR(ipStr, cidr string) (bool, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false, nil
	}

	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false, err
	}

	return ipnet.Contains(ip), nil
}

// IPInRange checks if an IP fits into an IP range (ip-ip).
func IPInRange(ipStr, rangeStr string) (bool, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false, nil
	}

	parts := strings.Split(rangeStr, "-")
	if len(parts) != 2 {
		return false, nil
	}

	startIP := net.ParseIP(parts[0])
	endIP := net.ParseIP(parts[1])
	if startIP == nil || endIP == nil {
		return false, nil
	}

	return bytesCompare(ip, startIP) >= 0 && bytesCompare(ip, endIP) <= 0, nil
}

// bytesCompare compares two IP addresses.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func bytesCompare(a, b net.IP) int {
	for i := range a {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

// CheckIPorCIDRNotInCIDR checks whether ipOrCIDR is inside cidr.
// Returns an error describing the problem if the IP is in the CIDR or if
// validation fails. Returns nil when the IP is safely outside the CIDR.
func CheckIPorCIDRNotInCIDR(ipOrCIDR, cidr, ipOrCIDRName, cidrName string) error {
	inCIDR, err := IPInCIDR(ipOrCIDR, cidr)
	if err != nil {
		return fmt.Errorf("error validating %s against %s: %v", ipOrCIDRName, cidrName, err)
	}
	if inCIDR {
		return fmt.Errorf("cannot proceed, %s cannot be in %s", ipOrCIDRName, cidrName)
	}
	return nil
}

func ValidateIPorCIDRNotInCIDR(ipOrCIDR, cidr, ipOrCIDRName, cidrName string) {
	if err := CheckIPorCIDRNotInCIDR(ipOrCIDR, cidr, ipOrCIDRName, cidrName); err != nil {
		log.Info().Msg(err.Error())
		os.Exit(1)
	}
}

// CheckRangeNotInCIDR checks whether any part of rangeStr falls inside cidr.
// Returns an error describing the problem or nil if the range is safely outside.
func CheckRangeNotInCIDR(rangeStr, cidr, rangeName, cidrName string) error {
	parts := strings.Split(rangeStr, "-")
	if len(parts) != 2 {
		return fmt.Errorf("invalid range format for %s", rangeName)
	}

	inCIDRStart, errStart := IPInCIDR(parts[0], cidr)
	inCIDREnd, errEnd := IPInCIDR(parts[1], cidr)

	if errStart != nil || errEnd != nil {
		return fmt.Errorf("error validating %s against %s: %v %v", rangeName, cidrName, errStart, errEnd)
	}

	if inCIDRStart || inCIDREnd {
		return fmt.Errorf("cannot proceed, %s cannot be in %s", rangeName, cidrName)
	}
	return nil
}

func ValidateRangeNotInCIDR(rangeStr, cidr, rangeName, cidrName string) {
	if err := CheckRangeNotInCIDR(rangeStr, cidr, rangeName, cidrName); err != nil {
		log.Info().Msg(err.Error())
		os.Exit(1)
	}
}

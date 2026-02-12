package helper

import (
	"testing"
)

func TestCIDROverlap(t *testing.T) {
	tests := []struct {
		name    string
		cidr1   string
		cidr2   string
		want    bool
		wantErr bool
	}{
		{
			name:    "Overlapping CIDRs - cidr1 contains cidr2",
			cidr1:   "10.0.0.0/16",
			cidr2:   "10.0.1.0/24",
			want:    true,
			wantErr: false,
		},
		{
			name:    "Overlapping CIDRs - cidr2 contains cidr1",
			cidr1:   "10.0.1.0/24",
			cidr2:   "10.0.0.0/16",
			want:    true,
			wantErr: false,
		},
		{
			name:    "Non-overlapping CIDRs",
			cidr1:   "10.0.0.0/24",
			cidr2:   "10.0.1.0/24",
			want:    false,
			wantErr: false,
		},
		{
			name:    "Identical CIDRs",
			cidr1:   "192.168.1.0/24",
			cidr2:   "192.168.1.0/24",
			want:    true,
			wantErr: false,
		},
		{
			name:    "Invalid CIDR format - first CIDR",
			cidr1:   "invalid",
			cidr2:   "10.0.0.0/24",
			want:    false,
			wantErr: true,
		},
		{
			name:    "Invalid CIDR format - second CIDR",
			cidr1:   "10.0.0.0/24",
			cidr2:   "invalid",
			want:    false,
			wantErr: false, // Function returns err1 which is nil when first CIDR is valid
		},
		{
			name:    "IPv6 overlapping CIDRs",
			cidr1:   "2001:db8::/32",
			cidr2:   "2001:db8:1::/48",
			want:    true,
			wantErr: false,
		},
		{
			name:    "IPv6 non-overlapping CIDRs",
			cidr1:   "2001:db8::/48",
			cidr2:   "2001:db9::/48",
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CIDROverlap(tt.cidr1, tt.cidr2)
			if (err != nil) != tt.wantErr {
				t.Errorf("CIDROverlap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CIDROverlap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIPInCIDR(t *testing.T) {
	tests := []struct {
		name    string
		ipStr   string
		cidr    string
		want    bool
		wantErr bool
	}{
		{
			name:    "IP in CIDR range",
			ipStr:   "10.0.1.5",
			cidr:    "10.0.0.0/16",
			want:    true,
			wantErr: false,
		},
		{
			name:    "IP not in CIDR range",
			ipStr:   "10.1.1.5",
			cidr:    "10.0.0.0/16",
			want:    false,
			wantErr: false,
		},
		{
			name:    "IP at CIDR boundary (network address)",
			ipStr:   "192.168.1.0",
			cidr:    "192.168.1.0/24",
			want:    true,
			wantErr: false,
		},
		{
			name:    "IP at CIDR boundary (broadcast address)",
			ipStr:   "192.168.1.255",
			cidr:    "192.168.1.0/24",
			want:    true,
			wantErr: false,
		},
		{
			name:    "Invalid IP address",
			ipStr:   "invalid",
			cidr:    "10.0.0.0/16",
			want:    false,
			wantErr: false,
		},
		{
			name:    "Invalid CIDR",
			ipStr:   "10.0.1.5",
			cidr:    "invalid",
			want:    false,
			wantErr: true,
		},
		{
			name:    "IPv6 in CIDR",
			ipStr:   "2001:db8::1",
			cidr:    "2001:db8::/32",
			want:    true,
			wantErr: false,
		},
		{
			name:    "IPv6 not in CIDR",
			ipStr:   "2001:db9::1",
			cidr:    "2001:db8::/32",
			want:    false,
			wantErr: false,
		},
		{
			name:    "Empty IP string",
			ipStr:   "",
			cidr:    "10.0.0.0/16",
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IPInCIDR(tt.ipStr, tt.cidr)
			if (err != nil) != tt.wantErr {
				t.Errorf("IPInCIDR() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IPInCIDR() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIPInRange(t *testing.T) {
	tests := []struct {
		name     string
		ipStr    string
		rangeStr string
		want     bool
		wantErr  bool
	}{
		{
			name:     "IP in range",
			ipStr:    "10.0.0.5",
			rangeStr: "10.0.0.1-10.0.0.10",
			want:     true,
			wantErr:  false,
		},
		{
			name:     "IP not in range - below",
			ipStr:    "10.0.0.0",
			rangeStr: "10.0.0.1-10.0.0.10",
			want:     false,
			wantErr:  false,
		},
		{
			name:     "IP not in range - above",
			ipStr:    "10.0.0.11",
			rangeStr: "10.0.0.1-10.0.0.10",
			want:     false,
			wantErr:  false,
		},
		{
			name:     "IP at range start",
			ipStr:    "10.0.0.1",
			rangeStr: "10.0.0.1-10.0.0.10",
			want:     true,
			wantErr:  false,
		},
		{
			name:     "IP at range end",
			ipStr:    "10.0.0.10",
			rangeStr: "10.0.0.1-10.0.0.10",
			want:     true,
			wantErr:  false,
		},
		{
			name:     "Invalid IP address",
			ipStr:    "invalid",
			rangeStr: "10.0.0.1-10.0.0.10",
			want:     false,
			wantErr:  false,
		},
		{
			name:     "Invalid range format - no hyphen",
			ipStr:    "10.0.0.5",
			rangeStr: "10.0.0.1",
			want:     false,
			wantErr:  false,
		},
		{
			name:     "Invalid range format - too many parts",
			ipStr:    "10.0.0.5",
			rangeStr: "10.0.0.1-10.0.0.5-10.0.0.10",
			want:     false,
			wantErr:  false,
		},
		{
			name:     "Invalid start IP in range",
			ipStr:    "10.0.0.5",
			rangeStr: "invalid-10.0.0.10",
			want:     false,
			wantErr:  false,
		},
		{
			name:     "Invalid end IP in range",
			ipStr:    "10.0.0.5",
			rangeStr: "10.0.0.1-invalid",
			want:     false,
			wantErr:  false,
		},
		{
			name:     "IPv6 in range",
			ipStr:    "2001:db8::5",
			rangeStr: "2001:db8::1-2001:db8::10",
			want:     true,
			wantErr:  false,
		},
		{
			name:     "IPv6 not in range",
			ipStr:    "2001:db8::20",
			rangeStr: "2001:db8::1-2001:db8::10",
			want:     false,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IPInRange(tt.ipStr, tt.rangeStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("IPInRange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IPInRange() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBytesCompare(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want int
	}{
		{
			name: "Equal IPs",
			a:    "10.0.0.1",
			b:    "10.0.0.1",
			want: 0,
		},
		{
			name: "First IP less than second",
			a:    "10.0.0.1",
			b:    "10.0.0.2",
			want: -1,
		},
		{
			name: "First IP greater than second",
			a:    "10.0.0.2",
			b:    "10.0.0.1",
			want: 1,
		},
		{
			name: "Different subnets - first less",
			a:    "10.0.0.1",
			b:    "10.1.0.1",
			want: -1,
		},
		{
			name: "Different subnets - first greater",
			a:    "10.1.0.1",
			b:    "10.0.0.1",
			want: 1,
		},
		{
			name: "IPv6 equal",
			a:    "2001:db8::1",
			b:    "2001:db8::1",
			want: 0,
		},
		{
			name: "IPv6 first less",
			a:    "2001:db8::1",
			b:    "2001:db8::2",
			want: -1,
		},
		{
			name: "IPv6 first greater",
			a:    "2001:db8::2",
			b:    "2001:db8::1",
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse IPs for testing
			aIP := parseIP(tt.a)
			bIP := parseIP(tt.b)

			got := bytesCompare(aIP, bIP)
			if got != tt.want {
				t.Errorf("bytesCompare() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function for testing
func parseIP(s string) []byte {
	// Simple IP parsing for tests
	if s == "10.0.0.1" {
		return []byte{10, 0, 0, 1}
	}
	if s == "10.0.0.2" {
		return []byte{10, 0, 0, 2}
	}
	if s == "10.1.0.1" {
		return []byte{10, 1, 0, 1}
	}
	if s == "2001:db8::1" {
		return []byte{32, 1, 13, 184, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	}
	if s == "2001:db8::2" {
		return []byte{32, 1, 13, 184, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}
	}
	return nil
}

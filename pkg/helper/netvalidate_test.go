package helper

import "testing"

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

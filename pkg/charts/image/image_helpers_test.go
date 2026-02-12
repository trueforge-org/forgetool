package image

import (
	"testing"
)

// TestCleanTagHelpers tests individual cleanup helper functions directly
func TestCleanRelease(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"RELEASE.2023-11-20T22-40-07Z", "2023.11.20"},
		{"RELEASE.2024-01-01T00-00-00Z", "2024.01.01"},
	}
	for _, tt := range tests {
		got := cleanRelease(tt.input)
		if got != tt.expect {
			t.Errorf("cleanRelease(%q) = %q, want %q", tt.input, got, tt.expect)
		}
	}
}

func TestCleanArch(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"x64-1.2.3", "1.2.3"},
		{"arm64-2.0.0", "2.0.0"},
	}
	for _, tt := range tests {
		got := cleanArch(tt.input)
		if got != tt.expect {
			t.Errorf("cleanArch(%q) = %q, want %q", tt.input, got, tt.expect)
		}
	}
}

func TestCleanYearMonthDay(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"2023-11-15", "2023.11.15"},
		{"2022-04", "2022.4.0"},
	}
	for _, tt := range tests {
		got := cleanYearMonthDay(tt.input)
		if got != tt.expect {
			t.Errorf("cleanYearMonthDay(%q) = %q, want %q", tt.input, got, tt.expect)
		}
	}
}

func TestCleanIncompleteSemVer(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"1.2", "1.2.0"},
		{"1", "1.0.0"},
		{"1.2.3", "1.2.3"},
		{"1.2.3.4", "1.2.3.4"},
	}
	for _, tt := range tests {
		got := cleanIncompleteSemVer(tt.input)
		if got != tt.expect {
			t.Errorf("cleanIncompleteSemVer(%q) = %q, want %q", tt.input, got, tt.expect)
		}
	}
}

func TestCleanLeadingSymbol(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"v1.2.3", "1.2.3"},
		{"V1.2.3", "1.2.3"},
		{"#1.2.3", "1.2.3"},
		{"1.2.3", "1.2.3"},
	}
	for _, tt := range tests {
		got := cleanLeadingSymbol(tt.input)
		if got != tt.expect {
			t.Errorf("cleanLeadingSymbol(%q) = %q, want %q", tt.input, got, tt.expect)
		}
	}
}

func TestCleanSha(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"v1.2.3@sha256:abc123", "v1.2.3"},
		{"1.0.0", "1.0.0"},
	}
	for _, tt := range tests {
		got := cleanSha(tt.input)
		if got != tt.expect {
			t.Errorf("cleanSha(%q) = %q, want %q", tt.input, got, tt.expect)
		}
	}
}

func TestCleanPrefixYearMonthDay(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"latest-2023-12-18", "2023.12.18"},
	}
	for _, tt := range tests {
		got := cleanPrefixYearMonthDay(tt.input)
		if got != tt.expect {
			t.Errorf("cleanPrefixYearMonthDay(%q) = %q, want %q", tt.input, got, tt.expect)
		}
	}
}

func TestKeepShortCommitHashSuffix(t *testing.T) {
	got := keepShortCommitHashSuffix("something-abcdefg")
	if got != "abcdefg" {
		t.Errorf("keepShortCommitHashSuffix(\"something-abcdefg\") = %q, want \"abcdefg\"", got)
	}
}

func TestConstructLink(t *testing.T) {
	tests := []struct {
		repo   string
		expect string
	}{
		{"nginx", "https://hub.docker.com/_/nginx"},
		{"library/nginx", "https://hub.docker.com/_/nginx"},
		{"lscr.io/linuxserver/sonarr", "https://fleet.linuxserver.io/image?name=linuxserver/sonarr"},
		{"oci.trueforge.org/containerforge/myapp", "https://github.com/trueforge-org/containers/tree/main/apps/myapp"},
		{"mcr.microsoft.com/dotnet/sdk", "https://mcr.microsoft.com/en-us/product/dotnet/sdk"},
		{"public.ecr.aws/lambda/python", "https://gallery.ecr.aws/lambda/python"},
		{"ghcr.io/owner/image", "https://ghcr.io/owner/image"},
		{"quay.io/owner/image", "https://quay.io/owner/image"},
		{"gcr.io/project/image", "https://gcr.io/project/image"},
		{"docker.io/owner/image", "https://hub.docker.com/r/owner/image"},
		{"index.docker.io/owner/image", "https://hub.docker.com/r/owner/image"},
		{"registry-1.docker.io/owner/image", "https://hub.docker.com/r/owner/image"},
		{"registry.hub.docker.com/owner/image", "https://hub.docker.com/r/owner/image"},
		{"author.ocir.io/image", ""},
	}
	for _, tt := range tests {
		t.Run(tt.repo, func(t *testing.T) {
			got := constructLink(tt.repo)
			if got != tt.expect {
				t.Errorf("constructLink(%q) = %q, want %q", tt.repo, got, tt.expect)
			}
		})
	}
}

func TestConstructLinkAzureCR(t *testing.T) {
	got := constructLink("myregistry.azurecr.io/myimage")
	expected := "https://myregistry.azurecr.io/myimage"
	if got != expected {
		t.Errorf("constructLink for azurecr = %q, want %q", got, expected)
	}
}

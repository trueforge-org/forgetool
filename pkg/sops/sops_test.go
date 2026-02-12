package sops

import "testing"

func TestSmokeSops(t *testing.T) {}

func TestGetFormat(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		want     string
	}{
		{
			name:     "YAML file with .yaml extension",
			filePath: "config.yaml",
			want:     "yaml",
		},
		{
			name:     "YAML file with .yml extension",
			filePath: "settings.yml",
			want:     "yaml",
		},
		{
			name:     "JSON file",
			filePath: "data.json",
			want:     "json",
		},
		{
			name:     "Dotenv file",
			filePath: "secrets.env",
			want:     "dotenv",
		},
		{
			name:     "INI file",
			filePath: "config.ini",
			want:     "ini",
		},
		{
			name:     "Unknown extension",
			filePath: "data.txt",
			want:     "binary",
		},
		{
			name:     "No extension",
			filePath: "README",
			want:     "binary",
		},
		{
			name:     "Path with directory",
			filePath: "/path/to/config.yaml",
			want:     "yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetFormat(tt.filePath)
			if got != tt.want {
				t.Errorf("GetFormat(%q) = %q, want %q", tt.filePath, got, tt.want)
			}
		})
	}
}

func TestContainsSopsField(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{
			name: "Valid YAML with sops field",
			data: []byte(`
apiVersion: v1
kind: Secret
sops:
  kms: []
  gcp_kms: []
`),
			want: true,
		},
		{
			name: "Valid YAML without sops field",
			data: []byte(`
apiVersion: v1
kind: Secret
data:
  key: value
`),
			want: false,
		},
		{
			name: "Invalid YAML",
			data: []byte(`
invalid: [unclosed
`),
			want: false,
		},
		{
			name: "Empty data",
			data: []byte(""),
			want: false,
		},
		{
			name: "JSON with sops field",
			data: []byte(`{"sops": {"kms": []}}`),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsSopsField(tt.data)
			if got != tt.want {
				t.Errorf("containsSopsField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainsEncMarker(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{
			name: "Contains ENC[ marker",
			data: []byte("DATABASE_PASSWORD=ENC[AES256_GCM,data:xyz,iv:abc,tag:def,type:str]"),
			want: true,
		},
		{
			name: "Does not contain ENC[ marker",
			data: []byte("DATABASE_PASSWORD=plaintextpassword"),
			want: false,
		},
		{
			name: "Empty data",
			data: []byte(""),
			want: false,
		},
		{
			name: "Multiple ENC[ markers",
			data: []byte(`
KEY1=ENC[AES256_GCM,data:xyz,iv:abc]
KEY2=ENC[AES256_GCM,data:def,iv:ghi]
`),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsEncMarker(tt.data)
			if got != tt.want {
				t.Errorf("containsEncMarker() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsEncrypted(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		filePath string
		want     bool
	}{
		{
			name: "Encrypted YAML file",
			data: []byte(`
apiVersion: v1
kind: Secret
sops:
  kms: []
`),
			filePath: "secret.yaml",
			want:     true,
		},
		{
			name: "Unencrypted YAML file",
			data: []byte(`
apiVersion: v1
kind: Secret
data:
  key: value
`),
			filePath: "secret.yaml",
			want:     false,
		},
		{
			name:     "Encrypted ENV file",
			data:     []byte("PASSWORD=ENC[AES256_GCM,data:xyz]"),
			filePath: "secrets.env",
			want:     true,
		},
		{
			name:     "Unencrypted ENV file",
			data:     []byte("PASSWORD=plaintext"),
			filePath: "secrets.env",
			want:     false,
		},
		{
			name:     "Unknown file type",
			data:     []byte("some content"),
			filePath: "data.txt",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isEncrypted(tt.data, tt.filePath)
			if got != tt.want {
				t.Errorf("isEncrypted() = %v, want %v", got, tt.want)
			}
		})
	}
}


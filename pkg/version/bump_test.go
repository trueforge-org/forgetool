package version

import "testing"

func TestIncrementVersion_TableDriven(t *testing.T) {
	type args struct {
		version string
		kind    string
	}

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{name: "major", args: args{version: "1.2.3", kind: Major}, want: "2.0.0", wantErr: false},
		{name: "minor", args: args{version: "1.2.3", kind: Minor}, want: "1.3.0", wantErr: false},
		{name: "patch", args: args{version: "1.2.3", kind: Patch}, want: "1.2.4", wantErr: false},
		{name: "patch-alias-pindigest", args: args{version: "1.2.3", kind: "pinDigest"}, want: "1.2.4", wantErr: false},
		{name: "patch-alias-digest", args: args{version: "1.2.3", kind: "digest"}, want: "1.2.4", wantErr: false},
		{name: "patch-alias-pin", args: args{version: "1.2.3", kind: "pin"}, want: "1.2.4", wantErr: false},
		{name: "patch-alias-lockfile", args: args{version: "1.2.3", kind: "lockfile"}, want: "1.2.4", wantErr: false},
		{name: "patch-alias-uppercase", args: args{version: "1.2.3", kind: "PATCH"}, want: "1.2.4", wantErr: false},
		{name: "invalid-kind", args: args{version: "1.2.3", kind: "invalid"}, want: "", wantErr: true},
		{name: "invalid-format", args: args{version: "1.2", kind: Patch}, want: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IncrementVersion(tt.args.version, tt.args.kind)
			if (err != nil) != tt.wantErr {
				t.Fatalf("IncrementVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("IncrementVersion() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBump_ReturnsNilOnValid(t *testing.T) {
	if err := Bump("0.1.2", Patch); err != nil {
		t.Fatalf("Bump returned error for valid input: %v", err)
	}
}

func TestIncrementVersion_SemverParserError(t *testing.T) {
	_, err := IncrementVersion("9999999999999999999999999999.1.1", Patch)
	if err == nil {
		t.Fatalf("expected semver parser error for overflow value")
	}
}

func TestBump_ReturnsErrorOnInvalid(t *testing.T) {
	if err := Bump("not-semver", Patch); err == nil {
		t.Fatalf("expected Bump to return error on invalid semver")
	}
}

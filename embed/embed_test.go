package embed

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

func TestGetTalosExec_ReturnsPathForPlatform(t *testing.T) {
	got := GetTalosExec()
	if !strings.HasPrefix(got, helper.CacheDir) {
		t.Fatalf("expected path to start with cache dir %s, got %s", helper.CacheDir, got)
	}

	var expect string
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	switch goos {
	case "windows":
		if goarch == "amd64" {
			expect = "talosctl-windows-amd64.exe"
		} else {
			expect = "talosctl-windows-arm64.exe"
		}
	case "linux":
		if goarch == "amd64" {
			expect = "talosctl-linux-amd64"
		} else {
			expect = "talosctl-linux-arm64"
		}
	case "darwin":
		if goarch == "amd64" {
			expect = "talosctl-darwin-amd64"
		} else {
			expect = "talosctl-darwin-arm64"
		}
	case "freebsd":
		if goarch == "amd64" {
			expect = "talosctl-freebsd-amd64"
		} else {
			expect = "talosctl-freebsd-arm64"
		}
	default:
		t.Skipf("unsupported platform %s/%s for this test", goos, goarch)
	}

	if filepath.Base(got) != expect {
		t.Fatalf("expected exec basename %s, got %s", expect, filepath.Base(got))
	}
}

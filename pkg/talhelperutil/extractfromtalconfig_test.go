package talhelperutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

func TestExtractIPs(t *testing.T) {
	td := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	cfg := strings.Join([]string{
		"nodes:",
		"  - hostname: cp1",
		"    ipAddress: 10.0.0.10",
		"    controlPlane: true",
		"  - hostname: wk1",
		"    ipAddress: 10.0.0.11",
		"    controlPlane: false",
	}, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(td, "config.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatalf("write config.yaml failed: %v", err)
	}

	helper.AllIPs = nil
	helper.ControlPlaneIPs = nil
	helper.WorkerIPs = nil
	ExtractIPs()

	if len(helper.AllIPs) != 2 || len(helper.ControlPlaneIPs) != 1 || len(helper.WorkerIPs) != 1 {
		t.Fatalf("unexpected extracted IP sets: all=%v cp=%v worker=%v", helper.AllIPs, helper.ControlPlaneIPs, helper.WorkerIPs)
	}
}

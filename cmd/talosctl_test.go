package cmd

import (
	"testing"

	"github.com/siderolabs/talos/cmd/talosctl/cmd/mgmt"
	"github.com/siderolabs/talos/cmd/talosctl/cmd/mgmt/cluster"
	"github.com/siderolabs/talos/cmd/talosctl/cmd/talos"
)

func TestTalosctlCommandConfig(t *testing.T) {
	if talosctl.Use != "talosctl" {
		t.Fatalf("expected use %q, got %q", "talosctl", talosctl.Use)
	}
	if !talosctl.SilenceUsage {
		t.Fatalf("expected SilenceUsage to be true")
	}
	if !talosctl.SilenceErrors {
		t.Fatalf("expected SilenceErrors to be true")
	}
}

func TestTalosctlCommandHasExpectedGroups(t *testing.T) {
	const (
		talosGroup   = "talos"
		mgmtGroup    = "mgmt"
		clusterGroup = "cluster"
	)

	registered := make(map[string]bool)
	for _, group := range talosctl.Groups() {
		registered[group.ID] = true
	}

	for _, groupID := range []string{talosGroup, mgmtGroup, clusterGroup} {
		if !registered[groupID] {
			t.Fatalf("expected group %q to be registered", groupID)
		}
	}
}

func TestTalosctlCommandHasExpectedSubcommands(t *testing.T) {
	const (
		talosGroup   = "talos"
		mgmtGroup    = "mgmt"
		clusterGroup = "cluster"
	)

	registered := make(map[string]bool)
	for _, command := range talosctl.Commands() {
		registered[command.Name()] = true
	}

	for _, command := range mgmt.Commands {
		if !registered[command.Name()] {
			t.Fatalf("expected mgmt command %q to be registered", command.Name())
		}

		expectedGroupID := mgmtGroup
		if command == cluster.Cmd {
			expectedGroupID = clusterGroup
		}

		if command.GroupID != expectedGroupID {
			t.Fatalf("expected mgmt command %q to have group %q, got %q", command.Name(), expectedGroupID, command.GroupID)
		}
	}

	for _, command := range talos.Commands {
		if !registered[command.Name()] {
			t.Fatalf("expected talos command %q to be registered", command.Name())
		}
		if command.GroupID != talosGroup {
			t.Fatalf("expected talos command %q to have group %q, got %q", command.Name(), talosGroup, command.GroupID)
		}
	}
}

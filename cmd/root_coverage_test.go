package cmd

import "testing"

func TestClusterNameFromArgsCoverage(t *testing.T) {
	t.Run("uses fallback when args empty", func(t *testing.T) {
		got := clusterNameFromArgs([]string{}, "main")
		if got != "main" {
			t.Fatalf("expected fallback main, got %q", got)
		}
	})

	t.Run("parses equals form", func(t *testing.T) {
		got := clusterNameFromArgs([]string{"--cluster=lab"}, "main")
		if got != "lab" {
			t.Fatalf("expected lab, got %q", got)
		}
	})

	t.Run("parses spaced form", func(t *testing.T) {
		got := clusterNameFromArgs([]string{"--cluster", "edge"}, "main")
		if got != "edge" {
			t.Fatalf("expected edge, got %q", got)
		}
	})

	t.Run("ignores dangling cluster flag", func(t *testing.T) {
		got := clusterNameFromArgs([]string{"--cluster"}, "main")
		if got != "main" {
			t.Fatalf("expected fallback main, got %q", got)
		}
	})

	t.Run("last value wins", func(t *testing.T) {
		got := clusterNameFromArgs([]string{"--cluster=one", "--cluster", "two"}, "main")
		if got != "two" {
			t.Fatalf("expected two, got %q", got)
		}
	})
}

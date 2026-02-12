package helper

import (
	"testing"
	"time"
)

func TestIsTimeWithinThreshold_WithinThreshold(t *testing.T) {
	now := time.Now()
	ref := now.Add(-5 * time.Second)
	if !IsTimeWithinThreshold(now, ref, 10*time.Second) {
		t.Fatal("expected time within 5s to be within 10s threshold")
	}
}

func TestIsTimeWithinThreshold_ExactlyAtThreshold(t *testing.T) {
	now := time.Now()
	ref := now.Add(-10 * time.Second)
	// At exactly the threshold boundary the difference equals threshold, so >= is false
	if IsTimeWithinThreshold(now, ref, 10*time.Second) {
		t.Fatal("expected time exactly at threshold boundary to be out of range")
	}
}

func TestIsTimeWithinThreshold_BeyondThreshold(t *testing.T) {
	now := time.Now()
	ref := now.Add(-30 * time.Second)
	if IsTimeWithinThreshold(now, ref, 10*time.Second) {
		t.Fatal("expected time 30s apart to be outside 10s threshold")
	}
}

func TestIsTimeWithinThreshold_NegativeDifference(t *testing.T) {
	now := time.Now()
	ref := now.Add(5 * time.Second) // reference is in the future
	if !IsTimeWithinThreshold(now, ref, 10*time.Second) {
		t.Fatal("expected negative 5s difference to be within 10s threshold")
	}
}

func TestIsTimeWithinThreshold_NegativeBeyondThreshold(t *testing.T) {
	now := time.Now()
	ref := now.Add(30 * time.Second)
	if IsTimeWithinThreshold(now, ref, 10*time.Second) {
		t.Fatal("expected negative 30s difference to be outside 10s threshold")
	}
}

func TestIsTimeWithinThreshold_ZeroDifference(t *testing.T) {
	now := time.Now()
	if !IsTimeWithinThreshold(now, now, 10*time.Second) {
		t.Fatal("expected zero difference to be within threshold")
	}
}

func TestIsTimeWithinThreshold_ZeroThreshold(t *testing.T) {
	now := time.Now()
	if IsTimeWithinThreshold(now, now, 0) {
		t.Fatal("expected zero threshold to reject even zero difference (strict inequality)")
	}
}

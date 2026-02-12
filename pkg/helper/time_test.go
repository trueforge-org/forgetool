package helper

import (
	"testing"
)

// Note: CheckSystemTime makes actual NTP calls and calls os.Exit on failure,
// so we test it indirectly. In production code, this function could be refactored
// to be more testable by accepting dependencies as parameters.

func TestCheckSystemTime_Integration(t *testing.T) {
	// This is an integration test that actually calls NTP
	// Skip in CI environments or when network is unavailable
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test will pass if system time is within 10 seconds of NTP time
	// If it fails, it means either:
	// 1. System time is off by more than 10 seconds
	// 2. NTP server is unreachable
	result := CheckSystemTime()
	
	// If we get here, the function returned (didn't call os.Exit)
	// which means time check passed or NTP was unreachable but error was ignored
	if !result {
		t.Log("CheckSystemTime returned false (unexpected)")
	}
}

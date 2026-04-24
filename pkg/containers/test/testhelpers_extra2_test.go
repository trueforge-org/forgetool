package containertest

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestVerboseHTTPWaitStrategy_NotObservedHTTP(t *testing.T) {
	v := &verboseHTTPWaitStrategy{
		base:           fakeWaitStrategyFailure{err: errors.New("timeout")},
		requestPath:    "/r",
		port:           "80",
		expectedStatus: 200,
	}
	err := v.WaitUntilReady(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "no HTTP response was observed") {
		t.Fatalf("expected no-HTTP-observed error, got %v", err)
	}
}

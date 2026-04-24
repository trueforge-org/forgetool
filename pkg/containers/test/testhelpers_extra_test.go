package containertest

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/testcontainers/testcontainers-go/wait"
)

func TestLabeledWaitStrategy_String_NilReceiver(t *testing.T) {
	var l *labeledWaitStrategy
	if got := l.String(); got != "<nil>" {
		t.Fatalf("expected <nil>, got %q", got)
	}
}

func TestLabeledWaitStrategy_String_EmptyLabelNilStrategy(t *testing.T) {
	l := &labeledWaitStrategy{}
	if got := l.String(); got != "<nil>" {
		t.Fatalf("expected <nil>, got %q", got)
	}
}

func TestLabeledWaitStrategy_String_StringerStrategy(t *testing.T) {
	l := &labeledWaitStrategy{strategy: wait.ForHealthCheck()}
	got := l.String()
	if got == "" || got == "<nil>" {
		t.Fatalf("expected stringer output, got %q", got)
	}
}

func TestLabeledWaitStrategy_String_DefaultFmt(t *testing.T) {
	l := &labeledWaitStrategy{strategy: fakeWaitStrategyNoStringer{}}
	got := l.String()
	if !strings.Contains(got, "fakeWaitStrategyNoStringer") {
		t.Fatalf("expected type fallback, got %q", got)
	}
}

func TestLabeledWaitStrategy_WaitUntilReady_NilReceiver(t *testing.T) {
	var l *labeledWaitStrategy
	if err := l.WaitUntilReady(context.Background(), nil); err == nil {
		t.Fatalf("expected error")
	}
}

func TestLabeledWaitStrategy_WaitUntilReady_NilStrategy(t *testing.T) {
	l := &labeledWaitStrategy{label: "x"}
	if err := l.WaitUntilReady(context.Background(), nil); err == nil {
		t.Fatalf("expected error")
	}
}

func TestWithWaitOrderLabels_EmptyInput(t *testing.T) {
	if got := withWaitOrderLabels(nil); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestWithWaitOrderLabels_NilLabeledEntry(t *testing.T) {
	var nilLabeled *labeledWaitStrategy
	out := withWaitOrderLabels([]wait.Strategy{nilLabeled})
	if len(out) != 1 {
		t.Fatalf("expected 1 entry")
	}
	if err := out[0].WaitUntilReady(context.Background(), nil); err == nil {
		t.Fatalf("expected nil-strategy error")
	}
}

func TestVerboseHTTPWaitStrategy_NilBase(t *testing.T) {
	v := &verboseHTTPWaitStrategy{}
	if err := v.WaitUntilReady(context.Background(), nil); err == nil {
		t.Fatalf("expected nil base error")
	}
}

func TestVerboseHTTPWaitStrategy_HasObservedHTTP_KnownStatus(t *testing.T) {
	v := &verboseHTTPWaitStrategy{
		base:            fakeWaitStrategyFailure{err: errors.New("timeout")},
		requestPath:     "/r",
		port:            "80",
		expectedStatus:  200,
		lastStatus:      503,
		hasObservedHTTP: true,
	}
	err := v.WaitUntilReady(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("expected 503 in error, got %v", err)
	}
}

func TestVerboseHTTPWaitStrategy_HasObservedHTTP_UnknownStatus(t *testing.T) {
	v := &verboseHTTPWaitStrategy{
		base:            fakeWaitStrategyFailure{err: errors.New("timeout")},
		requestPath:     "/r",
		port:            "80",
		expectedStatus:  200,
		lastStatus:      999,
		hasObservedHTTP: true,
	}
	err := v.WaitUntilReady(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "unknown status") {
		t.Fatalf("expected unknown status in error, got %v", err)
	}
}

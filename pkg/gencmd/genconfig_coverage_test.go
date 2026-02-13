package gencmd

import (
	"errors"
	"path"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

type badYAML struct{}

func (badYAML) MarshalYAML() (any, error) {
	return nil, errors.New("marshal fail")
}

func TestGenConfigCoverage(t *testing.T) {
	resetGencmdHooks(t)
	checkRunAgainFileExistsFn = func() bool { return true }
	osExitFn = func(int) { panic(exitPanic{}) }
	expectExitPanic(t, func() {
		_ = GenConfig(nil)
	})

	resetGencmdHooks(t)
	helper.ClusterPath = "/tmp/cluster"
	calls := 0
	sopsDecryptFilesFn = func() error { return errors.New("decrypt") }
	processDirectoryFn = func(p string) error {
		if p != path.Join(helper.ClusterPath, "kubernetes") {
			t.Fatalf("unexpected process path: %s", p)
		}
		calls++
		if calls == 1 {
			return errors.New("first")
		}
		return nil
	}
	if err := GenConfig(nil); err != nil {
		t.Fatalf("unexpected GenConfig error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected processDirectory called twice, got %d", calls)
	}
}

func TestDefaultEncodeSecretBundleError(t *testing.T) {
	if _, err := defaultEncodeSecretBundle(map[string]any{"bad": make(chan int)}); err == nil {
		t.Fatal("expected encode error for unsupported value type")
	}
	if _, err := defaultEncodeSecretBundle(badYAML{}); err == nil {
		t.Fatal("expected encode error from MarshalYAML")
	}
}

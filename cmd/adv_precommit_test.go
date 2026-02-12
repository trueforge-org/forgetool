package cmd

import "testing"

func TestRunPrecommitPassesExpectedFlags(t *testing.T) {
	oldFn := precommitCheckFilesAndReportEncryption
	t.Cleanup(func() { precommitCheckFilesAndReportEncryption = oldFn })

	called := false
	precommitCheckFilesAndReportEncryption = func(precommit bool, decrypt bool) error {
		called = true
		if !precommit {
			t.Fatalf("expected precommit=true")
		}
		if !decrypt {
			t.Fatalf("expected decrypt=true")
		}
		return nil
	}

	if err := runPrecommit(); err != nil {
		t.Fatalf("runPrecommit returned unexpected error: %v", err)
	}
	if !called {
		t.Fatalf("expected checker function to be called")
	}
}

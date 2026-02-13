package cmd

import "testing"

func TestInitFilesCommandConfig(t *testing.T) {
	if initFiles.Use != "init" {
		t.Fatalf("expected use %q, got %q", "init", initFiles.Use)
	}
	if initFiles.Run == nil {
		t.Fatalf("expected Run handler to be set")
	}
}

func TestInitFilesCommandRegisteredOnCluster(t *testing.T) {
	found := false
	for _, command := range clusterCmd.Commands() {
		if command.Name() == "init" {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected init command to be registered on cluster command")
	}
}

func TestRunClusterInitCallsDependencies(t *testing.T) {
	oldDecrypt := clusterInitDecryptFiles
	oldInit := clusterInitInitFiles
	t.Cleanup(func() {
		clusterInitDecryptFiles = oldDecrypt
		clusterInitInitFiles = oldInit
	})

	decryptCalled := false
	initCalled := false
	clusterInitDecryptFiles = func() error { decryptCalled = true; return nil }
	clusterInitInitFiles = func() error { initCalled = true; return nil }

	runClusterInit()

	if !decryptCalled || !initCalled {
		t.Fatalf("expected both decrypt and init calls")
	}
}

func TestInitFilesCommandRunUsesHelper(t *testing.T) {
	oldDecrypt := clusterInitDecryptFiles
	oldInit := clusterInitInitFiles
	t.Cleanup(func() {
		clusterInitDecryptFiles = oldDecrypt
		clusterInitInitFiles = oldInit
	})

	decryptCalled := false
	initCalled := false
	clusterInitDecryptFiles = func() error { decryptCalled = true; return nil }
	clusterInitInitFiles = func() error { initCalled = true; return nil }

	initFiles.Run(initFiles, nil)

	if !decryptCalled || !initCalled {
		t.Fatalf("expected both decrypt and init calls from command run")
	}
}

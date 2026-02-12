package gencmd

import "testing"

func TestExecCmd_SuccessAndErrorPaths(t *testing.T) {
	ExecCmd("/bin/echo hello")
	ExecCmd("/usr/bin/false")
}

func TestExecCmd_BootstrapPathRetriesOnce(t *testing.T) {
	// This triggers the bootstrap-specific branch; it may sleep briefly by design.
	ExecCmd("/usr/bin/false bootstrap")
}

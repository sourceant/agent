//go:build !unix

package supervise

import "os/exec"

// Process groups are a Unix idea. Elsewhere the supervisor reaches the process
// it started and no further, which is what the standard library offers.
func isolate(*exec.Cmd) {}

func askToStop(command *exec.Cmd) error { return command.Process.Kill() }

func forceStop(command *exec.Cmd) error { return command.Process.Kill() }

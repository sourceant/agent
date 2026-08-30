//go:build unix

package supervise

import (
	"os/exec"
	"syscall"
)

// isolate puts the process in a group of its own.
//
// Without it, stopping the core reaches only the process we started. Anything
// it started stays running, holding the port the next start wants.
func isolate(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// signal sends sig to the whole group, falling back to the process alone when
// there is no group to address.
func signal(command *exec.Cmd, sig syscall.Signal) error {
	pid := command.Process.Pid
	if group, err := syscall.Getpgid(pid); err == nil {
		return syscall.Kill(-group, sig)
	}
	return syscall.Kill(pid, sig)
}

func askToStop(command *exec.Cmd) error { return signal(command, syscall.SIGTERM) }

func forceStop(command *exec.Cmd) error { return signal(command, syscall.SIGKILL) }

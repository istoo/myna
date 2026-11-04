//go:build !windows

package codexrelay

import (
	"os/exec"
	"syscall"
)

// alive reports whether pid is still running.
func alive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}

// detach keeps the relay out of myna's process group, so it outlives
// myna.
func detach(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setsid = true
}

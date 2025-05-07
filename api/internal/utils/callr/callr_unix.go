//go:build !windows

package callr

import (
	"os/exec"
	"syscall"
)

type unixPlat struct{}

var plat platform = unixPlat{}

func (unixPlat) Setup(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return nil
}

func (unixPlat) Assign(_ *exec.Cmd) error { return nil }

func (unixPlat) Teardown(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		return cmd.Process.Kill()
	}
	return syscall.Kill(-pgid, syscall.SIGKILL)
}

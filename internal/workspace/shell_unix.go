//go:build !windows

package workspace

import (
	"context"
	"os/exec"
	"syscall"
)

func shellCommand(ctx context.Context, s string) *exec.Cmd {
	c := exec.CommandContext(ctx, "/bin/sh", "-c", s)
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	c.Cancel = func() error {
		if c.Process == nil {
			return nil
		}
		return syscall.Kill(-c.Process.Pid, syscall.SIGKILL)
	}
	return c
}

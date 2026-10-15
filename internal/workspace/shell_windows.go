package workspace

import (
	"context"
	"os/exec"
	"strconv"
)

func shellCommand(ctx context.Context, s string) *exec.Cmd {
	c := exec.CommandContext(ctx, "cmd.exe", "/d", "/s", "/c", s)
	c.Cancel = func() error {
		if c.Process == nil {
			return nil
		}
		return exec.Command("taskkill.exe", "/PID", strconv.Itoa(c.Process.Pid), "/T", "/F").Run()
	}
	return c
}

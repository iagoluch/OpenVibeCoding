package automation

import (
	"os/exec"
	"strconv"
	"syscall"
)

func contain(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000200}
	c.Cancel = func() error {
		if c.Process == nil {
			return nil
		}
		return exec.Command("taskkill.exe", "/PID", strconv.Itoa(c.Process.Pid), "/T", "/F").Run()
	}
}

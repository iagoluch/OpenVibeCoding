package automation

import (
	"context"
	"os"
	"os/exec"
	"time"
)

func command(ctx context.Context, name string, args ...string) *exec.Cmd {
	c := exec.CommandContext(ctx, name, args...)
	for _, key := range []string{"PATH", "HOME", "USERPROFILE", "LOCALAPPDATA", "APPDATA", "SystemRoot", "COMSPEC", "TEMP", "TMP", "TMPDIR", "DISPLAY", "XAUTHORITY", "DBUS_SESSION_BUS_ADDRESS", "XDG_RUNTIME_DIR", "LANG", "PLAYWRIGHT_BROWSERS_PATH"} {
		if v := os.Getenv(key); v != "" {
			c.Env = append(c.Env, key+"="+v)
		}
	}
	contain(c)
	c.WaitDelay = time.Second
	return c
}

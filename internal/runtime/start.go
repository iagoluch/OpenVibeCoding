package runtime

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"time"
)

// Ensure reuses an authenticated compatible service, spawning only when needed.
func Ensure(ctx context.Context, dir string) (*Client, error) {
	if c, e := Connect(dir); e == nil {
		return c, nil
	}
	exe, e := os.Executable()
	if e != nil {
		return nil, e
	}
	cmd := exec.Command(exe, "--runtime", "serve")
	cmd.Env = append(os.Environ(), "INTERPRETER_HOME="+dir)
	detach(cmd)
	null, e := os.OpenFile(os.DevNull, os.O_RDWR, 0600)
	if e != nil {
		return nil, e
	}
	defer null.Close()
	cmd.Stdin = null
	cmd.Stdout = null
	cmd.Stderr = null
	if e = cmd.Start(); e != nil {
		return nil, e
	}
	go cmd.Wait()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(10 * time.Second)
	defer timeout.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout.C:
			return nil, errors.New("runtime did not start; run interpreter --runtime serve for diagnostics")
		case <-ticker.C:
			if c, e := Connect(dir); e == nil {
				return c, nil
			}
		}
	}
}

package automation

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func SetupBrowser(ctx context.Context, dir string) error {
	prefix := filepath.Join(dir, "qa")
	if e := os.MkdirAll(prefix, 0700); e != nil {
		return e
	}
	for _, args := range [][]string{{"npm", "install", "--ignore-scripts", "--prefix", prefix, "playwright@1.58.2"}, {"node", filepath.Join(prefix, "node_modules", "playwright", "cli.js"), "install", "chromium"}} {
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if e := cmd.Run(); e != nil {
			return fmt.Errorf("browser setup failed; verify Node/npm and browser system libraries: %w", e)
		}
	}
	return nil
}

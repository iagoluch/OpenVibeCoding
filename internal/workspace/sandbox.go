package workspace

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// sandboxCommand exposes only OS runtime files and the selected project. It
// denies network, host home and /proc, and dies with its parent. No fallback.
func (w *Workspace) sandboxCommand(ctx context.Context, command string) (*exec.Cmd, error) {
	if runtime.GOOS != "linux" {
		return nil, errors.New("OS sandbox backend unavailable on this platform; host execution requires explicit opt-in")
	}
	path, e := exec.LookPath("bwrap")
	if e != nil {
		return nil, errors.New("install bubblewrap to enable confined commands")
	}
	args := []string{"--die-with-parent", "--new-session", "--unshare-all", "--ro-bind", "/usr", "/usr", "--tmpfs", "/tmp", "--dir", "/dev", "--ro-bind", "/dev/null", "/dev/null", "--ro-bind", "/dev/urandom", "/dev/urandom"}
	for _, name := range []string{"bin", "sbin", "lib", "lib64"} {
		p := "/" + name
		if info, e := os.Lstat(p); e == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				target, e := os.Readlink(p)
				if e != nil {
					return nil, e
				}
				args = append(args, "--symlink", target, p)
			} else {
				args = append(args, "--ro-bind", p, p)
			}
		}
	}
	if w.Network {
		args = append(args, "--share-net")
		for _, p := range []string{"/etc/resolv.conf", "/etc/hosts", "/etc/ssl/certs", "/etc/pki"} {
			if _, err := os.Stat(p); err == nil {
				args = append(args, "--ro-bind", p, p)
			}
		}
	}
	args = append(args, "--bind", w.Path, "/workspace")
	// Mask protected project entries, including secrets in nested directories.
	err := filepath.WalkDir(w.Path, func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if p == w.Path {
			return nil
		}
		if d.IsDir() && (d.Name() == "node_modules" || d.Name() == "vendor") {
			return filepath.SkipDir
		}
		if denied(d.Name()) {
			rel, e := filepath.Rel(w.Path, p)
			if e != nil {
				return e
			}
			target := "/workspace/" + filepath.ToSlash(rel)
			if d.IsDir() {
				args = append(args, "--tmpfs", target)
				return filepath.SkipDir
			}
			args = append(args, "--ro-bind", "/dev/null", target)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	args = append(args, "--chdir", "/workspace", "--setenv", "HOME", "/workspace", "/bin/sh", "-c", command)
	return exec.CommandContext(ctx, path, args...), nil
}
func (w *Workspace) SandboxStatus() string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd, e := w.sandboxCommand(ctx, "true")
	if e != nil {
		return e.Error()
	}
	cmd.Env = []string{"PATH=/usr/bin:/bin"}
	if e = cmd.Run(); e != nil {
		return "bubblewrap isolation unavailable (kernel or host policy)"
	}
	return "bubblewrap: workspace write, runtime read-only, network and host home denied"
}

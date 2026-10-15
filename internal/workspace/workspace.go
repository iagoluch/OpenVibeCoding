// Package workspace confines file operations with os.Root. Shell execution is
// explicitly outside this boundary and disabled unless opted in at launch.
package workspace

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/pmezard/go-difflib/difflib"
	ignore "github.com/sabhiram/go-gitignore"
)

const MaxFile = 128 << 10

type Workspace struct {
	Root       *os.Root
	Path       string
	AllowShell bool
	Network    bool
	mu         sync.Mutex
}
type Change struct {
	Path, Before, After, Diff string
	Exists                    bool
	Hash                      [32]byte
}

func Open(path string, allowShell bool) (*Workspace, error) {
	p, e := filepath.Abs(path)
	if e != nil {
		return nil, e
	}
	p, e = filepath.EvalSymlinks(p)
	if e != nil {
		return nil, e
	}
	r, e := os.OpenRoot(p)
	if e != nil {
		return nil, e
	}
	return &Workspace{Root: r, Path: p, AllowShell: allowShell}, nil
}
func (w *Workspace) Close() error { return w.Root.Close() }
func denied(name string) bool {
	n := strings.ToLower(name)
	return n == ".git" || n == ".ssh" || n == ".aws" || n == ".gnupg" || n == "node_modules" || n == "vendor" || n == ".env" || strings.HasPrefix(n, ".env.") || strings.HasSuffix(n, ".pem") || strings.HasSuffix(n, ".key") || strings.Contains(n, "credentials") || n == "id_rsa" || n == "id_ed25519"
}
func (w *Workspace) check(path string) error {
	if path == "" || filepath.IsAbs(path) || !filepath.IsLocal(path) {
		return errors.New("path must be relative and inside workspace")
	}
	for _, p := range strings.FieldsFunc(path, func(r rune) bool { return r == '/' || r == '\\' }) {
		if denied(p) {
			return errors.New("protected or generated path is excluded")
		}
	}
	// Apply root and nested gitignore patterns without invoking project programs.
	clean := filepath.Clean(path)
	dir := "."
	for {
		f, err := w.Root.Open(filepath.Join(dir, ".gitignore"))
		if err == nil {
			b, readErr := io.ReadAll(io.LimitReader(f, 65536))
			f.Close()
			if readErr == nil {
				rel, err := filepath.Rel(dir, clean)
				if err == nil && ignore.CompileIgnoreLines(strings.Split(string(b), "\n")...).MatchesPath(filepath.ToSlash(rel)) {
					return errors.New("path excluded by .gitignore")
				}
			}
		}
		rel, err := filepath.Rel(dir, clean)
		if err != nil {
			break
		}
		parts := strings.Split(rel, string(filepath.Separator))
		if len(parts) < 2 {
			break
		}
		dir = filepath.Join(dir, parts[0])
	}
	// Reject symlinks even within the root, so aliases cannot bypass protected names.
	parts := strings.Split(filepath.Clean(path), string(filepath.Separator))
	prefix := ""
	for _, p := range parts {
		prefix = filepath.Join(prefix, p)
		info, e := w.Root.Lstat(prefix)
		if e == nil && info.Mode()&os.ModeSymlink != 0 {
			return errors.New("symlink paths are excluded")
		}
		if e != nil && !errors.Is(e, os.ErrNotExist) {
			return e
		}
	}
	return nil
}
func (w *Workspace) Read(path string) (string, error) {
	if e := w.check(path); e != nil {
		return "", e
	}
	f, e := w.Root.Open(path)
	if e != nil {
		return "", e
	}
	defer f.Close()
	info, e := f.Stat()
	if e != nil {
		return "", e
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("only regular files can be read")
	}
	b, e := io.ReadAll(io.LimitReader(f, MaxFile+1))
	if e != nil {
		return "", e
	}
	if len(b) > MaxFile {
		return "", errors.New("file exceeds 128 KiB read limit")
	}
	if !utf8.Valid(b) || strings.ContainsRune(string(b), 0) {
		return "", errors.New("binary file excluded")
	}
	return string(b), nil
}
func (w *Workspace) List() (string, error) {
	var paths []string
	count := 0
	err := fs.WalkDir(w.Root.FS(), ".", func(path string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		count++
		if count > 10000 {
			return fs.SkipAll
		}
		if path == "." {
			return nil
		}
		if denied(d.Name()) || w.check(path) != nil {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if !d.IsDir() {
			paths = append(paths, path)
		}
		if len(paths) >= 1000 {
			return fs.SkipAll
		}
		return nil
	})
	return strings.Join(paths, "\n"), err
}
func (w *Workspace) Propose(path, after string) (Change, error) {
	c := Change{Path: path, After: after}
	if len(after) > MaxFile || !utf8.ValidString(after) || strings.ContainsRune(after, 0) {
		return c, errors.New("replacement must be UTF-8 text under 128 KiB")
	}
	if e := w.check(path); e != nil {
		return c, e
	}
	before, e := w.Read(path)
	if e != nil && !errors.Is(e, os.ErrNotExist) {
		return c, e
	}
	c.Exists = e == nil
	c.Before = before
	c.Hash = sha256.Sum256([]byte(before))
	c.Diff, e = difflib.GetUnifiedDiffString(difflib.UnifiedDiff{A: difflib.SplitLines(before), B: difflib.SplitLines(after), FromFile: "a/" + path, ToFile: "b/" + path, Context: 3})
	return c, e
}
func (w *Workspace) Apply(c Change) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	now, e := w.Read(c.Path)
	if e != nil && !errors.Is(e, os.ErrNotExist) {
		return e
	}
	if (e == nil) != c.Exists || sha256.Sum256([]byte(now)) != c.Hash {
		return errors.New("file changed since review; request a new diff")
	}
	if e = w.check(c.Path); e != nil {
		return e
	}
	// Exclusive temporary file plus rename prevents partial files on interruption.
	if e = w.Root.MkdirAll(filepath.Dir(c.Path), 0755); e != nil {
		return e
	}
	tmp := c.Path + fmt.Sprintf(".interpreter-%d", time.Now().UnixNano())
	f, e := w.Root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return e
	}
	defer w.Root.Remove(tmp)
	if c.Exists {
		info, e := w.Root.Stat(c.Path)
		if e != nil {
			f.Close()
			return e
		}
		if e = f.Chmod(info.Mode().Perm()); e != nil {
			f.Close()
			return e
		}
	}
	if _, e = f.WriteString(c.After); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return w.Root.Rename(tmp, c.Path)
}

type capped struct {
	mu sync.Mutex
	b  strings.Builder
}

func (c *capped) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := len(p)
	if left := 32768 - c.b.Len(); left > 0 {
		if len(p) > left {
			p = p[:left]
		}
		c.b.Write(p)
	}
	return n, nil
}
func (w *Workspace) Shell(ctx context.Context, command string) (string, error) {

	if len(command) > 8192 {
		return "", errors.New("command too long")
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	var cmd *exec.Cmd
	if w.AllowShell {
		cmd = shellCommand(ctx, command)
	} else {
		var err error
		cmd, err = w.sandboxCommand(ctx, command)
		if err != nil {
			return "", err
		}
	}
	cmd.Dir = w.Path
	// Do not pass API keys or arbitrary inherited secrets to subprocesses.
	for _, k := range []string{"PATH", "SystemRoot", "COMSPEC", "TEMP", "TMP", "LANG"} {
		if v := os.Getenv(k); v != "" {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}
	cmd.Env = append(cmd.Env, "HOME="+w.Path, "USERPROFILE="+w.Path)
	var stdout, stderr capped
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.WaitDelay = time.Second
	e := cmd.Run()
	code := 0
	if e != nil {
		code = -1
		var exit *exec.ExitError
		if errors.As(e, &exit) {
			code = exit.ExitCode()
		}
	}
	return fmt.Sprintf("exit_code=%d\nstdout (max 32 KiB):\n%s\nstderr (max 32 KiB):\n%s", code, stdout.b.String(), stderr.b.String()), e
}

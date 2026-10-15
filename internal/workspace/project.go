package workspace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func (w *Workspace) Search(query string) (string, error) {
	if query == "" {
		return "", errors.New("nonempty search query required")
	}
	list, e := w.List()
	if e != nil {
		return "", e
	}
	var out strings.Builder
	for _, p := range strings.Split(list, "\n") {
		if strings.Contains(p, query) {
			fmt.Fprintln(&out, p)
		}
		text, e := w.Read(p)
		if e != nil {
			continue
		}
		for i, line := range strings.Split(text, "\n") {
			if strings.Contains(line, query) {
				fmt.Fprintf(&out, "%s:%d:%s\n", p, i+1, line)
			}
		}
		if out.Len() > 24000 {
			return out.String()[:24000] + "\n[search truncated]", nil
		}
	}
	return out.String(), nil
}
func (w *Workspace) Git(ctx context.Context, operation string) (string, error) {
	if operation != "status" && operation != "diff" {
		return "", errors.New("git operation must be status or diff")
	}
	args := []string{"--no-pager", "-c", "core.fsmonitor=false", "-c", "core.hooksPath=/dev/null", operation}
	if operation == "diff" {
		args = append(args, "--no-ext-diff", "--no-textconv")
	} else {
		args = append(args, "--short")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = w.Path
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=" + os.DevNull, "GIT_TERMINAL_PROMPT=0"}
	var out capped
	cmd.Stdout = &out
	cmd.Stderr = &out
	e := cmd.Run()
	return out.b.String(), e
}
func (w *Workspace) unchanged(c Change) error {
	now, e := w.Read(c.Path)
	if e != nil {
		return e
	}
	if !c.Exists || sha256.Sum256([]byte(now)) != c.Hash {
		return errors.New("file changed since review")
	}
	return nil
}
func (w *Workspace) Delete(c Change) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if e := w.unchanged(c); e != nil {
		return e
	}
	return w.Root.Remove(c.Path)
}
func (w *Workspace) Move(c Change, dest string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if e := w.unchanged(c); e != nil {
		return e
	}
	if e := w.check(dest); e != nil {
		return e
	}
	if _, e := w.Root.Lstat(dest); !errors.Is(e, os.ErrNotExist) {
		return errors.New("destination exists or is inaccessible")
	}
	if e := w.Root.MkdirAll(filepath.Dir(dest), 0755); e != nil {
		return e
	}
	f, e := w.Root.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return e
	}
	info, e := w.Root.Stat(c.Path)
	if e != nil {
		f.Close()
		w.Root.Remove(dest)
		return e
	}
	if e = f.Chmod(info.Mode().Perm()); e != nil {
		f.Close()
		w.Root.Remove(dest)
		return e
	}
	_, e = f.WriteString(c.Before)
	if e == nil {
		e = f.Sync()
	}
	f.Close()
	if e != nil {
		w.Root.Remove(dest)
		return e
	}
	if e = w.Root.Remove(c.Path); e != nil {
		return errors.New("destination created but original could not be removed; inspect both paths")
	}
	return nil
}

// Inverse describes how to reverse one applied change. The agent builds it
// when a change is applied and /undo replays it later.
type Inverse struct {
	Action  string `json:"action"` // write | delete | move
	Path    string `json:"path"`
	From    string `json:"from,omitempty"`
	Content string `json:"content,omitempty"`
}

// Digest returns the sha256 hex digest used to detect edits made after a
// change was applied or reviewed.
func Digest(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// verifyState checks the workspace still matches the state recorded right
// after a change: expect is the digest of that content, or empty when the
// file was absent. Undo must never overwrite newer edits silently.
func (w *Workspace) verifyState(path, expect string) error {
	now, e := w.Read(path)
	if e != nil && !errors.Is(e, os.ErrNotExist) {
		return e
	}
	exists := e == nil
	if expect == "" {
		if exists {
			return errors.New("a file exists at that path now; refusing to overwrite newer work")
		}
		return nil
	}
	if !exists {
		return errors.New("the file is gone; nothing to undo")
	}
	if Digest(now) != expect {
		return errors.New("the file changed after that operation; refusing to overwrite newer edits")
	}
	return nil
}

// Undo reverses one applied change. expect is the post-change state digest
// recorded with it (empty when the file was absent afterwards). Confinement,
// size limits and freshness checks apply exactly as for reviewed writes.
func (w *Workspace) Undo(inv Inverse, expect string) error {
	check := inv.Path
	if inv.Action == "move" {
		check = inv.From
	}
	switch inv.Action {
	case "write", "delete", "move":
	default:
		return errors.New("unknown undo action")
	}
	if e := w.verifyState(check, expect); e != nil {
		return e
	}
	switch inv.Action {
	case "write":
		c, e := w.Propose(inv.Path, inv.Content)
		if e != nil {
			return e
		}
		return w.Apply(c)
	case "delete":
		c, e := w.Propose(inv.Path, "")
		if e != nil {
			return e
		}
		if !c.Exists {
			return errors.New("the file is already gone; nothing to undo")
		}
		return w.Delete(c)
	default: // move
		c, e := w.Propose(inv.From, "")
		if e != nil {
			return e
		}
		if !c.Exists {
			return errors.New("the moved file is gone; nothing to undo")
		}
		return w.Move(c, inv.Path)
	}
}

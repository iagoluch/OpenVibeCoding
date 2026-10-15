package workspace

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func setup(t *testing.T) *Workspace {
	t.Helper()
	w, e := Open(t.TempDir(), false)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { w.Close() })
	return w
}
func TestConfinedReads(t *testing.T) {
	w := setup(t)
	outside := t.TempDir()
	os.WriteFile(filepath.Join(outside, "secret"), []byte("secret"), 0600)
	os.WriteFile(filepath.Join(w.Path, ".env"), []byte("secret"), 0600)
	os.Symlink(outside, filepath.Join(w.Path, "escape"))
	os.Symlink(".env", filepath.Join(w.Path, "alias"))
	for _, p := range []string{"../secret", filepath.Join(outside, "secret"), ".env", "escape/secret", "alias", ".git/config"} {
		if _, e := w.Read(p); e == nil {
			t.Errorf("allowed %s", p)
		}
	}
}
func TestReviewAndConflict(t *testing.T) {
	w := setup(t)
	os.WriteFile(filepath.Join(w.Path, "a.txt"), []byte("before\n"), 0644)
	c, e := w.Propose("a.txt", "after\n")
	if e != nil || !strings.Contains(c.Diff, "-before") || !strings.Contains(c.Diff, "+after") {
		t.Fatalf("%+v %v", c, e)
	}
	b, _ := w.Read("a.txt")
	if b != "before\n" {
		t.Fatal("proposal modified file")
	}
	if e = w.Apply(c); e != nil {
		t.Fatal(e)
	}
	b, _ = w.Read("a.txt")
	if b != "after\n" {
		t.Fatal(b)
	}
	if e = w.Apply(c); e == nil {
		t.Fatal("stale diff applied")
	}
	c, _ = w.Propose("new.txt", "new")
	os.WriteFile(filepath.Join(w.Path, "new.txt"), []byte("someone else's work"), 0600)
	if e = w.Apply(c); e == nil {
		t.Fatal("overwrote concurrent creation")
	}
}
func TestLimitsAndShell(t *testing.T) {
	w := setup(t)
	os.WriteFile(filepath.Join(w.Path, "binary"), []byte{0, 1}, 0600)
	if _, e := w.Read("binary"); e == nil {
		t.Fatal("binary accepted")
	}

	if runtime.GOOS == "windows" {
		t.Skip("Unix shell fixture")
	}
	w.AllowShell = true
	s, e := w.Shell(context.Background(), "printf hello; printf error >&2; exit 7")
	if e == nil || !strings.Contains(s, "exit_code=7") || !strings.Contains(s, "hello") || !strings.Contains(s, "error") {
		t.Fatalf("%q %v", s, e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, e = w.Shell(ctx, "sleep 30")
	if e == nil || time.Since(start) > 3*time.Second {
		t.Fatal("cancellation failed", e)
	}
}


func TestUndoReversesAppliedChanges(t *testing.T) {
	w := setup(t)
	write := func(path, content string) {
		t.Helper()
		c, e := w.Propose(path, content)
		if e != nil {
			t.Fatal(e)
		}
		if e = w.Apply(c); e != nil {
			t.Fatal(e)
		}
	}
	write("doc.txt", "v1\n")
	write("doc.txt", "v2\n")
	if e := w.Undo(Inverse{Action: "write", Path: "doc.txt", Content: "v1\n"}, Digest("v2\n")); e != nil {
		t.Fatal(e)
	}
	if text, _ := w.Read("doc.txt"); text != "v1\n" {
		t.Fatalf("write undo left %q", text)
	}
	// Newer edits are never overwritten silently.
	write("doc.txt", "v2\n")
	write("doc.txt", "v3\n")
	if e := w.Undo(Inverse{Action: "write", Path: "doc.txt", Content: "v1\n"}, Digest("v2\n")); e == nil {
		t.Fatal("stale undo accepted")
	}
	// Undo of a creation removes the file.
	write("new.txt", "fresh\n")
	if e := w.Undo(Inverse{Action: "delete", Path: "new.txt"}, Digest("fresh\n")); e != nil {
		t.Fatal(e)
	}
	if _, e := w.Read("new.txt"); e == nil {
		t.Fatal("created file survived undo")
	}
	// Undo of a delete restores the content only while the path stays free.
	write("gone.txt", "keep\n")
	c, e := w.Propose("gone.txt", "")
	if e != nil {
		t.Fatal(e)
	}
	if e = w.Delete(c); e != nil {
		t.Fatal(e)
	}
	if e = w.Undo(Inverse{Action: "write", Path: "gone.txt", Content: "keep\n"}, ""); e != nil {
		t.Fatal(e)
	}
	if text, _ := w.Read("gone.txt"); text != "keep\n" {
		t.Fatalf("delete undo left %q", text)
	}
	if e = w.Undo(Inverse{Action: "write", Path: "gone.txt", Content: "keep\n"}, ""); e == nil {
		t.Fatal("recreated path overwritten")
	}
	// Undo of a move brings the file back to its source.
	write("a.txt", "moving\n")
	if c, e = w.Propose("a.txt", ""); e != nil {
		t.Fatal(e)
	}
	if e = w.Move(c, "sub/b.txt"); e != nil {
		t.Fatal(e)
	}
	if e = w.Undo(Inverse{Action: "move", Path: "a.txt", From: "sub/b.txt"}, Digest("moving\n")); e != nil {
		t.Fatal(e)
	}
	if text, _ := w.Read("a.txt"); text != "moving\n" {
		t.Fatal("move undo did not restore the source")
	}
	if _, e = w.Read("sub/b.txt"); e == nil {
		t.Fatal("move undo left the destination")
	}
	if e = w.Undo(Inverse{Action: "rotate", Path: "a.txt"}, ""); e == nil {
		t.Fatal("unknown undo action accepted")
	}
}


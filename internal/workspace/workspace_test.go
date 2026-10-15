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

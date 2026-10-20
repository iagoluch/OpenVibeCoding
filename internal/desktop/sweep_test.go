//go:build desktop

package desktop

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// TestDesktopSweepStaleUserdata covers the conservative boot sweep: stale
// folders go, the current one stays, fresh ones stay (a concurrently starting
// shell is never raced), non-matching names are untouched, and no staging
// leftovers remain.
func TestDesktopSweepStaleUserdata(t *testing.T) {
	base := t.TempDir()
	old := time.Now().Add(-2 * time.Hour)
	mkdir := func(name string, mtime time.Time) string {
		dir := filepath.Join(base, name)
		if e := os.MkdirAll(dir, 0700); e != nil {
			t.Fatal(e)
		}
		if e := os.Chtimes(dir, mtime, mtime); e != nil {
			t.Fatal(e)
		}
		return dir
	}
	stale := mkdir("webview-deadbeef", old)
	fresh := mkdir("webview-cafebabe", time.Now())
	keep := mkdir("webview-00000000", old)
	other := mkdir("unrelated-stale", old)
	stray := filepath.Join(base, "webview-notadir")
	if e := os.WriteFile(stray, []byte("x"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.Chtimes(stray, old, old); e != nil {
		t.Fatal(e)
	}

	swept := sweepStaleUserData(base, keep, time.Now(), nil)
	if len(swept) != 1 || swept[0] != "webview-deadbeef" {
		t.Fatalf("sweep must remove exactly the stale folder, got %v", swept)
	}
	if _, e := os.Stat(stale); !os.IsNotExist(e) {
		t.Fatal("stale folder not removed")
	}
	for _, kept := range []string{fresh, keep, other, stray} {
		if _, e := os.Stat(kept); e != nil {
			t.Fatalf("sweep removed something it must keep: %v", e)
		}
	}
	left, _ := filepath.Glob(filepath.Join(base, ".sweep-*"))
	if len(left) != 0 {
		t.Fatalf("staging leftovers: %v", left)
	}
}

// TestDesktopSweepSkipsLockedFolder proves the conservative probe: while a
// live handle is open inside an old folder, the rename fails and the sweep
// leaves the folder untouched. POSIX allows renaming files that are open, so
// the probe only bites on Windows — the only platform the desktop shell
// ships on in M1.
func TestDesktopSweepSkipsLockedFolder(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("the lock probe is a Windows semantics; POSIX renames open files")
	}
	base := t.TempDir()
	dir := filepath.Join(base, "webview-live")
	if e := os.MkdirAll(dir, 0700); e != nil {
		t.Fatal(e)
	}
	handle, e := os.OpenFile(filepath.Join(dir, "lockfile"), os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		t.Fatal(e)
	}
	defer handle.Close()
	old := time.Now().Add(-2 * time.Hour)
	if e := os.Chtimes(dir, old, old); e != nil {
		t.Fatal(e)
	}
	if swept := sweepStaleUserData(base, filepath.Join(base, "webview-mine"), time.Now(), nil); len(swept) != 0 {
		t.Fatalf("a folder with a live handle must not be swept: %v", swept)
	}
	if _, e := os.Stat(dir); e != nil {
		t.Fatal("folder with a live handle was removed anyway")
	}
}

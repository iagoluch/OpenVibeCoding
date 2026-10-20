//go:build desktop

package desktop

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/iagoluch/OpenVibeCoding/internal/config"
	"github.com/iagoluch/OpenVibeCoding/internal/workspace"
)

// fakePicker stands in for the native dialog.
type fakePicker struct {
	folder  string
	err     error
	calls   atomic.Int32
	entered atomic.Bool
	overlap atomic.Int32
}

func (f *fakePicker) pickFolder() (string, error) {
	f.calls.Add(1)
	if f.entered.Swap(true) {
		f.overlap.Add(1)
	}
	time.Sleep(20 * time.Millisecond)
	f.entered.Store(false)
	return f.folder, f.err
}

// TestDesktopPickFolderBridgeIsConfined drives the single bound service with
// an injected dialog: a valid directory flows into a session with the
// canonical path, a missing path surfaces the daemon's validation error (and
// creates no session), a cancel is a no-op, a dialog error is returned to the
// page instead of swallowed, calls are serialized, and nothing beyond
// PickFolder is bound.
func TestDesktopPickFolderBridgeIsConfined(t *testing.T) {
	_, client, _ := startDaemon(t, config.Default())

	// Confinement: exactly one exported method on the bound type, exactly one
	// bound service.
	ui := newUI(&fakePicker{folder: t.TempDir()})
	var methods []string
	pointerType := reflect.TypeOf(ui)
	for i := range pointerType.NumMethod() {
		methods = append(methods, pointerType.Method(i).Name)
	}
	if len(methods) != 1 || methods[0] != "PickFolder" {
		t.Fatalf("the bridge must expose exactly PickFolder, got %v", methods)
	}
	if services := boundServices(ui); len(services) != 1 {
		t.Fatalf("the shell must bind exactly one service, got %d", len(services))
	}

	// Valid directory: the page flow creates a session with the canonical
	// (daemon-validated) path.
	dir := t.TempDir()
	picked, err := newUI(&fakePicker{folder: dir}).PickFolder()
	if err != nil || picked != dir {
		t.Fatalf("pick failed: %q %v", picked, err)
	}
	session, e := client.New(picked)
	if e != nil {
		t.Fatal(e)
	}
	w, e := workspace.Open(dir, false)
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	stored, e := client.Get(session)
	if e != nil || stored.Workspace != w.Path {
		t.Fatalf("session workspace %q is not the canonical %q (%v)", stored.Workspace, w.Path, e)
	}

	// Missing path: the daemon's validation error surfaces and no session is
	// created.
	before, e := client.List()
	if e != nil {
		t.Fatal(e)
	}
	missing := filepath.Join(t.TempDir(), "no", "such", "folder")
	picked, err = newUI(&fakePicker{folder: missing}).PickFolder()
	if err != nil || picked != missing {
		t.Fatalf("pick failed: %q %v", picked, err)
	}
	if _, e = client.New(picked); e == nil {
		t.Fatal("daemon accepted a missing workspace")
	}
	after, e := client.List()
	if e != nil {
		t.Fatal(e)
	}
	if len(after) != len(before) {
		t.Fatalf("missing path created a session: %d -> %d", len(before), len(after))
	}

	// Cancel: empty string, no error, a no-op for the page.
	picker := &fakePicker{}
	picked, err = newUI(picker).PickFolder()
	if picked != "" || err != nil || picker.calls.Load() != 1 {
		t.Fatalf("cancel must be an empty no-op: %q %v calls=%d", picked, err, picker.calls.Load())
	}

	// Dialog error: returned to the page, never swallowed.
	sentinel := errors.New("dialog refused")
	_, err = newUI(&fakePicker{err: sentinel}).PickFolder()
	if err == nil || err.Error() != sentinel.Error() {
		t.Fatalf("dialog error must reach the page, got %v", err)
	}

	// Serialization: concurrent calls never overlap; one modal at a time.
	picker = &fakePicker{folder: t.TempDir()}
	ui = newUI(picker)
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = ui.PickFolder()
		}()
	}
	wg.Wait()
	if picker.overlap.Load() != 0 {
		t.Fatalf("concurrent PickFolder calls overlapped %d times", picker.overlap.Load())
	}
}

// TestDesktopWebviewHygiene pins the webview posture: no remote debugging
// port, and a unique per-process user-data folder under the local app data,
// never roaming, removed at exit.
func TestDesktopWebviewHygiene(t *testing.T) {
	if args := browserArgs(); strings.Contains(strings.Join(args, " "), "remote-debugging-port") {
		t.Fatal("webview args must not contain a remote debugging port")
	}
	first, cleanupFirst, e := userDataPath()
	if e != nil {
		t.Fatal(e)
	}
	second, cleanupSecond, e := userDataPath()
	if e != nil {
		t.Fatal(e)
	}
	defer cleanupSecond()
	cleanupFirst()
	if first == second {
		t.Fatal("processes would share a webview data folder, which deadlocks")
	}
	for _, dir := range []string{first, second} {
		if !strings.Contains(filepath.ToSlash(dir), "/openvibecoding/webview-") {
			t.Fatalf("webview data must live under the local app data, got %q", dir)
		}
		if strings.Contains(strings.ToLower(dir), "roaming") {
			t.Fatalf("webview data must never be roaming: %q", dir)
		}
	}
	if _, e := os.Stat(first); !os.IsNotExist(e) {
		t.Fatal("webview data folder must be removed at exit")
	}
}

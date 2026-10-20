//go:build desktop

package desktop

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/iagoluch/OpenVibeCoding/internal/config"
	daemon "github.com/iagoluch/OpenVibeCoding/internal/runtime"
)

// fakeSupervisor records Ensure calls and answers from a constructor.
type fakeSupervisor struct {
	mu    sync.Mutex
	calls int
	spawn func() (*daemon.Client, error)
}

func (f *fakeSupervisor) Ensure(context.Context, string) (*daemon.Client, error) {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()
	return f.spawn()
}

func (f *fakeSupervisor) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// TestDesktopDaemonDeathRespawnsAndReloads kills a live daemon under the
// shell's monitor, with an injected supervisor: the shell must observe the
// death, respawn through the supervisor, retarget the proxy at the new
// daemon and reload the page. Afterwards the proxy must serve the respawned
// daemon, not the dead address.
func TestDesktopDaemonDeathRespawnsAndReloads(t *testing.T) {
	_, client, kill := startDaemon(t, config.Default())
	p := newProxy(nil, "")
	if e := p.retarget(client.Meta); e != nil {
		t.Fatal(e)
	}

	reloaded := make(chan struct{}, 8)
	respawnHome := t.TempDir() // created on the test goroutine: t.TempDir is not safe elsewhere
	fake := &fakeSupervisor{}
	fake.spawn = func() (*daemon.Client, error) {
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- daemon.Serve(ctx, respawnHome, config.Default()) }()
		t.Cleanup(func() {
			cancel()
			select {
			case <-done:
			case <-time.After(12 * time.Second):
				t.Error("respawned daemon shutdown timed out")
			}
		})
		return connectDaemon(respawnHome)
	}
	shell := &Shell{
		Home:       respawnHome,
		Supervisor: fake,
		Proxy:      p,
		Reload:     func() { reloaded <- struct{}{} },
		PollEvery:  15 * time.Millisecond,
	}

	monCtx, cancelMonitor := context.WithCancel(context.Background())
	defer cancelMonitor()
	finished := make(chan struct{})
	go func() {
		shell.Monitor(monCtx, client)
		close(finished)
	}()

	// Let the monitor see the daemon healthy at least once.
	time.Sleep(60 * time.Millisecond)
	// The daemon dies.
	kill()

	select {
	case <-reloaded:
	case <-time.After(5 * time.Second):
		t.Fatal("shell never reloaded after the daemon died")
	}
	if fake.count() < 1 {
		t.Fatal("shell never asked the supervisor to respawn")
	}

	// The proxy now serves the respawned daemon.
	front := httptest.NewServer(p)
	defer front.Close()
	session := storeNew(t, front.URL, t.TempDir())
	if session == "" {
		t.Fatal("proxy did not serve the respawned daemon")
	}

	cancelMonitor()
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("monitor did not stop")
	}
}

// tripwireTransport replaces the OLD client's transport after a respawn: any
// request through it is a tray (or anything else) firing at the dead daemon
// address, which is exactly the stale-capture bug.
type tripwireTransport struct{ t *testing.T }

func (tr tripwireTransport) RoundTrip(*http.Request) (*http.Response, error) {
	tr.t.Error("stale client: request attempted against the dead daemon address")
	return nil, errors.New("stale client used")
}

// TestDesktopTrayStopsRespawnedDaemon is the B1 regression: after the monitor
// respawns the daemon (a new port and token every incarnation), the tray's
// "Parar serviço" must stop the RESPAWNED daemon through the shared
// reference — the 200/stopping path — and must never attempt the old, dead
// address.
func TestDesktopTrayStopsRespawnedDaemon(t *testing.T) {
	_, client, kill := startDaemon(t, config.Default())
	ref := &atomic.Pointer[daemon.Client]{}
	ref.Store(client)
	p := newProxy(nil, "")
	if e := p.retarget(client.Meta); e != nil {
		t.Fatal(e)
	}

	var messages []string
	stop := stopAction(slog.New(slog.DiscardHandler), ref.Load, func(title, body string) {
		messages = append(messages, title+" / "+body)
	})

	reloaded := make(chan struct{}, 8)
	respawnHome := t.TempDir() // created on the test goroutine: t.TempDir is not safe elsewhere
	fake := &fakeSupervisor{}
	fake.spawn = func() (*daemon.Client, error) {
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- daemon.Serve(ctx, respawnHome, config.Default()) }()
		t.Cleanup(func() {
			cancel()
			select {
			case <-done:
			case <-time.After(12 * time.Second):
				t.Error("respawned daemon shutdown timed out")
			}
		})
		return connectDaemon(respawnHome)
	}
	shell := &Shell{
		Home:       respawnHome,
		Supervisor: fake,
		Proxy:      p,
		Daemon:     ref,
		Reload:     func() { reloaded <- struct{}{} },
		PollEvery:  15 * time.Millisecond,
	}

	monCtx, cancelMonitor := context.WithCancel(context.Background())
	defer cancelMonitor()
	finished := make(chan struct{})
	go func() {
		shell.Monitor(monCtx, client)
		close(finished)
	}()

	// Let the monitor see the daemon healthy at least once.
	time.Sleep(60 * time.Millisecond)
	// The daemon dies.
	kill()

	select {
	case <-reloaded:
	case <-time.After(5 * time.Second):
		t.Fatal("shell never reloaded after the daemon died")
	}
	cancelMonitor()
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("monitor did not stop")
	}

	// The shared reference must hold the respawned client, never the stale
	// capture the tray would otherwise keep.
	if ref.Load() == client {
		t.Fatal("shared reference kept the stale client after respawn")
	}
	// From here any request through the old client is the bug in the act.
	client.HTTP = &http.Client{Transport: tripwireTransport{t}}

	// The tray's stop must reach the respawned daemon: stopDaemon returns
	// nil only on HTTP 200, and the {"stopping":true} answer must actually
	// stop it — the respawned address goes dark.
	stop()
	if len(messages) != 0 {
		t.Fatalf("stop against the respawned daemon must succeed without a dialog, got %v", messages)
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := ref.Load().Status(false); err != nil {
			break // unreachable: the respawned daemon stopped
		}
		if time.Now().After(deadline) {
			t.Fatal("respawned daemon still answering after the tray stop")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

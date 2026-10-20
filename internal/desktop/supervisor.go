//go:build desktop

package desktop

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	daemon "github.com/iagoluch/OpenVibeCoding/internal/runtime"
)

// Supervisor attaches the shell to the shared runtime: reuse the running
// daemon or spawn one from this executable. It is an interface so the
// recovery loop is testable in-process — a real Ensure would re-exec the
// test binary, which cannot serve the runtime.
type Supervisor interface {
	Ensure(ctx context.Context, home string) (*daemon.Client, error)
}

// ensureSupervisor is the production supervisor: the runtime's own Ensure,
// reused exactly as the CLI uses it.
type ensureSupervisor struct{}

func (ensureSupervisor) Ensure(ctx context.Context, home string) (*daemon.Client, error) {
	return daemon.Ensure(ctx, home)
}

// Shell owns the daemon lifecycle around the window. It watches the daemon
// and, when it dies, respawns it, retargets the proxy and reloads the page.
// Closing the window never stops the daemon; only the tray asks it to stop.
type Shell struct {
	Home       string
	Supervisor Supervisor
	Proxy      *proxy
	// Daemon is the always-current daemon client, the same shared reference
	// the tray stops through. The monitor stores each incarnation on
	// respawn — every daemon has a new port and token — so the tray never
	// keeps a stale capture (the proxy's daemonRef swap, for the client).
	Daemon    *atomic.Pointer[daemon.Client]
	Reload    func()
	PollEvery time.Duration
	Log       *slog.Logger
}

// Monitor polls the daemon until ctx is done. On a failed health poll it asks
// the supervisor to attach again (respawning when nothing is running),
// retargets the proxy at the new address, publishes the fresh client to the
// shared reference the tray stops through and reloads the page so the UI
// reattaches. A failed respawn is retried on the next poll.
func (s *Shell) Monitor(ctx context.Context, client *daemon.Client) {
	if s.Log == nil {
		s.Log = slog.New(slog.DiscardHandler)
	}
	for ctx.Err() == nil {
		if _, err := client.Status(false); err != nil {
			s.Log.Warn("desktop: daemon lost, respawning", "error", err)
			if fresh, e := s.Supervisor.Ensure(ctx, s.Home); e != nil {
				s.Log.Error("desktop: respawn failed", "error", e)
			} else if e := s.Proxy.retarget(fresh.Meta); e != nil {
				s.Log.Error("desktop: retarget failed", "error", e)
			} else {
				client = fresh
				if s.Daemon != nil {
					s.Daemon.Store(fresh)
				}
				if s.Reload != nil {
					s.Reload()
				}
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(s.poll()):
		}
	}
}

func (s *Shell) poll() time.Duration {
	if s.PollEvery <= 0 {
		return 3 * time.Second
	}
	return s.PollEvery
}

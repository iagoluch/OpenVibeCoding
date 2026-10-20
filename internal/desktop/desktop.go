//go:build desktop

package desktop

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	daemon "github.com/iagoluch/OpenVibeCoding/internal/runtime"
)

//go:embed assets/tray-32.png
var trayIcon []byte

//go:embed assets/ovc-bridge.js
var bridgeAlias string

// randomID is a per-process identity component (client id, webview data
// folder). crypto/rand failing here means the platform is unusable.
func randomID() string {
	var b [8]byte
	if _, e := rand.Read(b[:]); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b[:])
}

// browserArgs are the extra WebView2 arguments. It must never contain
// --remote-debugging-port or any other debugging switch: the webview is not
// a debugging surface (the hygiene test pins this).
func browserArgs() []string { return nil }

// userDataPath returns a fresh per-process webview user-data folder under
// %LOCALAPPDATA% — never the roaming %APPDATA%, never the install directory.
// Two processes sharing one data folder deadlock the webview (POC-verified),
// so every process gets its own; the folder is removed best-effort at exit.
func userDataPath() (string, func(), error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		var e error
		if base, e = os.UserCacheDir(); e != nil {
			return "", nil, e
		}
	}
	dir := filepath.Join(base, "openvibecoding", "webview-"+randomID())
	if e := os.MkdirAll(dir, 0700); e != nil {
		return "", nil, e
	}
	return dir, func() { _ = os.RemoveAll(dir) }, nil
}

// shellLogger mirrors the CLI's debug switch for the desktop process.
func shellLogger() *slog.Logger {
	level := slog.LevelInfo
	if os.Getenv("OPENVIBECODING_DEBUG") == "1" {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}

// Run opens the desktop window: the third thin client of the daemon. It
// attaches to the shared runtime (spawning it from this executable when
// absent), serves the daemon's own web UI through the confined reverse proxy
// and keeps the bearer token server-side. Closing the window or quitting the
// shell leaves the runtime running; the tray's "Parar serviço" is the only
// way the shell asks the daemon to stop.
func Run(ctx context.Context, home string) error {
	log := shellLogger()
	supervisor := ensureSupervisor{}
	client, e := supervisor.Ensure(ctx, home)
	if e != nil {
		return fmt.Errorf("runtime: %w", e)
	}
	p := newProxy(log, assetHost)
	if e := p.retarget(client.Meta); e != nil {
		return e
	}
	udf, cleanup, e := userDataPath()
	if e != nil {
		return e
	}
	defer cleanup() // best effort: the webview is closed by then
	// Boot hygiene: remove user-data folders orphaned by dead shells (a
	// killed process never runs the cleanup above). Best effort, logged.
	sweepStaleUserData(filepath.Dir(udf), udf, time.Now(), log)
	app := application.New(application.Options{
		Name:     "OpenVibeCoding",
		Services: boundServices(newUI(wailsPicker{})),
		Assets:   application.AssetOptions{Handler: p, DisableLogging: true},
		Windows: application.WindowsOptions{
			WebviewUserDataPath:   udf,
			AdditionalBrowserArgs: browserArgs(),
		},
	})
	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:   "main",
		Title:  "OpenVibeCoding",
		Width:  1280,
		Height: 860,
	})
	// The tray stops the daemon through this always-current reference: the
	// monitor republishes each respawned incarnation — every daemon has a
	// new port and token — so a click never posts to a dead address.
	var current atomic.Pointer[daemon.Client]
	current.Store(client)
	newTray(app, win, current.Load, trayIcon, dialogMessenger(app), log)
	shell := &Shell{
		Home:       home,
		Supervisor: supervisor,
		Proxy:      p,
		Daemon:     &current,
		Reload:     win.Reload,
		PollEvery:  3 * time.Second,
		Log:        log,
	}
	monCtx, cancelMonitor := context.WithCancel(ctx)
	defer cancelMonitor()
	go shell.Monitor(monCtx, client)
	go func() {
		<-ctx.Done()
		app.Quit()
	}()
	e = app.Run()
	cancelMonitor()
	return e
}

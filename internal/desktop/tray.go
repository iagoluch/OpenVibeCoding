//go:build desktop

package desktop

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	daemon "github.com/iagoluch/OpenVibeCoding/internal/runtime"
)

// errDaemonBusy reports that the daemon refused to stop because a turn is
// still running (HTTP 409).
var errDaemonBusy = errors.New("runtime has active tasks")

// The tray speaks pt-BR, the product's interface language.
const (
	trayTitle = "OpenVibeCoding"
	// trayBusyBody and trayStopFailBody are surfaced as dialogs: a stop that
	// is refused or fails must never be silent.
	trayBusyBody     = "Há um trabalho em andamento — pare a tarefa primeiro"
	trayStopFailBody = "Não foi possível parar o serviço"
)

// messenger shows a message dialog to the user. It is a type so tests can
// capture what the tray would say without opening a window.
type messenger func(title, message string)

// stopDaemon asks the runtime to stop. It never force-kills: a busy daemon
// answers 409, keeps running and surfaces as errDaemonBusy.
func stopDaemon(ctx context.Context, client *daemon.Client) error {
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, client.Meta.URL+"/stop", strings.NewReader("{}"))
	if e != nil {
		return e
	}
	req.Header.Set("Authorization", "Bearer "+client.Meta.Token)
	req.Header.Set("X-OpenVibeCoding-Client", client.ID)
	req.Header.Set("Content-Type", "application/json")
	resp, e := client.HTTP.Do(req)
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusConflict:
		return errDaemonBusy
	default:
		return fmt.Errorf("runtime HTTP %d", resp.StatusCode)
	}
}

// stopAction is what "Parar serviço" does: ask the daemon to stop and surface
// the refusal to the user when a turn is still running — and the failure when
// the runtime cannot be reached. The client is loaded per click through
// current: every respawned daemon incarnation has a new port and token, so a
// client captured at startup would post to a dead address.
func stopAction(log *slog.Logger, current func() *daemon.Client, message messenger) func() {
	return func() {
		client := current()
		if client == nil {
			log.Error("desktop: stop failed, runtime not attached")
			message(trayTitle, trayStopFailBody)
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		switch err := stopDaemon(ctx, client); {
		case err == nil:
			log.Info("desktop: stop requested from the tray")
		case errors.Is(err, errDaemonBusy):
			log.Info("desktop: stop refused, daemon busy")
			message(trayTitle, trayBusyBody)
		default:
			log.Error("desktop: stop failed", "error", err)
			message(trayTitle, trayStopFailBody)
		}
	}
}

// newTray installs the tray icon and menu: focus the window, stop the
// service, quit the shell. Quitting never stops the daemon, matching the
// `ovc --ui` semantics; "Parar serviço" is the only path that asks, always
// against the daemon the monitor last attached (current).
func newTray(app *application.App, win *application.WebviewWindow, current func() *daemon.Client, icon []byte, message messenger, log *slog.Logger) {
	tray := app.SystemTray.New()
	tray.SetIcon(icon)
	stop := stopAction(log, current, message)
	menu := application.NewMenuFromItems(
		application.NewMenuItem("Mostrar").OnClick(func(*application.Context) {
			win.Show()
			win.Focus()
		}),
		application.NewMenuItemSeparator(),
		application.NewMenuItem("Parar serviço").OnClick(func(*application.Context) { stop() }),
		application.NewMenuItemSeparator(),
		application.NewMenuItem("Sair").OnClick(func(*application.Context) { app.Quit() }),
	)
	tray.SetMenu(menu)
	menu.Update()
}

// dialogMessenger shows a native info dialog.
func dialogMessenger(app *application.App) messenger {
	return func(title, message string) {
		app.Dialog.Info().SetTitle(title).SetMessage(message).Show()
	}
}

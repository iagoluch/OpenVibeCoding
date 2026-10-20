//go:build desktop

package desktop

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/iagoluch/OpenVibeCoding/internal/config"
	daemon "github.com/iagoluch/OpenVibeCoding/internal/runtime"
)

// TestDesktopStopRefusedWhileBusy parks a turn on an approval and asks the
// daemon to stop the way the tray does: the 409 must surface as the pt-BR
// message dialog and the daemon must stay alive. After the turn is released,
// the same request succeeds.
func TestDesktopStopRefusedWhileBusy(t *testing.T) {
	provider := fakeProvider(t)
	defer provider.Close()
	ws := canonicalWorkspace(t, t.TempDir())
	_, client, _ := startDaemon(t, fixtureConfig(provider.URL))
	session, e := client.New(ws)
	if e != nil {
		t.Fatal(e)
	}
	turn := parkTurn(t, client.Meta.URL, "tray-owner", client.Meta.Token, session, ws)
	defer turn.release()

	var messages []string
	stop := stopAction(slog.New(slog.DiscardHandler), func() *daemon.Client { return client }, func(title, body string) {
		messages = append(messages, title+" / "+body)
	})
	stop()
	if len(messages) != 1 || !strings.Contains(messages[0], "Há um trabalho em andamento") {
		t.Fatalf("busy stop must surface the pt-BR refusal, got %v", messages)
	}
	if _, err := client.Status(false); err != nil {
		t.Fatal("daemon must stay alive after refusing to stop")
	}

	// Release the turn; once it unwinds, the same stop request succeeds.
	turn.release()
	deadline := time.Now().Add(10 * time.Second)
	for {
		err := stopDaemon(t.Context(), client)
		if err == nil {
			return
		}
		if !errors.Is(err, errDaemonBusy) {
			t.Fatalf("stop after release failed: %v", err)
		}
		if time.Now().After(deadline) {
			t.Fatal("daemon stayed busy after the turn was released")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestDesktopStopSurfacesUnreachableDaemon kills the daemon and asks the tray
// way: a stop against a dead address must never be silent — the user gets the
// pt-BR failure dialog instead of a log line nobody reads.
func TestDesktopStopSurfacesUnreachableDaemon(t *testing.T) {
	_, client, stop := startDaemon(t, config.Default())
	stop() // the daemon dies; the tray's captured address goes dead

	var messages []string
	act := stopAction(slog.New(slog.DiscardHandler), func() *daemon.Client { return client }, func(title, body string) {
		messages = append(messages, title+" / "+body)
	})
	act()
	if len(messages) != 1 || !strings.Contains(messages[0], "Não foi possível parar o serviço") {
		t.Fatalf("stop against a dead daemon must surface the pt-BR failure, got %v", messages)
	}
}

// TestDesktopApprovalOwnershipThroughTransport proves approval ownership
// across the transport: an approval issued to a foreign client identity
// cannot be answered through the proxy (403 from the daemon), while a turn
// the proxy itself started can be approved through it.
func TestDesktopApprovalOwnershipThroughTransport(t *testing.T) {
	provider := fakeProvider(t)
	defer provider.Close()
	_, client, _ := startDaemon(t, fixtureConfig(provider.URL))
	foreignWS := canonicalWorkspace(t, t.TempDir())
	foreignSession, e := client.New(foreignWS)
	if e != nil {
		t.Fatal(e)
	}
	// A turn owned by someone else, started directly against the daemon.
	foreign := parkTurn(t, client.Meta.URL, "foreign-owner", client.Meta.Token, foreignSession, foreignWS)
	defer foreign.release()

	p := newProxy(nil, "")
	if e := p.retarget(client.Meta); e != nil {
		t.Fatal(e)
	}
	front := httptest.NewServer(p)
	defer front.Close()

	// The proxy's identity is not the turn owner: the daemon must refuse.
	resp, err := http.Post(front.URL+"/approve", "application/json",
		strings.NewReader(fmt.Sprintf(`{"ID":%q,"Allow":true}`, foreign.approval)))
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("foreign approval through the proxy must be 403, got %d", resp.StatusCode)
	}

	// A turn started THROUGH the proxy is owned by the proxy identity and is
	// approved through it.
	ownedWS := canonicalWorkspace(t, t.TempDir())
	ownedSession, e := client.New(ownedWS)
	if e != nil {
		t.Fatal(e)
	}
	owned := parkTurn(t, front.URL, "", "", ownedSession, ownedWS)
	defer owned.release()
	resp, err = http.Post(front.URL+"/approve", "application/json",
		strings.NewReader(fmt.Sprintf(`{"ID":%q,"Allow":true}`, owned.approval)))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"ok":true`) {
		t.Fatalf("own approval through the proxy failed: %d %s", resp.StatusCode, body)
	}
}

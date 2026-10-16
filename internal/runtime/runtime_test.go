package runtime

import (
	"context"
	"encoding/json"
	"github.com/iagoluch/OpenVibeCoding/internal/agent"
	"github.com/iagoluch/OpenVibeCoding/internal/config"
	"github.com/iagoluch/OpenVibeCoding/internal/workspace"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func startFixture(t *testing.T, c config.Config) (string, *Client) {
	t.Helper()
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Serve(ctx, dir, c) }()
	t.Cleanup(func() {
		cancel()
		select {
		case e := <-done:
			if e != nil {
				t.Error(e)
			}
		case <-time.After(12 * time.Second):
			t.Error("daemon shutdown timed out")
		}
	})
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if client, e := Connect(dir); e == nil {
			return dir, client
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("runtime not ready")
	return "", nil
}
func TestRuntimeAuthReuseAndSessions(t *testing.T) {
	dir, c := startFixture(t, config.Default())
	other, e := Connect(dir)
	if e != nil {
		t.Fatal(e)
	}
	if other.Meta.PID != c.Meta.PID || other.ID == c.ID {
		t.Fatal("runtime not shared or identities reused")
	}
	id, e := c.New(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	if e = other.Rename(id, "shared"); e != nil {
		t.Fatal(e)
	}
	s, e := c.Get(id)
	if e != nil || s.Title != "shared" {
		t.Fatal(s, e)
	}
	r, e := http.Post(c.Meta.URL+"/health", "application/json", strings.NewReader("{}"))
	if e != nil {
		t.Fatal(e)
	}
	r.Body.Close()
	if r.StatusCode != 401 {
		t.Fatal("unauthenticated IPC accepted")
	}
	if e = Serve(context.Background(), dir, config.Default()); e == nil {
		t.Fatal("duplicate runtime started")
	}
}
func TestRuntimeTurnOwnership(t *testing.T) {
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case entered <- struct{}{}:
		default:
		}
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Observed done\"}}]}\n\ndata: [DONE]\n\n"))
	}))
	defer provider.Close()
	cfg := config.Default()
	p := cfg.Providers["local"]
	p.BaseURL = provider.URL
	cfg.Providers["local"] = p
	cfg.Provider = "local"
	cfg.Model = "qwen3:8b"
	dir, c := startFixture(t, cfg)
	other, _ := Connect(dir)
	ws, e := workspace.Open(t.TempDir(), false)
	if e != nil {
		t.Fatal(e)
	}
	defer ws.Close()
	id, _ := c.New(ws.Path)
	engine := &agent.Engine{Config: cfg, Store: c, Workspace: ws, Session: id}
	done := make(chan error, 1)
	go func() { done <- c.Run(context.Background(), engine, "inspect", func(agent.Event) {}) }()
	<-entered
	if e = other.Run(context.Background(), engine, "conflict", func(agent.Event) {}); e == nil {
		t.Fatal("concurrent session accepted")
	}
	if e = other.Delete(id); e == nil {
		t.Fatal("deleted running session")
	}
	close(release)
	if e = <-done; e != nil {
		t.Fatal(e)
	}
	// The runtime releases the session when the turn handler returns, which can
	// trail the done event the client returns on; wait for the release before
	// reading the session back.
	var b []byte
	for deadline := time.Now().Add(5 * time.Second); ; {
		if b, e = c.Export(id); e == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("export failed", e)
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !json.Valid(b) {
		t.Fatalf("export is not valid JSON: %q", b)
	}
}

func TestRemoteApprovalAppliesOnlyAfterDecision(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b struct{ Messages []struct{ Role string } }
		json.NewDecoder(r.Body).Decode(&b)
		hasTool := false
		for _, m := range b.Messages {
			if m.Role == "tool" {
				hasTool = true
			}
		}
		w.Header().Set("Content-Type", "text/event-stream")
		if !hasTool {
			w.Write([]byte("data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"review1\",\"function\":{\"name\":\"write_file\",\"arguments\":\"{\\\"path\\\":\\\"approved.txt\\\",\\\"content\\\":\\\"approved\\\"}\"}}]}}]}\n\ndata: [DONE]\n\n"))
		} else {
			w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Observed file write\"}}]}\n\ndata: [DONE]\n\n"))
		}
	}))
	defer provider.Close()
	cfg := config.Default()
	p := cfg.Providers["local"]
	p.BaseURL = provider.URL
	cfg.Providers["local"] = p
	cfg.Provider = "local"
	cfg.Model = "qwen3:8b"
	_, c := startFixture(t, cfg)
	ws, e := workspace.Open(t.TempDir(), false)
	if e != nil {
		t.Fatal(e)
	}
	defer ws.Close()
	id, _ := c.New(ws.Path)
	engine := &agent.Engine{Config: cfg, Store: c, Workspace: ws, Session: id}
	approved := false
	e = c.Run(context.Background(), engine, "write", func(event agent.Event) {
		if event.Decision != nil {
			if _, e := ws.Read("approved.txt"); e == nil {
				t.Error("file written before remote approval")
			}
			approved = true
			event.Decision <- true
		}
	})
	if e != nil {
		t.Fatal(e)
	}
	text, e := ws.Read("approved.txt")
	if !approved || e != nil || text != "approved" {
		t.Fatal("remote approved operation failed", e)
	}
}

// TestApprovalSurvivesAnotherTurnOfTheSameClient covers the end of a turn: it
// must retire only the approvals it handed out. The same client may have two
// sessions running, and dropping the other one's pending approval answers its
// /approve with a 403 and leaves that turn waiting.
func TestApprovalSurvivesAnotherTurnOfTheSameClient(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b struct{ Messages []struct{ Role string } }
		json.NewDecoder(r.Body).Decode(&b)
		hasTool := false
		for _, m := range b.Messages {
			if m.Role == "tool" {
				hasTool = true
			}
		}
		w.Header().Set("Content-Type", "text/event-stream")
		if !hasTool {
			w.Write([]byte("data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"p1\",\"function\":{\"name\":\"write_file\",\"arguments\":\"{\\\"path\\\":\\\"approved.txt\\\",\\\"content\\\":\\\"approved\\\"}\"}}]}}]}\n\ndata: [DONE]\n\n"))
			return
		}
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Observed write\"}}]}\n\ndata: [DONE]\n\n"))
	}))
	defer provider.Close()
	cfg := config.Default()
	p := cfg.Providers["local"]
	p.BaseURL = provider.URL
	cfg.Providers["local"] = p
	cfg.Provider = "local"
	cfg.Model = "qwen3:8b"
	_, c := startFixture(t, cfg)
	spaces := make([]*workspace.Workspace, 2)
	for i := range spaces {
		ws, e := workspace.Open(t.TempDir(), false)
		if e != nil {
			t.Fatal(e)
		}
		defer ws.Close()
		spaces[i] = ws
	}
	// waiting reports which turn reached its approval; release answers it.
	waiting := make(chan string, len(spaces))
	release := make([]chan struct{}, len(spaces))
	done := make([]chan error, len(spaces))
	for i, ws := range spaces {
		session, e := c.New(ws.Path)
		if e != nil {
			t.Fatal(e)
		}
		engine := &agent.Engine{Config: cfg, Store: c, Workspace: ws, Session: session}
		release[i] = make(chan struct{})
		done[i] = make(chan error, 1)
		go func(engine *agent.Engine, release chan struct{}, done chan error) {
			done <- c.Run(context.Background(), engine, "write", func(event agent.Event) {
				if event.Decision == nil {
					return
				}
				waiting <- engine.Session
				<-release
				event.Decision <- true
			})
		}(engine, release[i], done[i])
	}
	// Both turns are parked on their own approval before either one ends.
	parked := map[string]bool{}
	for range spaces {
		select {
		case session := <-waiting:
			parked[session] = true
		case <-time.After(20 * time.Second):
			t.Fatal("a turn never asked for approval")
		}
	}
	if len(parked) != len(spaces) {
		t.Fatalf("the two turns did not park separately: %v", parked)
	}
	close(release[0])
	if e := <-done[0]; e != nil {
		t.Fatalf("first turn failed: %v", e)
	}
	// The first turn ended and cleaned up after itself; the second approval is
	// still pending and its owner is still allowed to answer it.
	close(release[1])
	if e := <-done[1]; e != nil {
		t.Fatalf("the second turn lost its approval to the first one's cleanup: %v", e)
	}
	for i, ws := range spaces {
		if text, e := ws.Read("approved.txt"); e != nil || text != "approved" {
			t.Fatalf("turn %d did not apply the approved write: %q %v", i, text, e)
		}
	}
}

func TestRemoteUndoReversesAppliedChange(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b struct{ Messages []struct{ Role string } }
		json.NewDecoder(r.Body).Decode(&b)
		hasTool := false
		for _, m := range b.Messages {
			if m.Role == "tool" {
				hasTool = true
			}
		}
		w.Header().Set("Content-Type", "text/event-stream")
		if !hasTool {
			w.Write([]byte("data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"u1\",\"function\":{\"name\":\"write_file\",\"arguments\":\"{\\\"path\\\":\\\"undo.txt\\\",\\\"content\\\":\\\"created\\\"}\"}}]}}]}\n\ndata: [DONE]\n\n"))
		} else {
			w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Observed\"}}]}\n\ndata: [DONE]\n\n"))
		}
	}))
	defer provider.Close()
	cfg := config.Default()
	p := cfg.Providers["local"]
	p.BaseURL = provider.URL
	cfg.Providers["local"] = p
	cfg.Provider = "local"
	cfg.Model = "qwen3:8b"
	_, c := startFixture(t, cfg)
	ws, e := workspace.Open(t.TempDir(), false)
	if e != nil {
		t.Fatal(e)
	}
	defer ws.Close()
	id, _ := c.New(ws.Path)
	engine := &agent.Engine{Config: cfg, Store: c, Workspace: ws, Session: id}
	if e = c.Run(context.Background(), engine, "create file", func(event agent.Event) {
		if event.Decision != nil {
			event.Decision <- true
		}
	}); e != nil {
		t.Fatal(e)
	}
	if text, err := ws.Read("undo.txt"); err != nil || text != "created" {
		t.Fatalf("write not applied: %v %q", err, text)
	}
	summary, e := c.Undo(id)
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(summary, "undo.txt") {
		t.Fatalf("summary: %q", summary)
	}
	if _, e = ws.Read("undo.txt"); e == nil {
		t.Fatal("undo did not remove the created file")
	}
	if _, e = c.Undo(id); e == nil {
		t.Fatal("second undo accepted")
	}
}


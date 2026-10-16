// Package runtime provides authenticated loopback IPC on all supported OSes.
// The daemon owns SQLite and serializes turns per session and workspace.
package runtime

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gofrs/flock"
	"github.com/iagoluch/OpenVibeCoding/internal/agent"
	"github.com/iagoluch/OpenVibeCoding/internal/automation"
	"github.com/iagoluch/OpenVibeCoding/internal/config"
	"github.com/iagoluch/OpenVibeCoding/internal/provider"
	"github.com/iagoluch/OpenVibeCoding/internal/store"
	"github.com/iagoluch/OpenVibeCoding/internal/webui"
	"github.com/iagoluch/OpenVibeCoding/internal/workspace"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const Protocol = 1

type Metadata struct {
	URL, Token string
	Version    int
	PID        int
}

func random() string {
	var b [32]byte
	if _, e := rand.Read(b[:]); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b[:])
}

type approval struct {
	owner    string
	decision chan bool
}
type Server struct {
	ctx       context.Context
	log       *slog.Logger
	db        *store.Store
	ui        http.Handler
	config    config.Config
	dir       string
	addr      string
	workspace string
	pairings  map[string]time.Time
	mu        sync.Mutex
	busy      map[string]bool
	approvals map[string]approval
	stop      chan struct{}
	token     string
}
type request struct {
	Op, ID, Text, Workspace string
	Message                 provider.Message
	Summary                 store.Summary
	Usage                   store.Usage
}
type response struct {
	Value json.RawMessage `json:"value,omitempty"`
	Error string          `json:"error,omitempty"`
}
type turn struct {
	Permissions                                          config.Permissions
	Automation                                           automation.Settings
	Session, Prompt, Provider, Model, Harness, Workspace string
	Shell                                                bool
}
type wireEvent struct {
	Kind, Text, Approval string
	Usage                provider.Usage
}

func Serve(ctx context.Context, dir string, c config.Config) error {
	lock := flock.New(filepath.Join(dir, "runtime.lock"))
	ok, e := lock.TryLock()
	if e != nil {
		return e
	}
	if !ok {
		return errors.New("compatible runtime already running")
	}
	defer lock.Close()
	db, e := store.Open(dir)
	if e != nil {
		return e
	}
	defer db.Close()
	listener, e := net.Listen("tcp4", "127.0.0.1:0")
	if e != nil {
		return e
	}
	defer listener.Close()
	logFile, e := os.OpenFile(filepath.Join(dir, "runtime.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	defer logFile.Close()
	level := slog.LevelInfo
	if c.Debug {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(logFile, &slog.HandlerOptions{Level: level}))
	serverCtx, cancelServer := context.WithCancel(ctx)
	defer cancelServer()
	ui, e := webui.Handler()
	if e != nil {
		return e
	}
	s := &Server{ctx: serverCtx, log: logger, db: db, ui: ui, dir: dir, addr: listener.Addr().String(), config: c, pairings: map[string]time.Time{}, busy: map[string]bool{}, approvals: map[string]approval{}, stop: make(chan struct{}), token: random()}
	meta := Metadata{URL: "http://" + listener.Addr().String(), Token: s.token, Version: Protocol, PID: os.Getpid()}
	b, _ := json.Marshal(meta)
	path := filepath.Join(dir, "runtime.json")
	tmp, e := os.CreateTemp(dir, ".runtime-*")
	if e != nil {
		return e
	}
	defer os.Remove(tmp.Name())
	if _, e = tmp.Write(b); e != nil {
		tmp.Close()
		return e
	}
	tmp.Close()
	if e = os.Rename(tmp.Name(), path); e != nil {
		return e
	}
	defer os.Remove(path)
	httpServer := &http.Server{Handler: s, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 8192}
	done := make(chan error, 1)
	go func() { done <- httpServer.Serve(listener) }()
	select {
	case <-ctx.Done():
	case <-s.stop:
	case e = <-done:
		return e
	}
	cancelServer()
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return httpServer.Shutdown(shutdown)
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// The browser defence headers are applied once, here, so error paths and
	// the /run stream carry them too. The UI handler refines Cache-Control for
	// the page and its assets, which may be revalidated instead of stored.
	h := w.Header()
	h.Set("Cache-Control", "no-store")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "no-referrer")
	if r.Host != s.addr {
		http.Error(w, "forbidden host", 403)
		return
	}
	if !sameOrigin(r) {
		http.Error(w, "unauthorized", 401)
		return
	}
	if r.Method == "GET" {
		s.serveUI(w, r)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "POST required", 405)
		return
	}
	bearer := subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")), []byte(s.token)) == 1
	if !bearer && !s.cookieAuth(r) {
		http.Error(w, "unauthorized", 401)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	defer r.Body.Close()
	owner := r.Header.Get("X-OpenVibeCoding-Client")
	if owner == "" {
		http.Error(w, "client identity required", 400)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	s.log.Debug("authenticated_request", "method", r.Method)
	switch r.URL.Path {
	case "/health":
		s.mu.Lock()
		n := len(s.busy) / 2
		s.mu.Unlock()
		json.NewEncoder(w).Encode(map[string]any{"protocol": Protocol, "active_tasks": n})
	case "/stop":
		s.mu.Lock()
		defer s.mu.Unlock()
		if len(s.busy) > 0 {
			http.Error(w, "runtime has active tasks; cancel them before stopping", 409)
			return
		}
		select {
		case <-s.stop:
		default:
			close(s.stop)
		}
		json.NewEncoder(w).Encode(map[string]bool{"stopping": true})
	case "/pair":
		if !bearer {
			http.Error(w, "unauthorized", 401)
			return
		}
		s.pairCreate(w, r)
	case "/approve":
		var a struct {
			ID    string
			Allow bool
		}
		if json.NewDecoder(r.Body).Decode(&a) != nil {
			http.Error(w, "invalid approval", 400)
			return
		}
		s.mu.Lock()
		pending, ok := s.approvals[a.ID]
		if ok && pending.owner == owner {
			delete(s.approvals, a.ID)
		} else {
			ok = false
		}
		s.mu.Unlock()
		if !ok {
			http.Error(w, "approval not owned by this client", 403)
			return
		}
		pending.decision <- a.Allow
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	case "/store":
		s.storage(w, r)
	case "/run":
		s.run(w, r, owner)
	case "/ui/info":
		s.uiInfo(w, r)
	case "/ui/setup":
		s.uiSetup(w, r)
	case "/ui/select":
		s.uiSelect(w, r)
	default:
		http.NotFound(w, r)
	}
}

// sameOrigin reports whether a request may address the daemon on behalf of the
// browser. Requests without Origin and fetch metadata (the CLI and TUI) pass;
// a browser request must come from the daemon's own origin. This replaces the
// earlier blanket refusal of any Origin header, so the paired page can use the
// API while cross-site pages still cannot.
func sameOrigin(r *http.Request) bool {
	if o := r.Header.Get("Origin"); o != "" && o != "http://"+r.Host {
		return false
	}
	switch r.Header.Get("Sec-Fetch-Site") {
	case "", "same-origin", "none":
		return true
	}
	return false
}

// cookieAuth reports whether the request carries the cookie minted by a pairing
// link. It is only consulted after the origin policy passed. Another local page
// can send a cookie of the same name from its own port, and browsers disagree
// on which one comes first, so every ovc_ui cookie is considered and any value
// matching the token authenticates.
func (s *Server) cookieAuth(r *http.Request) bool {
	for _, c := range r.CookiesNamed("ovc_ui") {
		if subtle.ConstantTimeCompare([]byte(c.Value), []byte(s.token)) == 1 {
			return true
		}
	}
	return false
}
func (s *Server) storage(w http.ResponseWriter, r *http.Request) {
	var req request
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		http.Error(w, "invalid storage request", 400)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	switch req.Op {
	case "delete", "rename", "recover", "audit", "export", "undo":
		if s.busy["session:"+req.ID] {
			http.Error(w, "session is running", 409)
			return
		}
	}
	var value any
	var e error
	switch req.Op {
	case "new":
		var ws *workspace.Workspace
		ws, e = workspace.Open(req.Workspace, false)
		if e == nil {
			value, e = s.db.New(ws.Path)
			ws.Close()
		}
	case "list":
		value, e = s.db.List()
	case "get":
		value, e = s.db.Get(req.ID)
	case "history":
		value, e = s.db.History(req.ID)
	case "audit":
		e = s.db.Audit(req.ID, s.config.Redact(req.Text))
	case "rename":
		e = s.db.Rename(req.ID, s.config.Redact(req.Text))
	case "delete":
		e = s.db.Delete(req.ID)
	case "recover":
		e = s.db.Recover(req.ID)
	case "summary":
		value, e = s.db.Summary(req.ID)
	case "usage":
		value, e = s.db.Usage(req.ID)
	case "undo":
		var session store.Session
		session, e = s.db.Get(req.ID)
		if e == nil && s.busy["workspace:"+session.Workspace] {
			e = errors.New("workspace is running; wait for the current turn to finish")
		}
		if e == nil {
			value, e = s.undo(session)
		}
	case "export":
		value, e = s.db.Export(req.ID)
	default:
		e = errors.New("storage mutation is owned by runtime")
	}
	out := response{}
	if e != nil {
		out.Error = s.config.Redact(e.Error())
	} else {
		out.Value, _ = json.Marshal(value)
	}
	json.NewEncoder(w).Encode(out)
}

// undo reverses the most recent applied change of one session inside its
// workspace, then marks the change undone and audits the result.
func (s *Server) undo(session store.Session) (string, error) {
	change, e := s.db.LastChange(session.ID)
	if e != nil {
		return "", e
	}
	var inverse workspace.Inverse
	if e = json.Unmarshal([]byte(change.Inverse), &inverse); e != nil {
		return "", errors.New("stored undo record is unreadable")
	}
	ws, e := workspace.Open(session.Workspace, false)
	if e != nil {
		return "", errors.New("workspace unavailable")
	}
	defer ws.Close()
	if e = ws.Undo(inverse, change.Expect); e != nil {
		return "", e
	}
	if e = s.db.MarkChangeUndone(session.ID, change.ID); e != nil {
		return "", e
	}
	summary := "Undid the last applied change: " + change.Path
	if e = s.db.Audit(session.ID, summary); e != nil {
		return "", e
	}
	return summary, nil
}
func (s *Server) run(w http.ResponseWriter, r *http.Request, owner string) {
	var req turn
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		http.Error(w, "invalid turn", 400)
		return
	}
	session, e := s.db.Get(req.Session)
	if e != nil || session.Workspace != req.Workspace {
		http.Error(w, "invalid session workspace", 400)
		return
	}
	s.mu.Lock()
	c := s.config
	s.mu.Unlock()
	c.Provider = req.Provider
	c.Model = req.Model
	c.Harness = req.Harness
	c.Permissions = req.Permissions
	c.Automation.Browser = req.Automation.Browser
	c.Automation.Desktop = req.Automation.Desktop
	c.Automation.Clipboard = req.Automation.Clipboard
	if e = c.Validate(); e != nil {
		http.Error(w, "invalid provider/model/harness", 400)
		return
	}
	s.mu.Lock()
	sk, wk := "session:"+req.Session, "workspace:"+session.Workspace
	if s.busy[sk] || s.busy[wk] {
		s.mu.Unlock()
		http.Error(w, "session or workspace already running", 409)
		return
	}
	s.busy[sk] = true
	s.busy[wk] = true
	s.mu.Unlock()
	defer func() { s.mu.Lock(); delete(s.busy, sk); delete(s.busy, wk); s.mu.Unlock() }()
	ws, e := workspace.Open(session.Workspace, req.Shell)
	if e != nil {
		http.Error(w, "workspace unavailable", 400)
		return
	}
	defer ws.Close()
	if err := s.db.Audit(req.Session, "task_started provider="+req.Provider+" model="+req.Model+" harness="+req.Harness); err != nil {
		http.Error(w, "audit unavailable", 500)
		return
	}
	s.log.Info("task_started", "session", req.Session, "provider", req.Provider, "model", req.Model)
	engine := agent.Engine{Config: c, Store: s.db, Workspace: ws, Session: req.Session}
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Minute)
	stopDisconnect := context.AfterFunc(r.Context(), cancel)
	defer stopDisconnect()
	defer cancel()
	enc := json.NewEncoder(w)
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unavailable", 500)
		return
	}
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.WriteHeader(200)
	flusher.Flush()
	// issued collects the approvals this turn handed out: the same client may
	// run another session at the same time, and clearing its pending approvals
	// would answer its /approve with a 403 and leave that turn waiting.
	var issued []string
	e = engine.Run(ctx, req.Prompt, func(event agent.Event) {
		out := wireEvent{Kind: event.Kind, Text: c.Redact(event.Text), Usage: event.Usage}
		if event.Decision != nil {
			out.Approval = random()
			s.mu.Lock()
			s.approvals[out.Approval] = approval{owner, event.Decision}
			issued = append(issued, out.Approval)
			s.mu.Unlock()
			defer func() {
				if ctx.Err() != nil {
					s.mu.Lock()
					delete(s.approvals, out.Approval)
					s.mu.Unlock()
				}
			}()
		}
		if enc.Encode(out) != nil {
			cancel()
		}
		flusher.Flush()
	})
	s.mu.Lock()
	for _, id := range issued {
		delete(s.approvals, id)
	}
	s.mu.Unlock()
	if auditErr := s.db.Audit(req.Session, fmt.Sprintf("task_finished success=%t", e == nil)); auditErr != nil && e == nil {
		e = auditErr
	}
	s.log.Info("task_finished", "session", req.Session, "success", e == nil)
	out := wireEvent{Kind: "done"}
	if e != nil {
		out.Text = c.Redact(e.Error())
	}
	enc.Encode(out)
	flusher.Flush()
}

// ErrNotRunning reports that no runtime metadata exists for the configured
// directory, which is the ordinary state before the first ovc start.
var ErrNotRunning = errors.New("runtime is not running")

func Discover(dir string) (Metadata, error) {
	var m Metadata
	b, e := os.ReadFile(filepath.Join(dir, "runtime.json"))
	if errors.Is(e, os.ErrNotExist) {
		return m, ErrNotRunning
	}
	if e != nil {
		return m, e
	}
	if e = json.Unmarshal(b, &m); e != nil {
		return m, e
	}
	u, err := url.Parse(m.URL)
	if err != nil || u.User != nil || u.Hostname() != "127.0.0.1" || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return m, errors.New("invalid runtime address")
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port < 1 || port > 65535 {
		return m, errors.New("invalid runtime port")
	}
	if m.Version != Protocol || !strings.HasPrefix(m.URL, "http://127.0.0.1:") || len(m.Token) != 64 {
		return m, fmt.Errorf("incompatible runtime metadata")
	}
	return m, nil
}

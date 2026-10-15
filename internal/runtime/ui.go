package runtime

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/iagoluch/OpenVibeCoding/internal/automation"
	"github.com/iagoluch/OpenVibeCoding/internal/config"
	"github.com/iagoluch/OpenVibeCoding/internal/setup"
	"github.com/iagoluch/OpenVibeCoding/internal/webui"
)

// pairTTL bounds how long a pairing link stays usable. It is a variable so the
// tests can shorten it.
var pairTTL = 5 * time.Minute

// pairRequest is the optional body of POST /pair.
type pairRequest struct {
	Workspace string `json:"workspace"`
}

// UIInfo is everything the browser needs to render its first screen. It never
// carries a credential: only whether one is stored.
type UIInfo struct {
	Configured  bool                `json:"configured"`
	Active      UIActive            `json:"active"`
	Permissions config.Permissions  `json:"permissions"`
	Automation  automation.Settings `json:"automation"`
	Workspace   string              `json:"workspace"`
	Workspaces  []string            `json:"workspaces"`
	Providers   []UIProvider        `json:"providers"`
}
type UIActive struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Harness  string `json:"harness"`
}
type UIProvider struct {
	ID     string    `json:"id"`
	Title  string    `json:"title"`
	Detail string    `json:"detail"`
	KeyURL string    `json:"key_url"`
	Free   bool      `json:"free"`
	Local  bool      `json:"local"`
	HasKey bool      `json:"has_key"`
	Active bool      `json:"active"`
	Models []UIModel `json:"models"`
}
type UIModel struct {
	ID      string `json:"id"`
	Free    bool   `json:"free"`
	Context int    `json:"context"`
}

// UIAction is the outcome of one onboarding mutation.
type UIAction struct {
	OK       bool   `json:"ok"`
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
	Error    string `json:"error,omitempty"`
}

// serveUI answers browser reads: the page and its assets, and the pairing link.
// Assets carry no authority, so they need no cookie; the page itself asks
// /ui/info, which is authenticated.
func (s *Server) serveUI(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/pair":
		s.pairAccept(w, r)
	case r.URL.Path == "/" || strings.HasPrefix(r.URL.Path, "/assets/"):
		s.ui.ServeHTTP(w, r)
	case postOnly[r.URL.Path]:
		http.Error(w, "POST required", 405)
	default:
		http.NotFound(w, r)
	}
}

// postOnly lists the paths that accept requests only: a browser GET on them is
// a method error, not a missing page.
var postOnly = map[string]bool{
	"/health": true, "/stop": true, "/approve": true, "/store": true, "/run": true,
	"/ui/info": true, "/ui/setup": true, "/ui/select": true,
}

// pairCreate mints one single-use code the browser exchanges for the runtime
// cookie. Only a bearer client (the CLI) may create one.
func (s *Server) pairCreate(w http.ResponseWriter, r *http.Request) {
	var body pairRequest
	if e := json.NewDecoder(r.Body).Decode(&body); e != nil && !errors.Is(e, io.EOF) {
		http.Error(w, "invalid pairing request", 400)
		return
	}
	if ws := strings.TrimSpace(body.Workspace); ws != "" {
		if !filepath.IsAbs(ws) {
			http.Error(w, "workspace must be an absolute path", 400)
			return
		}
		s.mu.Lock()
		s.workspace = ws
		s.mu.Unlock()
	}
	code := random()
	expires := time.Now().Add(pairTTL)
	s.mu.Lock()
	for candidate, expiry := range s.pairings {
		if time.Now().After(expiry) {
			delete(s.pairings, candidate)
		}
	}
	s.pairings[code] = expires
	s.mu.Unlock()
	s.log.Info("pairing_created")
	json.NewEncoder(w).Encode(map[string]string{"code": code, "expires": expires.UTC().Format(time.RFC3339)})
}

// pairAccept trades a single-use code for the browser cookie. The cookie value
// is the runtime token, so it carries the same authority as the bearer header
// and only ever reaches this origin (HttpOnly, SameSite=Strict, loopback HTTP).
func (s *Server) pairAccept(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("token")
	paired := false
	if code != "" {
		s.mu.Lock()
		if expires, ok := s.pairings[code]; ok {
			delete(s.pairings, code)
			paired = time.Now().Before(expires)
		}
		s.mu.Unlock()
	}
	webui.Secure(w.Header())
	if !paired {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusGone)
		w.Write(webui.PairInvalid())
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "ovc_ui", Value: s.token, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode})
	s.log.Info("pairing_accepted")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// uiInfo describes the daemon to the paired page: the active configuration, the
// workspace a new session should use, and the providers a beginner may connect.
func (s *Server) uiInfo(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	c := s.config
	workspace := s.workspace
	s.mu.Unlock()
	info := UIInfo{
		Active:      UIActive{Provider: c.Provider, Model: c.Model, Harness: c.Harness},
		Permissions: c.Permissions,
		Automation:  c.Automation,
		Workspace:   workspace,
		Workspaces:  []string{},
		Providers:   uiProviders(c, setup.Detect()),
	}
	if p, ok := c.Providers[c.Provider]; ok {
		info.Configured = p.Secret != "" || loopback(p.BaseURL)
	}
	if sessions, e := s.db.List(); e == nil {
		seen := map[string]bool{}
		for _, session := range sessions {
			if len(info.Workspaces) >= 10 {
				break
			}
			if !seen[session.Workspace] {
				seen[session.Workspace] = true
				info.Workspaces = append(info.Workspaces, session.Workspace)
			}
		}
	}
	json.NewEncoder(w).Encode(info)
}

// uiSetup connects one provider for the browser: store an optional credential,
// activate the same way the wizard does, and keep the change only when a model
// answered a real request. force trusts the choice without a smoke test.
func (s *Server) uiSetup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider, Model, Key string
		Force                bool
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		http.Error(w, "invalid setup request", 400)
		return
	}
	s.mu.Lock()
	c := clone(s.config)
	s.mu.Unlock()
	name, model := strings.TrimSpace(req.Provider), strings.TrimSpace(req.Model)
	if _, ok := c.Providers[name]; !ok {
		s.uiFailure(w, c, errors.New("unknown provider"))
		return
	}
	if key := strings.TrimSpace(req.Key); key != "" {
		if e := setup.Store(s.dir, &c, name, key); e != nil {
			s.uiFailure(w, c, e)
			return
		}
	}
	p := c.Providers[name]
	if model != "" && setup.LocalProvider(name) {
		// A locally served model may not be in the shipped preset yet.
		p = setup.EnsureModel(p, model, 32768)
		c.Providers[name] = p
	}
	if model == "" && len(p.Models) > 0 {
		model = p.Models[0].ID
	}
	active := model
	if !req.Force {
		var e error
		if active, e = setup.Probe(r.Context(), p, model, nil); e != nil {
			s.uiFailure(w, c, e)
			return
		}
	}
	updated, e := setup.Select(s.dir, c, name, active)
	if e != nil {
		s.uiFailure(w, c, e)
		return
	}
	s.mu.Lock()
	s.config = updated
	s.mu.Unlock()
	s.log.Info("ui_provider_activated", "provider", name, "model", active)
	json.NewEncoder(w).Encode(UIAction{OK: true, Provider: name, Model: active})
}

// uiSelect activates a provider/model pair that is already known to work.
func (s *Server) uiSelect(w http.ResponseWriter, r *http.Request) {
	var req struct{ Provider, Model string }
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		http.Error(w, "invalid selection", 400)
		return
	}
	s.mu.Lock()
	c := clone(s.config)
	s.mu.Unlock()
	name, model := strings.TrimSpace(req.Provider), strings.TrimSpace(req.Model)
	updated, e := setup.Select(s.dir, c, name, model)
	if e != nil {
		s.uiFailure(w, c, e)
		return
	}
	s.mu.Lock()
	s.config = updated
	s.mu.Unlock()
	json.NewEncoder(w).Encode(UIAction{OK: true, Provider: name, Model: model})
}

// clone returns a configuration the caller may mutate. Config carries a map
// and slices, so a plain copy would still write into the daemon's live
// configuration while other requests read it.
func clone(c config.Config) config.Config {
	providers := make(map[string]config.Provider, len(c.Providers))
	for name, p := range c.Providers {
		p.Models = append([]config.Model(nil), p.Models...)
		providers[name] = p
	}
	c.Providers = providers
	return c
}

// uiFailure reports a redacted reason and leaves the active configuration
// untouched; the stored credential survives so the user can pick another model.
func (s *Server) uiFailure(w http.ResponseWriter, c config.Config, err error) {
	s.log.Info("ui_setup_failed")
	json.NewEncoder(w).Encode(UIAction{Error: c.Redact(err.Error())})
}

// uiProviders lists providers in beginner order: local runners that answer
// right now, then the free-tier catalog, then everything else by name.
func uiProviders(c config.Config, detected setup.Local) []UIProvider {
	var out []UIProvider
	for _, name := range providerOrder(c, detected) {
		p := c.Providers[name]
		entry := UIProvider{ID: name, Title: name, Local: loopback(p.BaseURL), HasKey: p.Secret != "", Active: name == c.Provider}
		for _, meta := range setup.Providers() {
			if meta.ID == name {
				entry.Title, entry.Detail, entry.KeyURL = meta.Title, meta.Detail, meta.KeyURL
			}
		}
		models := p.Models
		if setup.LocalProvider(name) {
			models = localModels(p, detected, name)
		}
		for _, m := range models {
			free := entry.Local || (m.InputPrice != nil && *m.InputPrice == 0)
			entry.Models = append(entry.Models, UIModel{ID: m.ID, Free: free, Context: m.Context})
			entry.Free = entry.Free || free
		}
		out = append(out, entry)
	}
	return out
}

// localModels lists what a local runner serves right now first, then the
// preset's models that are not running, so the page reflects the machine.
func localModels(p config.Provider, detected setup.Local, name string) []config.Model {
	served := detected.Ollama
	if name == "lmstudio" {
		served = detected.LMStudio
	}
	zero := 0.0
	var models []config.Model
	seen := map[string]bool{}
	for _, id := range served {
		if seen[id] {
			continue
		}
		seen[id] = true
		m := config.Model{ID: id, Context: 32768, Tools: true, InputPrice: &zero, OutputPrice: &zero}
		for _, known := range p.Models {
			if known.ID == id {
				m = known
			}
		}
		models = append(models, m)
	}
	for _, known := range p.Models {
		if !seen[known.ID] {
			models = append(models, known)
		}
	}
	return models
}

func providerOrder(c config.Config, detected setup.Local) []string {
	added := map[string]bool{}
	var order []string
	add := func(name string) {
		if _, ok := c.Providers[name]; ok && !added[name] {
			added[name] = true
			order = append(order, name)
		}
	}
	if len(detected.Ollama) > 0 {
		add("local")
	}
	if len(detected.LMStudio) > 0 {
		add("lmstudio")
	}
	for _, meta := range setup.Providers() {
		add(meta.ID)
	}
	var rest []string
	for name := range c.Providers {
		if !added[name] {
			rest = append(rest, name)
		}
	}
	sort.Strings(rest)
	return append(order, rest...)
}

// loopback reports whether a provider is served by a local runner, which needs
// no credential.
func loopback(raw string) bool {
	u, e := url.Parse(raw)
	if e != nil {
		return false
	}
	switch u.Hostname() {
	case "127.0.0.1", "localhost", "::1":
		return true
	}
	return false
}

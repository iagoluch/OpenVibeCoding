package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/iagoluch/OpenVibeCoding/internal/config"
	"github.com/iagoluch/OpenVibeCoding/internal/workspace"
)

// browser mirrors what the paired page does: same host, explicit headers,
// redirects returned instead of followed and no implicit cookie jar.
var browser = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

func ask(t *testing.T, c *Client, method, path string, headers map[string]string, body string) *http.Response {
	t.Helper()
	req, e := http.NewRequest(method, c.Meta.URL+path, strings.NewReader(body))
	if e != nil {
		t.Fatal(e)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, e := browser.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

// pairBrowser consumes a pairing link the way a browser navigation does.
func pairBrowser(t *testing.T, c *Client, link string) (*http.Response, string) {
	t.Helper()
	path := strings.TrimPrefix(link, c.Meta.URL)
	resp := ask(t, c, "GET", path, map[string]string{"Sec-Fetch-Site": "none"}, "")
	cookie := ""
	for _, c := range resp.Cookies() {
		if c.Name == "ovc_ui" {
			cookie = c.Value
		}
	}
	return resp, cookie
}

func TestRuntimeRejectsForeignHost(t *testing.T) {
	_, c := startFixture(t, config.Default())
	host := strings.TrimPrefix(c.Meta.URL, "http://")
	for _, foreign := range []string{"rebind.example", "localhost:" + strings.Split(host, ":")[1], "127.0.0.1"} {
		req, e := http.NewRequest("POST", c.Meta.URL+"/health", strings.NewReader("{}"))
		if e != nil {
			t.Fatal(e)
		}
		req.Host = foreign
		req.Header.Set("Authorization", "Bearer "+c.Meta.Token)
		req.Header.Set("X-OpenVibeCoding-Client", "test")
		resp, e := browser.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		resp.Body.Close()
		if resp.StatusCode != 403 {
			t.Fatalf("host %q accepted with %d", foreign, resp.StatusCode)
		}
	}
}

func TestRuntimePairingIsSingleUseAndExpires(t *testing.T) {
	_, c := startFixture(t, config.Default())
	link, e := c.Pair(context.Background(), t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	if !strings.HasPrefix(link, c.Meta.URL+"/pair?token=") {
		t.Fatalf("unexpected link %q", link)
	}
	resp, cookie := pairBrowser(t, c, link)
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/" || cookie == "" {
		t.Fatalf("pairing rejected: %d %q %q", resp.StatusCode, resp.Header.Get("Location"), cookie)
	}
	if again := ask(t, c, "GET", strings.TrimPrefix(link, c.Meta.URL), nil, ""); again.StatusCode != http.StatusGone {
		t.Fatalf("single-use code reused: %d", again.StatusCode)
	}
	if unknown := ask(t, c, "GET", "/pair?token=deadbeef", nil, ""); unknown.StatusCode != http.StatusGone {
		t.Fatalf("unknown code accepted: %d", unknown.StatusCode)
	}
	b, _ := io.ReadAll(ask(t, c, "GET", "/pair?token=deadbeef", nil, "").Body)
	if !strings.Contains(string(b), "ovc --ui") {
		t.Fatalf("pairing failure page is not actionable: %q", string(b))
	}
	restore := pairTTL
	pairTTL = -time.Second
	defer func() { pairTTL = restore }()
	stale, e := c.Pair(context.Background(), "")
	if e != nil {
		t.Fatal(e)
	}
	if resp, _ := pairBrowser(t, c, stale); resp.StatusCode != http.StatusGone {
		t.Fatalf("expired code accepted: %d", resp.StatusCode)
	}
}

func TestRuntimeCookieNeedsOwnOrigin(t *testing.T) {
	_, c := startFixture(t, config.Default())
	link, e := c.Pair(context.Background(), t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	resp, cookie := pairBrowser(t, c, link)
	if resp.StatusCode != http.StatusSeeOther || cookie == "" {
		t.Fatalf("pairing failed: %d", resp.StatusCode)
	}
	own := map[string]string{"Cookie": "ovc_ui=" + cookie, "X-OpenVibeCoding-Client": "tab-1", "Origin": c.Meta.URL, "Sec-Fetch-Site": "same-origin"}
	if got := ask(t, c, "POST", "/ui/info", own, "{}"); got.StatusCode != 200 {
		t.Fatalf("same-origin cookie rejected: %d", got.StatusCode)
	}
	cases := []struct {
		name    string
		headers map[string]string
	}{
		{"cross-origin page", map[string]string{"Cookie": "ovc_ui=" + cookie, "X-OpenVibeCoding-Client": "tab-1", "Origin": "http://evil.example"}},
		{"cross-site fetch metadata", map[string]string{"Cookie": "ovc_ui=" + cookie, "X-OpenVibeCoding-Client": "tab-1", "Origin": c.Meta.URL, "Sec-Fetch-Site": "cross-site"}},
		{"foreign cookie", map[string]string{"Cookie": "ovc_ui=forged", "X-OpenVibeCoding-Client": "tab-1", "Origin": c.Meta.URL}},
		{"no credential at all", map[string]string{"X-OpenVibeCoding-Client": "tab-1", "Origin": c.Meta.URL}},
		{"bearer from a foreign page", map[string]string{"Authorization": "Bearer " + c.Meta.Token, "X-OpenVibeCoding-Client": "tab-1", "Origin": "http://evil.example"}},
	}
	for _, tc := range cases {
		if got := ask(t, c, "POST", "/ui/info", tc.headers, "{}"); got.StatusCode != 401 {
			t.Fatalf("%s accepted: %d", tc.name, got.StatusCode)
		}
	}
	// A cookie is a browser credential: it may use the API, never mint links.
	mint := map[string]string{"Cookie": "ovc_ui=" + cookie, "X-OpenVibeCoding-Client": "tab-1", "Origin": c.Meta.URL, "Sec-Fetch-Site": "same-origin"}
	if got := ask(t, c, "POST", "/pair", mint, "{}"); got.StatusCode != 401 {
		t.Fatalf("cookie minted a pairing link: %d", got.StatusCode)
	}
	// The bearer path keeps working for same-origin and headless clients.
	bearer := map[string]string{"Authorization": "Bearer " + c.Meta.Token, "X-OpenVibeCoding-Client": "tui", "Origin": c.Meta.URL}
	if got := ask(t, c, "POST", "/ui/info", bearer, "{}"); got.StatusCode != 200 {
		t.Fatalf("bearer with own origin rejected: %d", got.StatusCode)
	}
}

func TestRuntimeUIRoutesAndHeaders(t *testing.T) {
	_, c := startFixture(t, config.Default())
	for _, path := range []string{"/", "/assets/app.js", "/assets/style.css"} {
		resp := ask(t, c, "GET", path, nil, "")
		if resp.StatusCode != 200 {
			t.Fatalf("%s: %d", path, resp.StatusCode)
		}
		for header, want := range map[string]string{
			"Cache-Control":          "no-cache",
			"X-Content-Type-Options": "nosniff",
			"Referrer-Policy":        "no-referrer",
		} {
			if got := resp.Header.Get(header); got != want {
				t.Fatalf("%s: %s = %q", path, header, got)
			}
		}
		if got := resp.Header.Get("Content-Security-Policy"); !strings.Contains(got, "default-src 'self'") || !strings.Contains(got, "frame-ancestors 'none'") {
			t.Fatalf("%s: CSP = %q", path, got)
		}
		if resp.Header.Get("Access-Control-Allow-Origin") != "" {
			t.Fatalf("%s: CORS header served", path)
		}
	}
	for _, path := range []string{"/assets/missing.js", "/assets/", "/missing"} {
		if got := ask(t, c, "GET", path, nil, ""); got.StatusCode != 404 {
			t.Fatalf("%s: %d", path, got.StatusCode)
		}
	}
	if got := ask(t, c, "GET", "/health", nil, ""); got.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("health served on GET: %d", got.StatusCode)
	}
	bearer := map[string]string{"Authorization": "Bearer " + c.Meta.Token, "X-OpenVibeCoding-Client": "tui"}
	if got := ask(t, c, "POST", "/", bearer, "{}"); got.StatusCode != 404 {
		t.Fatalf("page served on POST: %d", got.StatusCode)
	}
}

func TestRuntimeUIInfoDescribesProvidersAndWorkspaces(t *testing.T) {
	root, e := workspace.Open(t.TempDir(), false)
	if e != nil {
		t.Fatal(e)
	}
	defer root.Close()
	ws := root.Path
	_, c := startFixture(t, config.Default())
	if _, e := c.New(ws); e != nil {
		t.Fatal(e)
	}
	if _, e := c.New(ws); e != nil {
		t.Fatal(e)
	}
	info, e := c.UIInfo(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	if len(info.Workspaces) != 1 || info.Workspaces[0] != ws {
		t.Fatalf("workspaces not deduplicated: %v", info.Workspaces)
	}
	if info.Configured {
		t.Fatal("a provider without credential or local endpoint must not read as configured")
	}
	if len(info.Providers) == 0 || info.Providers[0].ID != "openrouter" {
		t.Fatalf("provider order is not beginner first: %v", info.Providers)
	}
	active := 0
	for _, p := range info.Providers {
		if p.Active {
			active++
		}
		if p.ID == "local" && !p.Local {
			t.Fatal("local endpoint not marked local")
		}
	}
	if active != 1 {
		t.Fatalf("%d active providers", active)
	}
}

func TestRuntimeUISetupUsesWizardActivation(t *testing.T) {
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "retired-model") {
			w.WriteHeader(503)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"OK\"}}]}\n\ndata: [DONE]\n\n")
	}))
	defer fake.Close()
	cfg := config.Default()
	cfg.Providers["probe"] = config.Provider{BaseURL: fake.URL, Secret: "in-memory", Models: []config.Model{
		{ID: "retired-model", Context: 32768, Tools: true},
		{ID: "healthy-model", Context: 32768, Tools: true},
	}}
	dir, c := startFixture(t, cfg)
	action, e := c.UISetup(context.Background(), "probe", "retired-model", "", false)
	if e != nil || !action.OK || action.Model != "healthy-model" {
		t.Fatalf("setup did not fall back: %+v %v", action, e)
	}
	saved := readSavedConfig(t, dir)
	if saved.Provider != "probe" || saved.Model != "healthy-model" {
		t.Fatalf("not persisted: %s/%s", saved.Provider, saved.Model)
	}
	info, e := c.UIInfo(context.Background())
	if e != nil || info.Active.Provider != "probe" || info.Active.Model != "healthy-model" || !info.Configured {
		t.Fatalf("daemon did not adopt the new configuration: %+v %v", info.Active, e)
	}
}

func TestRuntimeUISetupFailureKeepsConfigurationAndForcesOnRequest(t *testing.T) {
	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	defer down.Close()
	cfg := config.Default()
	cfg.Providers["probe"] = config.Provider{BaseURL: down.URL, Models: []config.Model{{ID: "only-model", Context: 32768, Tools: true}}}
	dir, c := startFixture(t, cfg)
	before := readSavedConfig(t, dir)
	if _, e := c.UISetup(context.Background(), "probe", "", "", false); e == nil {
		t.Fatal("unreachable model accepted")
	}
	after := readSavedConfig(t, dir)
	if before.Provider != after.Provider || before.Model != after.Model {
		t.Fatalf("failed setup changed the active configuration: %s/%s -> %s/%s", before.Provider, before.Model, after.Provider, after.Model)
	}
	action, e := c.UISetup(context.Background(), "probe", "only-model", "", true)
	if e != nil || !action.OK || action.Model != "only-model" {
		t.Fatalf("forced activation failed: %+v %v", action, e)
	}
	if _, e := c.UISetup(context.Background(), "missing", "", "", true); e == nil {
		t.Fatal("unknown provider accepted")
	}
}

func TestDiscoverReportsNotRunning(t *testing.T) {
	if _, e := Discover(t.TempDir()); !errors.Is(e, ErrNotRunning) {
		t.Fatalf("missing metadata must map to ErrNotRunning, got %v", e)
	}
	if _, e := Connect(t.TempDir()); !errors.Is(e, ErrNotRunning) {
		t.Fatalf("connect must report the sentinel, got %v", e)
	}
}

func readSavedConfig(t *testing.T, dir string) config.Config {
	t.Helper()
	var c config.Config
	b, e := os.ReadFile(filepath.Join(dir, "config.json"))
	if errors.Is(e, os.ErrNotExist) {
		return config.Default()
	}
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(b, &c); e != nil {
		t.Fatal(e)
	}
	return c
}
